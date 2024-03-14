package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

var (
	errHTTPErrorResponse  = errors.New("HTTP error response")
	errMaxRetriesExceeded = errors.New("max retries exceeded")
)

const PathGetMevPayload = "eth/v1/builder/get_payload"

type BuilderAPIConfig struct {
	Endpoint string
}

func BuilderAPIDefaultConfig() *BuilderAPIConfig {
	return &BuilderAPIConfig{
		Endpoint: "",
	}
}

type BuilderAPIClient struct {
	log    log.Logger
	config *BuilderAPIConfig
}

func NewBuilderAPIClient(log log.Logger, config *BuilderAPIConfig) *BuilderAPIClient {
	return &BuilderAPIClient{
		config: config,
		log:    log,
	}
}

func (s *BuilderAPIClient) GetBuilderPayload(ctx context.Context, parent common.Hash) (*eth.ExecutionPayloadEnvelope, error) {
	responsePayload := new(eth.ExecutionPayloadEnvelope)
	url := fmt.Sprintf("%s/%s/%s", s.config.Endpoint, PathGetMevPayload, parent.Hex())
	httpClient := http.Client{Timeout: 10 * time.Second}

	if code, err := SendHTTPRequestWithRetries(
		ctx,
		httpClient,
		"GET",
		url,
		nil,
		nil,
		responsePayload,
		5,
		s.log); err != nil {
		return nil, err
	} else if code == http.StatusNoContent {
		s.log.Info("Could not get payload", "parent", parent.Hex())
		return nil, errors.New("could not get payload")
	}

	s.log.Info("Got payload", "payload", responsePayload)
	return responsePayload, nil
}

// SendHTTPRequest - prepare and send HTTP request, marshaling the payload if any, and decoding the response if dst is set
func SendHTTPRequest(ctx context.Context, client http.Client, method, url string, headers map[string]string, payload, dst any) (code int, err error) {
	var req *http.Request

	if payload == nil {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	} else {
		payloadBytes, err2 := json.Marshal(payload)
		if err2 != nil {
			return 0, fmt.Errorf("could not marshal request: %w", err2)
		}
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payloadBytes))

		// Set headers
		req.Header.Add("Content-Type", "application/json")
	}
	if err != nil {
		return 0, fmt.Errorf("could not prepare request: %w", err)
	}

	// Set user agent header
	req.Header.Set("User-Agent", "pbs-optimism")

	// Set other headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return resp.StatusCode, nil
	}

	if resp.StatusCode > 299 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp.StatusCode, fmt.Errorf("could not read error response body for status code %d: %w", resp.StatusCode, err)
		}
		return resp.StatusCode, fmt.Errorf("%w: %d / %s", errHTTPErrorResponse, resp.StatusCode, string(bodyBytes))
	}

	if dst != nil {
		log.Info("Decoding response body")
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp.StatusCode, fmt.Errorf("could not read response body: %w", err)
		}

		if err := json.Unmarshal(bodyBytes, dst); err != nil {
			return resp.StatusCode, fmt.Errorf("could not unmarshal response %s: %w", string(bodyBytes), err)
		}
	}

	return resp.StatusCode, nil
}

// SendHTTPRequestWithRetries - prepare and send HTTP request, retrying the request if within the client timeout
func SendHTTPRequestWithRetries(ctx context.Context, client http.Client, method, url string, headers map[string]string, payload, dst any, maxRetries int, log log.Logger) (code int, err error) {
	var requestCtx context.Context
	var cancel context.CancelFunc
	if client.Timeout > 0 {
		// Create a context with a timeout as configured in the http client
		requestCtx, cancel = context.WithTimeout(context.Background(), client.Timeout)
	} else {
		requestCtx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	attempts := 0
	for {
		attempts++
		if requestCtx.Err() != nil {
			return 0, fmt.Errorf("request context error after %d attempts: %w", attempts, requestCtx.Err())
		}
		if attempts > maxRetries {
			return 0, errMaxRetriesExceeded
		}

		code, err = SendHTTPRequest(ctx, client, method, url, headers, payload, dst)
		if err != nil {
			log.Error("error making request to relay, retrying", "err", err, "attempts", attempts)
			time.Sleep(100 * time.Millisecond) // note: this timeout is only applied between retries, it does not delay the initial request!
			continue
		}
		return code, nil
	}
}
