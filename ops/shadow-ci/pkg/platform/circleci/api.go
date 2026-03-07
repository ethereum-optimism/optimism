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
	Items []artifactItem `json:"items"`
}

type artifactItem struct {
	Path string `json:"path"`
	URL  string `json:"url"`
}

// GetJobArtifacts returns all artifacts for a job.
func (c *Client) GetJobArtifacts(projectSlug string, jobNumber int) ([]artifactItem, error) {
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
