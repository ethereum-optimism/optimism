package cihistory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
)

// fakeCircleCI is a minimal httptest server that serves a canned set
// of pipelines → workflows → jobs. Pagination and auth behavior are
// tested separately.
type fakeCircleCI struct {
	pipelines map[string]ccPipelinePage // query string → page
	workflows map[string]ccWorkflowPage // pipeline ID → workflows
	jobs      map[string]ccJobPage      // workflow ID → jobs
	sawAuth   []string                  // every Circle-Token header seen
}

func (f *fakeCircleCI) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.Helper()
		f.sawAuth = append(f.sawAuth, r.Header.Get("Circle-Token"))
		p := r.URL.Path

		switch {
		case strings.HasSuffix(p, "/pipeline"):
			key := r.URL.Query().Get("page-token")
			page, ok := f.pipelines[key]
			if !ok {
				http.Error(w, "unexpected page-token: "+key, http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(page)

		case strings.Contains(p, "/pipeline/") && strings.HasSuffix(p, "/workflow"):
			id := pathSegment(p, "/pipeline/", "/workflow")
			_ = json.NewEncoder(w).Encode(f.workflows[id])

		case strings.Contains(p, "/workflow/") && strings.HasSuffix(p, "/job"):
			id := pathSegment(p, "/workflow/", "/job")
			_ = json.NewEncoder(w).Encode(f.jobs[id])

		default:
			http.Error(w, "unexpected path: "+p, http.StatusNotFound)
		}
	}
}

func pathSegment(full, prefix, suffix string) string {
	i := strings.Index(full, prefix)
	if i < 0 {
		return ""
	}
	s := full[i+len(prefix):]
	j := strings.Index(s, suffix)
	if j < 0 {
		return ""
	}
	return s[:j]
}

// TestCircleCI_Fetch_HappyPath — one pipeline, one workflow, two jobs
// (forge-test passes, forge-test variant fails) maps to one failed
// check via CIJobNames, and the file list comes from the stub.
func TestCircleCI_Fetch_HappyPath(t *testing.T) {
	merged := time.Now().UTC().Add(-2 * time.Hour)
	fc := &fakeCircleCI{
		pipelines: map[string]ccPipelinePage{
			"": {Items: []ccPipeline{{
				ID:        "pipe-1",
				CreatedAt: merged,
				VCS: struct {
					Revision string `json:"revision"`
					Branch   string `json:"branch"`
				}{Revision: "abc123", Branch: "develop"},
			}}},
		},
		workflows: map[string]ccWorkflowPage{
			"pipe-1": {Items: []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{ID: "wf-1", Name: "main", Status: "failed"}}},
		},
		jobs: map[string]ccJobPage{
			"wf-1": {Items: []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			}{
				{Name: "contracts-bedrock-tests", Status: "success"},
				{Name: "contracts-bedrock-tests-custom_gas_token", Status: "failed"},
			}},
		},
	}

	srv := httptest.NewServer(fc.handler(t))
	defer srv.Close()

	cat, _ := catalog.Parse([]byte(`
check_types:
  - id: forge-test
    name: forge-test
    kind: test
    language: solidity
    command: forge test
    scopeable: true
    scope_type: paths
    avg_duration: 3600
    ci_job_names:
      - contracts-bedrock-tests
      - contracts-bedrock-tests-custom_gas_token
`))

	f := NewCircleCIFetcher(CircleCIConfig{
		Org:     "ethereum-optimism",
		Repo:    "optimism",
		Branch:  "develop",
		BaseURL: srv.URL,
		FilesFor: func(rev string) ([]string, error) {
			if rev != "abc123" {
				t.Errorf("FilesFor called with %q, want abc123", rev)
			}
			return []string{"packages/contracts-bedrock/src/L1/X.sol"}, nil
		},
	}, JobMapFromCatalog(cat))

	events, err := f.Fetch(time.Time{})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	e := events[0]
	if len(e.Checks) != 1 || e.Checks[0].ID != "forge-test" {
		t.Fatalf("checks = %+v, want one forge-test", e.Checks)
	}
	if !e.Checks[0].Failed {
		t.Error("forge-test should be marked failed (any job-variant fail = fail)")
	}
	if len(e.Files) != 1 || e.Files[0] != "packages/contracts-bedrock/src/L1/X.sol" {
		t.Errorf("files = %+v, want the stub's file list", e.Files)
	}
}

// TestCircleCI_Fetch_FiltersBranch — pipelines on other branches are
// skipped even if the API returns them.
func TestCircleCI_Fetch_FiltersBranch(t *testing.T) {
	fc := &fakeCircleCI{
		pipelines: map[string]ccPipelinePage{
			"": {Items: []ccPipeline{
				{ID: "p-feature", VCS: struct {
					Revision string `json:"revision"`
					Branch   string `json:"branch"`
				}{Revision: "deadbeef", Branch: "feature-branch"}},
				{ID: "p-develop", VCS: struct {
					Revision string `json:"revision"`
					Branch   string `json:"branch"`
				}{Revision: "cafef00d", Branch: "develop"}},
			}},
		},
		workflows: map[string]ccWorkflowPage{
			"p-develop": {Items: []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{ID: "wf-d"}}},
		},
		jobs: map[string]ccJobPage{
			"wf-d": {Items: []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{Name: "x", Status: "success"}}},
		},
	}
	srv := httptest.NewServer(fc.handler(t))
	defer srv.Close()

	f := NewCircleCIFetcher(CircleCIConfig{
		Org: "o", Repo: "r", Branch: "develop", BaseURL: srv.URL,
		FilesFor: func(rev string) ([]string, error) { return []string{"a.go"}, nil },
	}, map[string]string{"x": "go-test"})

	events, _ := f.Fetch(time.Time{})
	for _, e := range events {
		// Only develop pipeline should yield an event.
		for _, f := range e.Files {
			if f == "" {
				t.Error("empty file in event")
			}
		}
	}
	// The feature-branch pipeline should be filtered; only develop remains.
	if len(events) != 1 {
		t.Errorf("events = %d, want 1 (feature branch filtered)", len(events))
	}
}

// TestCircleCI_Fetch_IgnoresUnmappedJobs — jobs without a check-ID
// mapping don't show up in the output, and pipelines with *only*
// unmapped jobs produce zero events.
func TestCircleCI_Fetch_IgnoresUnmappedJobs(t *testing.T) {
	fc := &fakeCircleCI{
		pipelines: map[string]ccPipelinePage{
			"": {Items: []ccPipeline{{ID: "p1", VCS: struct {
				Revision string `json:"revision"`
				Branch   string `json:"branch"`
			}{Revision: "r", Branch: "develop"}}}},
		},
		workflows: map[string]ccWorkflowPage{
			"p1": {Items: []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{ID: "w1"}}},
		},
		jobs: map[string]ccJobPage{
			"w1": {Items: []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{Name: "unknown-lint", Status: "failed"}}},
		},
	}
	srv := httptest.NewServer(fc.handler(t))
	defer srv.Close()

	f := NewCircleCIFetcher(CircleCIConfig{
		Org: "o", Repo: "r", Branch: "develop", BaseURL: srv.URL,
		FilesFor: func(string) ([]string, error) { return []string{"x.sol"}, nil },
	}, map[string]string{}) // empty map

	events, _ := f.Fetch(time.Time{})
	if len(events) != 0 {
		t.Errorf("events = %d, want 0 (no mapped jobs)", len(events))
	}
}

// TestCircleCI_Fetch_StatusClassification — success+failed mapped to
// the same check ID aggregates to Failed=true; in-flight jobs are
// ignored (not counted as "ran").
func TestCircleCI_Fetch_StatusClassification(t *testing.T) {
	cases := []struct {
		name       string
		statuses   []string
		wantFailed bool
		wantRan    bool
	}{
		{"all success", []string{"success"}, false, true},
		{"one failed among successes", []string{"success", "failed"}, true, true},
		{"timedout counts as failed", []string{"success", "timedout"}, true, true},
		{"only running — not ran", []string{"running"}, false, false},
		{"canceled is skip, not fail", []string{"canceled"}, false, false},
		{"mixed running + failed", []string{"running", "failed"}, true, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			items := make([]struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			}, len(c.statuses))
			for i, s := range c.statuses {
				items[i] = struct {
					Name   string `json:"name"`
					Status string `json:"status"`
				}{Name: "forge-test", Status: s}
			}
			fc := &fakeCircleCI{
				pipelines: map[string]ccPipelinePage{
					"": {Items: []ccPipeline{{ID: "p", VCS: struct {
						Revision string `json:"revision"`
						Branch   string `json:"branch"`
					}{Revision: "r", Branch: "develop"}}}},
				},
				workflows: map[string]ccWorkflowPage{
					"p": {Items: []struct {
						ID     string `json:"id"`
						Name   string `json:"name"`
						Status string `json:"status"`
					}{{ID: "w"}}},
				},
				jobs: map[string]ccJobPage{"w": {Items: items}},
			}
			srv := httptest.NewServer(fc.handler(t))
			defer srv.Close()

			f := NewCircleCIFetcher(CircleCIConfig{
				Org: "o", Repo: "r", Branch: "develop", BaseURL: srv.URL,
				FilesFor: func(string) ([]string, error) { return []string{"x.sol"}, nil },
			}, map[string]string{"forge-test": "forge-test"})

			events, _ := f.Fetch(time.Time{})
			if !c.wantRan {
				if len(events) != 0 {
					t.Errorf("%s: events=%d, want 0 (nothing ran)", c.name, len(events))
				}
				return
			}
			if len(events) != 1 {
				t.Fatalf("%s: events=%d, want 1", c.name, len(events))
			}
			if len(events[0].Checks) != 1 {
				t.Fatalf("%s: checks=%d, want 1", c.name, len(events[0].Checks))
			}
			if events[0].Checks[0].Failed != c.wantFailed {
				t.Errorf("%s: Failed=%v, want %v", c.name, events[0].Checks[0].Failed, c.wantFailed)
			}
		})
	}
}

// TestCircleCI_Fetch_Pagination — the pipeline walk follows
// next_page_token until the API returns an empty token, and each page
// can carry a different page-token in the request.
func TestCircleCI_Fetch_Pagination(t *testing.T) {
	noVCS := struct {
		Revision string `json:"revision"`
		Branch   string `json:"branch"`
	}{Revision: "r", Branch: "develop"}

	fc := &fakeCircleCI{
		pipelines: map[string]ccPipelinePage{
			"": {
				Items:         []ccPipeline{{ID: "p1", VCS: noVCS}},
				NextPageToken: "TOKEN1",
			},
			"TOKEN1": {
				Items: []ccPipeline{{ID: "p2", VCS: noVCS}},
			},
		},
		workflows: map[string]ccWorkflowPage{
			"p1": {Items: []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{ID: "w1"}}},
			"p2": {Items: []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{ID: "w2"}}},
		},
		jobs: map[string]ccJobPage{
			"w1": {Items: []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{Name: "t", Status: "success"}}},
			"w2": {Items: []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{Name: "t", Status: "failed"}}},
		},
	}
	srv := httptest.NewServer(fc.handler(t))
	defer srv.Close()

	f := NewCircleCIFetcher(CircleCIConfig{
		Org: "o", Repo: "r", Branch: "develop", BaseURL: srv.URL,
		FilesFor: func(string) ([]string, error) { return []string{"x"}, nil },
	}, map[string]string{"t": "test-check"})

	events, err := f.Fetch(time.Time{})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("events = %d, want 2 (both pipelines across pages)", len(events))
	}
}

// TestCircleCI_Fetch_SkipsPipelinesWithNoFiles — when FilesFor errors
// (revision not in local repo), the pipeline is silently dropped
// rather than failing the whole walk.
func TestCircleCI_Fetch_SkipsPipelinesWithNoFiles(t *testing.T) {
	fc := &fakeCircleCI{
		pipelines: map[string]ccPipelinePage{
			"": {Items: []ccPipeline{{ID: "p", VCS: struct {
				Revision string `json:"revision"`
				Branch   string `json:"branch"`
			}{Revision: "missing", Branch: "develop"}}}},
		},
		workflows: map[string]ccWorkflowPage{
			"p": {Items: []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{ID: "w"}}},
		},
		jobs: map[string]ccJobPage{
			"w": {Items: []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{Name: "t", Status: "failed"}}},
		},
	}
	srv := httptest.NewServer(fc.handler(t))
	defer srv.Close()

	callErr := false
	f := NewCircleCIFetcher(CircleCIConfig{
		Org: "o", Repo: "r", Branch: "develop", BaseURL: srv.URL,
		FilesFor: func(rev string) ([]string, error) {
			callErr = true
			return nil, http.ErrAbortHandler // any error — representative of "git can't find revision"
		},
	}, map[string]string{"t": "x"})

	events, err := f.Fetch(time.Time{})
	if err != nil {
		t.Fatalf("Fetch errored: %v; expected silent skip", err)
	}
	if !callErr {
		t.Error("FilesFor was never called")
	}
	if len(events) != 0 {
		t.Errorf("events = %d, want 0 (skipped)", len(events))
	}
}

// TestCircleCI_Auth_TokenSent — when Token is set, Circle-Token header
// is present on every request; when empty, it's omitted.
func TestCircleCI_Auth_TokenSent(t *testing.T) {
	fc := &fakeCircleCI{
		pipelines: map[string]ccPipelinePage{"": {}},
	}
	srv := httptest.NewServer(fc.handler(t))
	defer srv.Close()

	// With token
	f := NewCircleCIFetcher(CircleCIConfig{
		Org: "o", Repo: "r", Branch: "develop", BaseURL: srv.URL, Token: "SECRET",
		FilesFor: func(string) ([]string, error) { return nil, nil },
	}, nil)
	_, _ = f.Fetch(time.Time{})
	for _, tok := range fc.sawAuth {
		if tok != "SECRET" {
			t.Errorf("Circle-Token = %q, want SECRET", tok)
		}
	}
	fc.sawAuth = nil

	// Without token
	f = NewCircleCIFetcher(CircleCIConfig{
		Org: "o", Repo: "r", Branch: "develop", BaseURL: srv.URL,
		FilesFor: func(string) ([]string, error) { return nil, nil },
	}, nil)
	_, _ = f.Fetch(time.Time{})
	for _, tok := range fc.sawAuth {
		if tok != "" {
			t.Errorf("Circle-Token should be omitted, got %q", tok)
		}
	}
}

// TestCircleCI_Fetch_StopsAtSince — pipelines older than `since` halt
// pagination (since pipelines are newest-first).
func TestCircleCI_Fetch_StopsAtSince(t *testing.T) {
	now := time.Now().UTC()
	recent := now.Add(-1 * time.Hour)
	ancient := now.Add(-30 * 24 * time.Hour)

	fc := &fakeCircleCI{
		pipelines: map[string]ccPipelinePage{
			"": {Items: []ccPipeline{
				{ID: "recent", CreatedAt: recent, VCS: struct {
					Revision string `json:"revision"`
					Branch   string `json:"branch"`
				}{Revision: "r", Branch: "develop"}},
				{ID: "ancient", CreatedAt: ancient, VCS: struct {
					Revision string `json:"revision"`
					Branch   string `json:"branch"`
				}{Revision: "a", Branch: "develop"}},
			}},
		},
		workflows: map[string]ccWorkflowPage{
			"recent": {Items: []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{ID: "w"}}},
		},
		jobs: map[string]ccJobPage{
			"w": {Items: []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			}{{Name: "t", Status: "success"}}},
		},
	}
	srv := httptest.NewServer(fc.handler(t))
	defer srv.Close()

	f := NewCircleCIFetcher(CircleCIConfig{
		Org: "o", Repo: "r", Branch: "develop", BaseURL: srv.URL,
		FilesFor: func(string) ([]string, error) { return []string{"x"}, nil },
	}, map[string]string{"t": "test-check"})

	events, _ := f.Fetch(now.Add(-24 * time.Hour))
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1 (ancient filtered)", len(events))
	}
}

// TestJobMapFromCatalog — ci_job_names from multiple checks flatten to
// a single map.
func TestJobMapFromCatalog(t *testing.T) {
	cat, err := catalog.Parse([]byte(`
check_types:
  - id: forge-test
    name: forge-test
    kind: test
    language: solidity
    command: forge test
    scopeable: true
    avg_duration: 1
    ci_job_names:
      - contracts-bedrock-tests
      - contracts-bedrock-tests-cgt
  - id: go-test
    name: go-test
    kind: test
    language: go
    command: go test
    scopeable: true
    avg_duration: 1
    ci_job_names:
      - go-tests
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	m := JobMapFromCatalog(cat)
	if m["contracts-bedrock-tests"] != "forge-test" {
		t.Errorf("contracts-bedrock-tests → %q, want forge-test", m["contracts-bedrock-tests"])
	}
	if m["contracts-bedrock-tests-cgt"] != "forge-test" {
		t.Errorf("variant → %q, want forge-test", m["contracts-bedrock-tests-cgt"])
	}
	if m["go-tests"] != "go-test" {
		t.Errorf("go-tests → %q, want go-test", m["go-tests"])
	}
}
