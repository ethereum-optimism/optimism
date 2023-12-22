package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

const (
	DefaultTimeoutSeconds = 30
)

var _ HTTP = (*BasicHTTPClient)(nil)

type HTTP interface {
	Get(ctx context.Context, path string, headers http.Header) (*http.Response, error)
}

type BasicHTTPClient struct {
	endpoint string
	log      log.Logger
	client   *http.Client
}

func NewBasicHTTPClient(endpoint string, log log.Logger) *BasicHTTPClient {
	// Make sure the endpoint ends in trailing slash
	trimmedEndpoint := strings.TrimSuffix(endpoint, "/") + "/"
	return &BasicHTTPClient{
		endpoint: trimmedEndpoint,
		log:      log,
		client:   &http.Client{Timeout: DefaultTimeoutSeconds * time.Second},
	}
}

func (cl *BasicHTTPClient) Get(ctx context.Context, p string, headers http.Header) (*http.Response, error) {
	u, err := url.JoinPath(cl.endpoint, p)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to join path", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to construct request", err)
	}
	for k, values := range headers {
		for _, v := range values {
			req.Header.Add(k, v)
		}
	}
	return cl.client.Do(req)
}
