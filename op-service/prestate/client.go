package prestate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

// These constants should be in sync with op-program/chainconfig/chaincfg.go.
const (
	InteropDepSetName    = "depsets.json"
	rollupConfigSuffix   = "-rollup.json"
	genensisConfigSuffix = "-genesis-l2.json"
)

// PrestateManifest maps prestate identifiers to their hashes.
type PrestateManifest map[string]string

// PrestateBuilderClient is a client for the prestate builder service.
type PrestateBuilderClient struct {
	url    string
	client *http.Client
}

// NewPrestateBuilderClient creates a new client for the prestate builder service.
func NewPrestateBuilderClient(url string) *PrestateBuilderClient {
	return &PrestateBuilderClient{
		url:    url,
		client: &http.Client{},
	}
}

// FileInput represents a file to be used in the build process.
type FileInput struct {
	Name    string
	Content io.Reader
	Type    string
}

type buildContext struct {
	chains                 []string
	files                  []FileInput
	generatedInteropDepSet bool
}

// PrestateBuilderOption is a functional option for configuring a build.
type PrestateBuilderOption func(*buildContext)

// WithInteropDepSet adds an interop dependency set file to the build.
func WithInteropDepSet(content io.Reader) PrestateBuilderOption {
	return func(c *buildContext) {
		c.files = append(c.files, FileInput{
			Name:    InteropDepSetName,
			Content: content,
			Type:    "interop",
		})
	}
}

func generateInteropDepSet(chains []string) ([]byte, error) {
	deps := make(map[eth.ChainID]*depset.StaticConfigDependency)
	for _, chain := range chains {
		id, err := eth.ParseDecimalChainID(chain)
		if err != nil {
			return nil, fmt.Errorf("failed to parse chain ID: %w", err)
		}
		deps[id] = &depset.StaticConfigDependency{}
	}

	interopDepSet, err := depset.NewStaticConfigDependencySet(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create interop dependency set: %w", err)
	}

	jsonBytes, err := json.Marshal(interopDepSet)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

// WithGeneratedInteropDepSet requests generation of the interop dependency set from the chain list.
func WithGeneratedInteropDepSet() PrestateBuilderOption {
	return func(c *buildContext) {
		c.generatedInteropDepSet = true
	}
}

// WithChainConfig adds a pair of rollup and genesis config files to the build.
func WithChainConfig(chainID string, rollupContent io.Reader, genesisContent io.Reader) PrestateBuilderOption {
	return func(c *buildContext) {
		c.chains = append(c.chains, chainID)
		c.files = append(c.files,
			FileInput{
				Name:    chainID + rollupConfigSuffix,
				Content: rollupContent,
				Type:    "rollup-config",
			},
			FileInput{
				Name:    chainID + genensisConfigSuffix,
				Content: genesisContent,
				Type:    "genesis-config",
			},
		)
	}
}

// BuildPrestate sends the files to the prestate builder service and returns a manifest of the built prestates.
func (c *PrestateBuilderClient) BuildPrestate(ctx context.Context, opts ...PrestateBuilderOption) (PrestateManifest, error) {
	fmt.Println("Starting prestate build...")

	bc := &buildContext{
		files: []FileInput{},
	}
	for _, opt := range opts {
		opt(bc)
	}

	if bc.generatedInteropDepSet {
		depSet, err := generateInteropDepSet(bc.chains)
		if err != nil {
			return nil, fmt.Errorf("failed to generate interop dependency set: %w", err)
		}
		bc.files = append(bc.files, FileInput{
			Name:    InteropDepSetName,
			Content: bytes.NewReader(depSet),
			Type:    "interop",
		})
	}

	fmt.Printf("Preparing to upload %d files\n", len(bc.files))

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, file := range bc.files {
		fmt.Printf("Adding file to form: %s (type: %s)\n", file.Name, file.Type)
		formFile, err := writer.CreateFormFile("files[]", filepath.Base(file.Name))
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}
		if _, err := io.Copy(formFile, file.Content); err != nil {
			return nil, fmt.Errorf("failed to copy file content: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	fmt.Printf("Sending build request to %s\n", c.url)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotModified {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	fmt.Println("Build request successful, fetching build manifest...")

	infoURL := c.url
	if infoURL[len(infoURL)-1] != '/' {
		infoURL += "/"
	}
	infoURL += "info.json"

	fmt.Printf("Requesting manifest from %s\n", infoURL)
	infoReq, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create info request: %w", err)
	}

	infoResp, err := c.client.Do(infoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get info.json: %w", err)
	}
	defer infoResp.Body.Close()

	if infoResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(infoResp.Body)
		return nil, fmt.Errorf("unexpected info.json status code: %d, body: %s", infoResp.StatusCode, string(respBody))
	}

	var manifest PrestateManifest
	if err := json.NewDecoder(infoResp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode info.json response: %w", err)
	}

	fmt.Printf("Build complete. Generated %d prestate entries\n", len(manifest))
	return manifest, nil
}
