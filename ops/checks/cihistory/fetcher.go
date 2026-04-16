package cihistory

import "time"

// Fetcher returns CI events in a time window. Implementations can read
// from a local JSON file (FileFetcher), the CircleCI API, or the GitHub
// check-runs API — the analysis code doesn't care.
type Fetcher interface {
	Fetch(since time.Time) ([]Event, error)
}

// FileFetcher reads events from a JSON file produced by an external
// scraper. The file format is a top-level JSON array of Event objects.
type FileFetcher struct {
	Path string
}

// NewFileFetcher returns a Fetcher that reads from path.
func NewFileFetcher(path string) *FileFetcher {
	return &FileFetcher{Path: path}
}

// Fetch reads events from disk and filters by since. Events with no
// MergedAt timestamp are included regardless (treat as current).
func (f *FileFetcher) Fetch(since time.Time) ([]Event, error) {
	all, err := LoadEvents(f.Path)
	if err != nil {
		return nil, err
	}
	if since.IsZero() {
		return all, nil
	}
	filtered := all[:0]
	for _, e := range all {
		if e.MergedAt.IsZero() || !e.MergedAt.Before(since) {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
}
