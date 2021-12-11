package server

import (
	"context"
	"github.com/hlandau/nccald/caldavout"
	"github.com/hlandau/nccald/icsutil"
	"github.com/hlandau/nccald/types"
	"github.com/hlandau/xlog"
	"github.com/namecoin/btcd/rpcclient"
	"github.com/namecoin/ncbtcjson"
	"github.com/namecoin/ncrpcclient"
	"time"
)

var log, Log = xlog.New("server")

// Configuration for the daemon.
type Config struct {
	NamecoinRPCUsername   string `default:"" usage:"Namecoin RPC username"`
	NamecoinRPCPassword   string `default:"" usage:"Namecoin RPC password"`
	NamecoinRPCAddress    string `default:"127.0.0.1:8336" usage:"Namecoin RPC server address"`
	NamecoinRPCCookiePath string `default:"" usage:"Namecoin RPC cookie path (used if password is unspecified)"`
	NamecoinRPCTimeout    int    `default:"1500" usage:"Timeout (in milliseconds) for Namecoin RPC requests"`

	CalMargin        time.Duration `default:"72h" usage:"Number of days uncertainty to allow in projected expiration calculation"`
	CalQuantum       time.Duration `default:"72h" usage:"Projected expiration day is rounded to a multiple of this value"`
	CalQueryInterval time.Duration `default:"10m" usage:"Query interval"`
	ICSPath          string        `default:"" usage:"Write ICS calendar file to this path if specified"`
	CalDavURL        string        `default:"" usage:"Updates CalDAV resource if desired"`
	CalDavUsername   string        `default:"" usage:"Username to use to authenticate to CalDAV resource"`
	CalDavPassword   string        `default:"" usage:"Password to use to authenticate to CalDAV resource"`
	Once             bool          `default:"" usage:"Write ICS file/update CalDAV resource once and exit"`
}

// Main object for this daemon.
type Server struct {
	cfg       Config
	closeChan chan struct{}
	client    *ncrpcclient.Client
	cdCfg     *caldavout.Config

	firstPollDoneChan chan struct{}
}

// needing update: github.com/namecoin/ncbtcjson
// needing update: github.com/namecoin/ncrpcclient

func New(cfg *Config) (*Server, error) {
	if cfg.CalQueryInterval < 1*time.Second {
		log.Noticef("using query interval of 10 minutes")
		cfg.CalQueryInterval = 10 * time.Minute
	}

	// Connect to local namecoin core RPC server using HTTP POST mode.
	connCfg := &rpcclient.ConnConfig{
		Host:         cfg.NamecoinRPCAddress,
		User:         cfg.NamecoinRPCUsername,
		Pass:         cfg.NamecoinRPCPassword,
		CookiePath:   cfg.NamecoinRPCCookiePath,
		HTTPPostMode: true, // Namecoin core only supports HTTP POST mode
		DisableTLS:   true, // Namecoin core does not provide TLS by default
	}

	// Notice the notification parameter is nil since notifications are not
	// supported in HTTP POST mode.
	client, err := ncrpcclient.New(connCfg, nil)
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:               *cfg,
		client:            client,
		closeChan:         make(chan struct{}),
		firstPollDoneChan: make(chan struct{}),
	}

	if s.cfg.CalDavURL != "" {
		s.cdCfg = &caldavout.Config{
			User: s.cfg.CalDavUsername,
			Pass: s.cfg.CalDavPassword,
		}
	}

	log.Debugf("server instantiated")
	return s, nil
}

func (s *Server) Start() error {
	log.Debugf("starting server...")
	go s.pollLoop()
	return nil
}

func (s *Server) Stop() error {
	log.Debugf("processing request to stop server...")
	close(s.closeChan)
	return nil
}

func (s *Server) pollLoop() {
	log.Debugf("performing initial poll...")
	s.poll()
	close(s.firstPollDoneChan)

	ticker := time.NewTicker(s.cfg.CalQueryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Debugf("poll interval expired, polling...")
			s.poll()

		case <-s.closeChan:
			log.Debugf("shutting down poll loop")
			return
		}
	}
}

func (s *Server) poll() {
	log.Debug("polling: retrieving names...")
	r, err := s.getNames()
	if err != nil {
		log.Errore(err, "could not list names")
		return
	}

	log.Debug("polling: checking names...")
	err = s.checkNames(r)
	log.Errore(err, "error when checking names")

	log.Debug("polling completed")
}

func (s *Server) getNames() (ncbtcjson.NameListResult, error) {
	return s.client.NameList("")
}

const expiryBlocks = 36000
const timePerBlock = 10 * time.Minute

func (s *Server) checkNames(r ncbtcjson.NameListResult) error {
	if s.cfg.ICSPath == "" && s.cfg.CalDavURL == "" {
		log.Warn("neither ICS path nor CalDAV URL configured, nothing to do")
		return nil
	}

	now := time.Now()
	extraInfo := s.computeExtraInfo(now, r)

	if s.cfg.ICSPath != "" {
		log.Debugf("updating ICS file %q...", s.cfg.ICSPath)
		err := icsutil.Write(now, s.cfg.ICSPath, r, extraInfo)
		log.Errore(err, "could not update ICS file")
	}

	if s.cfg.CalDavURL != "" {
		log.Debugf("updating CalDAV server %q...", s.cfg.CalDavURL)
		err := caldavout.Put(context.TODO(), now, s.cfg.CalDavURL, s.cdCfg, r, extraInfo)
		log.Errore(err, "could not update CalDAV resources")
	}

	return nil
}

func (s *Server) computeExtraInfo(now time.Time, r ncbtcjson.NameListResult) (extraInfo []types.ExtraNameInfo) {
	var curBlockHeight int32

	for i := range r {
		n := &r[i]

		if curBlockHeight == 0 {
			// Infer current block height from name_list results so we don't have to
			// make another RPC request to get it.
			curBlockHeight = n.Height + expiryBlocks - n.ExpiresIn
		}

		expiryHeight := n.Height + expiryBlocks
		var earliestExpectedExpiryDate time.Time
		if n.Expired {
			earliestExpectedExpiryDate = now
		} else {
			earliestExpectedExpiryDate = s.estimateExpiry(curBlockHeight, expiryHeight, now)
		}

		e := types.ExtraNameInfo{
			EstimatedExpiryTime: earliestExpectedExpiryDate,
			ExpiryHeight:        expiryHeight,
		}

		extraInfo = append(extraInfo, e)
	}

	return
}

func (s *Server) estimateExpiry(curHeight, expiryHeight int32, now time.Time) time.Time {
	if expiryHeight <= curHeight {
		return now
	}

	blocksToGo := expiryHeight - curHeight
	estimatedTimeToGo := time.Duration(blocksToGo) * timePerBlock

	return now.Add(estimatedTimeToGo).Add(-s.cfg.CalMargin).Truncate(s.cfg.CalQuantum)
}

// Generates ICS files and/or updates CalDAV resources one time only.
func Once(cfg *Config) error {
	log.Debug("running one time only...")

	s, err := New(cfg)
	if err != nil {
		return err
	}

	err = s.Start()
	if err != nil {
		return err
	}

	log.Debug("waiting for first poll to be completed...")
	<-s.firstPollDoneChan

	log.Debug("stopping...")
	err = s.Stop()
	if err != nil {
		return err
	}

	return nil
}
