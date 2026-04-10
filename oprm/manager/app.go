package manager

import (
	"context"
	"io"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/components"
	"github.com/ethereum-optimism/optimism/oprm/journal"
	"github.com/ethereum-optimism/optimism/oprm/providers/ghcli"
	"github.com/ethereum-optimism/optimism/oprm/providers/gitcli"
	"github.com/ethereum-optimism/optimism/oprm/providers/shell"
	"github.com/ethereum-optimism/optimism/oprm/release"
)

type App struct {
	Config   *Config
	Stdout   io.Writer
	Stderr   io.Writer
	now      func() time.Time
	store    *journal.Store
	registry *components.Registry
	git      gitcli.Provider
	gh       ghcli.Provider
}

func New(cfg *Config, stdout, stderr io.Writer) *App {
	runner := shell.NewRealRunner()
	return NewWithProviders(cfg, stdout, stderr, journal.NewStore(cfg.RunsDir), gitcli.New(runner), ghcli.New(runner), time.Now)
}

func NewWithProviders(cfg *Config, stdout, stderr io.Writer, store *journal.Store, git gitcli.Provider, gh ghcli.Provider, now func() time.Time) *App {
	if now == nil {
		now = time.Now
	}
	return &App{
		Config:   cfg,
		Stdout:   stdout,
		Stderr:   stderr,
		now:      now,
		store:    store,
		registry: components.NewRegistry(),
		git:      git,
		gh:       gh,
	}
}

func (a *App) Store() *journal.Store {
	return a.store
}

func (a *App) CheckoutLocation(componentID string) string {
	path, err := a.checkoutPath(componentID)
	if err != nil {
		return ""
	}
	return path
}

func (a *App) ReleaseNotesPath(run *release.Run, componentID string) string {
	proposal, ok := run.Versions[componentID]
	if !ok {
		return ""
	}
	return a.releaseNotesPath(run, componentID, proposal)
}

func (a *App) ComponentHeadSHA(componentID string) string {
	checkout, err := a.checkoutPath(componentID)
	if err != nil {
		return ""
	}
	sha, err := a.git.HeadSHA(context.Background(), checkout)
	if err != nil {
		return ""
	}
	return sha
}

func (a *App) ReleaseNotesBody(run *release.Run, componentID string) string {
	proposal, ok := run.Versions[componentID]
	if !ok {
		return ""
	}
	body, err := a.releaseNotesBody(run, componentID, proposal)
	if err != nil {
		return ""
	}
	return body
}
