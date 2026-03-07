package events

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// LocalStore implements Store using local NDJSON files.
type LocalStore struct {
	dir string
	mu  sync.Mutex
}

// NewLocalStore creates a local file-backed event store.
func NewLocalStore(dir string) *LocalStore {
	os.MkdirAll(dir, 0o755)
	return &LocalStore{dir: dir}
}

func (s *LocalStore) Emit(event model.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	date := event.Timestamp.Format("2006-01-02")
	hour := event.Timestamp.Format("15")

	dayDir := filepath.Join(s.dir, "events", date)
	if err := os.MkdirAll(dayDir, 0o755); err != nil {
		return err
	}

	path := filepath.Join(dayDir, hour+".ndjson")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

func (s *LocalStore) Query(filter EventFilter) ([]model.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	eventsDir := filepath.Join(s.dir, "events")
	var events []model.Event

	err := filepath.Walk(eventsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		if info.IsDir() || !strings.HasSuffix(path, ".ndjson") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			var event model.Event
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}

			if matchesFilter(event, filter) {
				events = append(events, event)
				if filter.Limit > 0 && len(events) >= filter.Limit {
					return filepath.SkipAll
				}
			}
		}
		return nil
	})

	return events, err
}

func matchesFilter(event model.Event, filter EventFilter) bool {
	if len(filter.Types) > 0 {
		found := false
		for _, t := range filter.Types {
			if event.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if !filter.After.IsZero() && event.Timestamp.Before(filter.After) {
		return false
	}
	if !filter.Before.IsZero() && event.Timestamp.After(filter.Before) {
		return false
	}

	if filter.PR != nil && event.PR != *filter.PR {
		return false
	}
	if filter.PipelineID != nil && event.PipelineID != *filter.PipelineID {
		return false
	}

	return true
}

// QueryByTimeRange returns events within a time range, useful for reporting.
func (s *LocalStore) QueryByTimeRange(start, end time.Time, types ...model.EventType) ([]model.Event, error) {
	return s.Query(EventFilter{
		Types:  types,
		After:  start,
		Before: end,
	})
}
