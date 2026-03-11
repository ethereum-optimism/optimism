package circleci

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Client is a CircleCI API client.
type Client struct {
	token   string
	baseURL string
	project string
	client  *http.Client
}

// NewClient creates a new CircleCI API client.
func NewClient(token, project string) *Client {
	return &Client{
		token:   token,
		baseURL: "https://circleci.com/api/v2",
		project: project,
		client:  &http.Client{},
	}
}

// NewClientFromEnv creates a client using environment variables.
func NewClientFromEnv() *Client {
	return NewClient(
		os.Getenv("CIRCLE_TOKEN"),
		"gh/ethereum-optimism/optimism",
	)
}

func (c *Client) get(path string) ([]byte, error) {
	url := c.baseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Circle-Token", c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d: %s", path, resp.StatusCode, string(body))
	}

	return body, nil
}

// pipelineWorkflows represents the response from the workflows endpoint.
type pipelineWorkflows struct {
	Items []workflowItem `json:"items"`
}

type workflowItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// GetWorkflows returns all workflows for a pipeline.
func (c *Client) GetWorkflows(pipelineID string) ([]workflowItem, error) {
	data, err := c.get(fmt.Sprintf("/pipeline/%s/workflow", pipelineID))
	if err != nil {
		return nil, err
	}

	var resp pipelineWorkflows
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// workflowJobs represents the response from the workflow jobs endpoint.
type workflowJobs struct {
	Items []jobItem `json:"items"`
}

type jobItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	JobNumber   int    `json:"job_number"`
	ProjectSlug string `json:"project_slug"`
}

// GetWorkflowJobs returns all jobs for a workflow.
func (c *Client) GetWorkflowJobs(workflowID string) ([]jobItem, error) {
	data, err := c.get(fmt.Sprintf("/workflow/%s/job", workflowID))
	if err != nil {
		return nil, err
	}

	var resp workflowJobs
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// jobArtifacts represents the response from the artifacts endpoint.
type jobArtifacts struct {
	Items []ArtifactItem `json:"items"`
}

// ArtifactItem represents a CircleCI job artifact.
type ArtifactItem struct {
	Path string `json:"path"`
	URL  string `json:"url"`
}

// GetJobArtifacts returns all artifacts for a job.
func (c *Client) GetJobArtifacts(projectSlug string, jobNumber int) ([]ArtifactItem, error) {
	data, err := c.get(fmt.Sprintf("/project/%s/%d/artifacts", projectSlug, jobNumber))
	if err != nil {
		return nil, err
	}

	var resp jobArtifacts
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// Pipeline represents a CircleCI pipeline.
type Pipeline struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	Branch    string `json:"vcs,omitempty"`
	CreatedAt string `json:"created_at"`
}

// pipelineResponse is the API response for pipeline listing.
type pipelineResponse struct {
	Items         []Pipeline `json:"items"`
	NextPageToken string     `json:"next_page_token"`
}

// ListPipelines lists recent pipelines for the project.
func (c *Client) ListPipelines(branch string, limit int) ([]Pipeline, error) {
	path := fmt.Sprintf("/project/%s/pipeline?page-token=", c.project)
	if branch != "" {
		path += "&branch=" + branch
	}

	data, err := c.get(path)
	if err != nil {
		return nil, err
	}

	var resp pipelineResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	result := resp.Items
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// DownloadArtifact downloads an artifact by its URL.
func (c *Client) DownloadArtifact(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Circle-Token", c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
