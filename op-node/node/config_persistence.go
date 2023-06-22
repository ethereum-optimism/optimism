package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type RunningState int

const (
	Unset RunningState = iota
	Started
	Stopped
)

type persistedState struct {
	SequencerStarted *bool `json:"sequencerStarted,omitempty"`
}

type ConfigPersistence interface {
	SequencerStarted() error
	SequencerStopped() error
	SequencerState() (RunningState, error)
}

var _ ConfigPersistence = (*ActiveConfigPersistence)(nil)
var _ ConfigPersistence = DisabledConfigPersistence{}

type ActiveConfigPersistence struct {
	lock sync.Mutex
	file string
}

func NewConfigPersistence(file string) (*ActiveConfigPersistence, error) {
	return &ActiveConfigPersistence{file: file}, nil
}

func (p *ActiveConfigPersistence) SequencerStarted() error {
	return p.persist(true)
}

func (p *ActiveConfigPersistence) SequencerStopped() error {
	return p.persist(false)
}

// persist writes the new config state to the file as safely as possible.
// It uses sync to ensure the data is actually persisted to disk and initially writes to a temp file
// before renaming it into place. On UNIX systems this rename is typically atomic, ensuring the
// actual file isn't corrupted if IO errors occur during writing.
func (p *ActiveConfigPersistence) persist(sequencerStarted bool) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	data, err := json.Marshal(persistedState{SequencerStarted: &sequencerStarted})
	if err != nil {
		return fmt.Errorf("marshall new config: %w", err)
	}
	dir := filepath.Dir(p.file)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir (%v): %w", p.file, err)
	}
	// Write the new content to a temp file first, then rename into place
	// Avoids corrupting the content if the disk is full or there are IO errors
	tmpFile := p.file + ".tmp"
	file, err := os.OpenFile(tmpFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open file (%v) for writing: %w", tmpFile, err)
	}
	defer file.Close()
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("write new config to temp file (%v): %w", tmpFile, err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync new config temp file (%v): %w", tmpFile, err)
	}

	// Rename to replace the previous file
	if err := os.Rename(tmpFile, p.file); err != nil {
		return fmt.Errorf("rename temp config file to final destination: %w", err)
	}
	return nil
}

func (p *ActiveConfigPersistence) SequencerState() (RunningState, error) {
	config, err := p.read()
	if err != nil {
		return Unset, err
	}

	if config.SequencerStarted == nil {
		return Unset, nil
	} else if *config.SequencerStarted {
		return Started, nil
	} else {
		return Stopped, nil
	}
}

func (p *ActiveConfigPersistence) read() (persistedState, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	data, err := os.ReadFile(p.file)
	if errors.Is(err, os.ErrNotExist) {
		return persistedState{}, nil
	} else if err != nil {
		return persistedState{}, fmt.Errorf("read config file (%v): %w", p.file, err)
	}
	var config persistedState
	if err = json.Unmarshal(data, &config); err != nil {
		return persistedState{}, fmt.Errorf("invalid config file (%v): %w", p.file, err)
	}
	return config, nil
}

// DisabledConfigPersistence provides an implementation of config persistence
// that does not persist anything and reports unset for all values
type DisabledConfigPersistence struct {
}

func (d DisabledConfigPersistence) SequencerState() (RunningState, error) {
	return Unset, nil
}

func (d DisabledConfigPersistence) SequencerStarted() error {
	return nil
}

func (d DisabledConfigPersistence) SequencerStopped() error {
	return nil
}
