// Package icsutil contains utilities for generating ICS files based on
// Namecoin name expiration.
package icsutil

import (
	"fmt"
	"github.com/arran4/golang-ical"
	"github.com/hlandau/nccald/types"
	"github.com/namecoin/ncbtcjson"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// GenerateICS generates a string containing an ICS calendar file. It contains
// a single VCALENDAR which contains zero or more VEVENTs, each corresponding
// to the estimated expiry time of the names passed in.
//
// showInfo and extraInfo are slices of name information. The information for
// a name showInfo[i] must correspond to the information in extraInfo[i] for a
// given value of i. The number of events generated is equal to the length of
// showInfo. showInfo can contain a single item if you want to generate a calendar
// file containing only a single event (e.g. for CalDAV).
//
// Precondition: The length of showInfo and extraInfo must be equal.
func GenerateICS(now time.Time, showInfo []ncbtcjson.NameShowResult, extraInfo []types.ExtraNameInfo) (string, error) {
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetProductId("nccald")
	cal.SetName("nccald calendar")
	cal.SetLastModified(now)
	cal.SetDescription("")

	for i := range showInfo {
		n := &showInfo[i]

		earliestExpectedExpiryTime := extraInfo[i].EstimatedExpiryTime

		name := EncodeName(n.Name)
		e := cal.AddEvent(fmt.Sprintf("%s@nccald", name))
		e.SetCreatedTime(now)
		e.SetDtStampTime(now)
		e.SetStartAt(earliestExpectedExpiryTime)
		e.SetEndAt(earliestExpectedExpiryTime)
		e.SetSummary(fmt.Sprintf("Expiry of Namecoin name %q (%v)", name, extraInfo[i].ExpiryHeight))
		e.SetDescription(fmt.Sprintf("Namecoin name %q is estimated to expire around this time (expires at height (%v)", name, extraInfo[i].ExpiryHeight))
		e.SetOrganizer("nccald@namecoin.org")
	}

	return cal.Serialize(), nil
}

// Write generates an ICS calendar file and writes it to the given filename.
// The write is performed by writing to a temporary file followed by a rename
// so it is atomic. For usage of the arguments showInfo and extraInfo, see
// GenerateICS.
func Write(now time.Time, filename string, showInfo ncbtcjson.NameListResult, extraInfo []types.ExtraNameInfo) error {
	data, err := GenerateICS(now, showInfo, extraInfo)
	if err != nil {
		return err
	}

	// Write to a temporary file and rename it over the destination filename.
	// This ensures updates are atomic, which is useful if e.g. anything is
	// listening using inotify etc. for changes and might read it the instant we
	// start writing it.
	tmpFilename := filename + ".tmp"
	err = ioutil.WriteFile(tmpFilename, []byte(data), 0644)
	if err != nil {
		return fmt.Errorf("error while writing ICS file: %v", err)
	}

	err = os.Rename(tmpFilename, filename)
	if err != nil {
		return fmt.Errorf("error while renaming ICS file: %v", err)
	}

	return nil
}

// Munges a Namecoin name to something safe to use in URLs and paths.
func EncodeName(name string) string {
	var b strings.Builder

	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r) // cannot fail
		} else {
			b.WriteString(fmt.Sprintf("_%02x", r))
		}
	}

	return b.String()
}
