package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

var (
	errHTTPErrorResponse  = errors.New("HTTP error response")
	errMaxRetriesExceeded = errors.New("max retries exceeded")
)

const PathGetPayload = "/eth/v1/builder/payload"

type BuilderAPIConfig struct {
	Endpoint string
}

func BuilderAPIDefaultConfig() *BuilderAPIConfig {
	return &BuilderAPIConfig{
		Endpoint: "",
	}
}

type BuilderAPIClient struct {
	log        log.Logger
	config     *BuilderAPIConfig
	httpClient *client.BasicHTTPClient
}

func NewBuilderAPIClient(log log.Logger, config *BuilderAPIConfig) *BuilderAPIClient {
	httpClient := client.NewBasicHTTPClient(config.Endpoint, log)

	return &BuilderAPIClient{
		httpClient: httpClient,
		config:     config,
		log:        log,
	}
}

func (s *BuilderAPIClient) GetPayload(ctx context.Context, ref eth.L2BlockRef) (*eth.ExecutionPayloadEnvelope, error) {
	responsePayload := new(eth.ExecutionPayloadEnvelope)
	ps := fmt.Sprintf("%s/%s/%s", PathGetPayload, ref.Number, ref.Hash)
	query := url.Values{"key": []string{"123"}}
	resp, err := s.httpClient.Get(ctx, ps, query, http.Header{})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errHTTPErrorResponse
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bodyBytes, responsePayload); err != nil {
		return nil, err
	}

	return responsePayload, nil
}
