package state

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// CircleCIStore implements Store by fetching state from the most recent
// successful pipeline's artifacts and saving to a local artifacts directory.
//
// Load: queries CircleCI API for the latest artifact matching the key
// Save: writes to artifactsDir (CircleCI's store_artifacts step uploads it)
//
// This gives cross-pipeline state persistence without external storage.
type CircleCIStore struct {
	client       *http.Client
	token        string
	projectSlug  string // e.g. "gh/ethereum-optimism/optimism"
	branch       string
	workflow     string // workflow name to search for artifacts
	artifactsDir string // local dir for saving (uploaded by store_artifacts)
	statePrefix  string // artifact path prefix, e.g. "shadow-ci-state/"
}

// CircleCIStoreConfig configures the CircleCI artifact-backed state store.
type CircleCIStoreConfig struct {
	Token        string // CircleCI API token
	ProjectSlug  string // e.g. "gh/ethereum-optimism/optimism"
	Branch       string // branch to search for previous artifacts
	Workflow     string // workflow name filter
	ArtifactsDir string // local directory for saving artifacts
	StatePrefix  string // artifact path prefix (default: "shadow-ci-state/")
}

// NewCircleCIStore creates a CircleCI artifact-backed state store.
func NewCircleCIStore(cfg CircleCIStoreConfig) *CircleCIStore {
	prefix := cfg.StatePrefix
	if prefix == "" {
		prefix = "shadow-ci-state/"
	}
	os.MkdirAll(cfg.ArtifactsDir, 0o755)
	return &CircleCIStore{
		client:       &http.Client{},
		token:        cfg.Token,
		projectSlug:  cfg.ProjectSlug,
		branch:       cfg.Branch,
		workflow:      cfg.Workflow,
		artifactsDir: cfg.ArtifactsDir,
		statePrefix:  prefix,
	}
}

func (s *CircleCIStore) Load(key string) ([]byte, error) {
	// First check local (in case Save was called earlier in this run).
	localPath := filepath.Join(s.artifactsDir, s.statePrefix, key+".json")
	if data, err := os.ReadFile(localPath); err == nil {
		return data, nil
	}

	// Fetch from the most recent successful pipeline's artifacts.
	artifactURL, err := s.findLatestArtifact(key)
	if err != nil {
		return nil, err
	}
	if artifactURL == "" {
		return nil, ErrNotFound
	}

	return s.downloadArtifact(artifactURL)
}

func (s *CircleCIStore) Save(key string, data []byte) error {
	dir := filepath.Join(s.artifactsDir, s.statePrefix)
	os.MkdirAll(dir, 0o755)
	return os.WriteFile(filepath.Join(dir, key+".json"), data, 0o644)
}

// findLatestArtifact searches recent pipelines for the artifact matching key.
func (s *CircleCIStore) findLatestArtifact(key string) (string, error) {
	// Get recent pipelines for this branch.
	pipelines, err := s.getRecentPipelines()
	if err != nil {
		return "", fmt.Errorf("listing pipelines: %w", err)
	}

	targetPath := s.statePrefix + key + ".json"

	for _, pipelineID := range pipelines {
		// Get workflows for this pipeline.
		workflows, err := s.getWorkflows(pipelineID)
		if err != nil {
			continue
		}

		for _, wfID := range workflows {
			// Get jobs for this workflow.
			jobs, err := s.getJobs(wfID)
			if err != nil {
				continue
			}

			for _, jobNum := range jobs {
				// Get artifacts for this job.
				url, err := s.findArtifactInJob(jobNum, targetPath)
				if err != nil {
					continue
				}
				if url != "" {
					return url, nil
				}
			}
		}
	}

	return "", nil
}

// API response types.

type pipelineListResponse struct {
	Items []struct {
		ID string `json:"id"`
	} `json:"items"`
}

type workflowListResponse struct {
	Items []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"items"`
}

type jobListResponse struct {
	Items []struct {
		JobNumber int    `json:"job_number"`
		Status    string `json:"status"`
	} `json:"items"`
}

type artifactListResponse struct {
	Items []struct {
		Path string `json:"path"`
		URL  string `json:"url"`
	} `json:"items"`
}

func (s *CircleCIStore) getRecentPipelines() ([]string, error) {
	url := fmt.Sprintf("https://circleci.com/api/v2/project/%s/pipeline?branch=%s",
		s.projectSlug, s.branch)

	var resp pipelineListResponse
	if err := s.apiGet(url, &resp); err != nil {
		return nil, err
	}

	// Return up to 5 most recent.
	var ids []string
	for i, p := range resp.Items {
		if i >= 5 {
			break
		}
		ids = append(ids, p.ID)
	}
	return ids, nil
}

func (s *CircleCIStore) getWorkflows(pipelineID string) ([]string, error) {
	url := fmt.Sprintf("https://circleci.com/api/v2/pipeline/%s/workflow", pipelineID)

	var resp workflowListResponse
	if err := s.apiGet(url, &resp); err != nil {
		return nil, err
	}

	var ids []string
	for _, wf := range resp.Items {
		if wf.Status != "success" {
			continue
		}
		if s.workflow != "" && wf.Name != s.workflow {
			continue
		}
		ids = append(ids, wf.ID)
	}
	return ids, nil
}

func (s *CircleCIStore) getJobs(workflowID string) ([]int, error) {
	url := fmt.Sprintf("https://circleci.com/api/v2/workflow/%s/job", workflowID)

	var resp jobListResponse
	if err := s.apiGet(url, &resp); err != nil {
		return nil, err
	}

	var nums []int
	for _, j := range resp.Items {
		if j.Status == "success" {
			nums = append(nums, j.JobNumber)
		}
	}
	return nums, nil
}

func (s *CircleCIStore) findArtifactInJob(jobNumber int, targetPath string) (string, error) {
	url := fmt.Sprintf("https://circleci.com/api/v2/project/%s/%d/artifacts",
		s.projectSlug, jobNumber)

	var resp artifactListResponse
	if err := s.apiGet(url, &resp); err != nil {
		return "", err
	}

	for _, a := range resp.Items {
		if strings.HasSuffix(a.Path, targetPath) || a.Path == targetPath {
			return a.URL, nil
		}
	}
	return "", nil
}

func (s *CircleCIStore) apiGet(url string, result any) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Circle-Token", s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("CircleCI API %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (s *CircleCIStore) downloadArtifact(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Circle-Token", s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading artifact: HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
