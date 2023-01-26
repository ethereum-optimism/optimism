package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type HTTP interface {
	Get(ctx context.Context, path string, headers http.Header) (*http.Response, error)
}

type BasicHTTPClient struct {
	endpoint string
	log      log.Logger
	client   http.Client
}

func NewBasicHTTPClient(endpoint string, log log.Logger) *BasicHTTPClient {
	return &BasicHTTPClient{endpoint: endpoint, log: log, client: http.Client{Timeout: 30 * time.Second}}
}

func (cl *BasicHTTPClient) Get(ctx context.Context, p string, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cl.endpoint+"/"+p, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request: %w", err)
	}
	for k, values := range headers {
		for _, v := range values {
			req.Header.Add(k, v)
		}
	}
	return cl.client.Do(req)
}

// TODO HTTP client wrapper to track response time and error metrics
