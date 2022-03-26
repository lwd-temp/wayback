// Copyright 2022 Wayback Archiver. All rights reserved.
// Use of this source code is governed by the GNU GPL v3
// license that can be found in the LICENSE file.

package planner // import "github.com/wabarc/wayback/planner"

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	"github.com/wabarc/logger"
)

// Shuttle represents a struct which stores tickers and other data.
type Shuttle struct {
	mutex   sync.Mutex
	tickers []*time.Ticker
	home    string
}

// New returns a Shuttle.
func New() *Shuttle {
	s := &Shuttle{
		mutex: sync.Mutex{},
	}
	s.home, _ = ioutil.TempDir(os.TempDir(), "planner-")

	return s
}

// Start starts scheduling services. It is the caller's responsibility to close tickers.
func (s *Shuttle) Start(ctx context.Context) *Shuttle {
	tArchiveis := time.NewTicker(5 * time.Minute) // ticker for archive.is
	s.tickers = []*time.Ticker{tArchiveis}

	go func() {
		wd := path.Join(s.home, "starter")
		if err := os.MkdirAll(wd, 0o600); err != nil {
			logger.Error("create starter directory failed: %v", err)
		}
		today := today{
			userDataDir: path.Join(wd, "UserDataDir"),
			workspace:   wd,
		}
		go today.init()

		for {
			select {
			default:
			case <-tArchiveis.C:
				rctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
				defer cancel()
				if err := today.run(rctx); err != nil {
					logger.Error("regularly update the 'ARCHIVE_COOKIE' environment failed: %v", err)
				}
			}
		}
	}()

	return s
}

// Stop stop scheduling services.
func (s *Shuttle) Stop() {
	for _, ticker := range s.tickers {
		s.mutex.Lock()
		ticker.Stop()
		s.mutex.Unlock()
	}
	os.RemoveAll(s.home)
}
