package main

import (
	"os"
	"path/filepath"

	"github.com/masonhuemmer/m365/internal/adapters/chatmap"
	"github.com/masonhuemmer/m365/internal/adapters/cli"
	"github.com/masonhuemmer/m365/internal/adapters/fs"
	"github.com/masonhuemmer/m365/internal/adapters/graph"
	"github.com/masonhuemmer/m365/internal/adapters/jev"
	"github.com/masonhuemmer/m365/internal/adapters/keychain"
	"github.com/masonhuemmer/m365/internal/adapters/mailclassifier"
	"github.com/masonhuemmer/m365/internal/adapters/mailwatchstate"
	"github.com/masonhuemmer/m365/internal/adapters/watchstate"
	"github.com/masonhuemmer/m365/internal/config"
)

func main() {
	cfg, cfgErr := config.Load()
	os.Exit(cli.Run(os.Args, buildDeps(cfg, cfgErr, os.Getenv)))
}

func buildDeps(cfg config.Config, cfgErr error, getenv func(string) string) cli.Deps {
	if getenv == nil {
		getenv = os.Getenv
	}
	d := cli.Deps{
		Config:         cfg,
		ConfigError:    cfgErr,
		Write:          fs.WriteFile,
		Watch:          &watchstate.File{},
		MailWatchState: &mailwatchstate.File{},
		ChatMap:        &chatmap.File{},
	}
	if getenv("M365_FAKE") == "1" {
		mem := graph.Seed()
		mailDelta := graph.MailDeltaAPI{Memory: mem}
		d.Mail = graph.MailAPI{Memory: mem}
		d.MailChanges = mailDelta
		d.MailThreads = mailDelta
		d.MailClassifier = &mailclassifier.Fake{}
		d.Teams = graph.TeamsAPI{Memory: mem}
		d.Calendar = graph.CalendarAPI{Memory: mem}
		d.Files = graph.FilesAPI{Memory: mem}
		d.Login = graph.FakeLoginAll(true, true, true, true)
		d.DeviceLogin = d.Login
		home, _ := os.UserHomeDir()
		state := os.Getenv("XDG_STATE_HOME")
		if state == "" {
			state = filepath.Join(home, ".local", "state")
		}
		d.Store = &keychain.FileStore{Path: filepath.Join(state, "m365", "fake-session.json")}
	} else {
		store := &keychain.Fallback{
			Primary:   keychain.Keyring{},
			Secondary: &keychain.FileStore{Path: keychain.LiveSessionPath()},
		}
		ref := &graph.Refresher{
			Store:  store,
			Config: graph.PKCEConfig(cfg.ClientID, cfg.TenantID, ""),
		}
		httpc := &graph.HTTPClient{
			Base:    "https://graph.microsoft.com/v1.0",
			Refresh: ref.Token,
		}
		d.Store = store
		d.Mail = httpc
		mailDelta := &graph.HTTPMailDelta{HTTPClient: httpc}
		d.MailChanges = mailDelta
		d.MailThreads = mailDelta
		d.Teams = &graph.HTTPTeams{HTTPClient: httpc}
		d.Calendar = &graph.HTTPCalendar{HTTPClient: httpc}
		d.Files = &graph.HTTPFiles{HTTPClient: httpc}
		d.Login = graph.RealLogin(cfg.ClientID, cfg.TenantID)
		d.DeviceLogin = graph.RealDeviceLogin(cfg.ClientID, cfg.TenantID)
		classification := cfg.Experimental.MailResponseClassification
		if cfgErr == nil && classification.Enabled {
			classifier, err := jev.NewClient(typesafeKey(getenv), classification.Model)
			if err != nil {
				d.MailClassifierError = err
			} else {
				d.MailClassifier = classifier
			}
		}
	}
	return d
}

// typesafeKey reads TYPESAFE_API_KEY, falling back to the TYPESAFE_AI_TOKEN alias.
func typesafeKey(getenv func(string) string) string {
	if key := getenv("TYPESAFE_API_KEY"); key != "" {
		return key
	}
	return getenv("TYPESAFE_AI_TOKEN")
}
