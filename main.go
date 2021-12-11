// Package nccald is a simple daemon to provide calendar notifications for
// Namecoin name expirations.
package main

import (
	"github.com/hlandau/dexlogconfig"
	"github.com/hlandau/nccald/server"
	"github.com/hlandau/xlog"
	"gopkg.in/hlandau/easyconfig.v1"
	"gopkg.in/hlandau/service.v2"
)

var log, Log = xlog.New("nccald")

func main() {
	cfg := &server.Config{}

	config := easyconfig.Configurator{
		ProgramName: "nccald",
	}
	config.ParseFatal(cfg)
	dexlogconfig.Init()

	if cfg.Once {
		server.Once(cfg)
		return
	}

	service.Main(&service.Info{
		Name:          "nccald",
		Description:   "Namecoin calendar daemon",
		DefaultChroot: service.EmptyChrootPath,
		NewFunc: func() (service.Runnable, error) {
			return server.New(cfg)
		},
	})
}
