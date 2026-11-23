package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/log"
)

const (
	DefaultTimeoutSeconds = 30

	// DefaultHTTPMaxResponseSize is the size limit for a successful BasicHTTPClient response. A
	// beacon API blobs response takes about 262 KB per blob, so this fits over 200 blobs.
	DefaultHTTPMaxResponseSize = 64 << 20
)

var _ HTTP = (*BasicHTTPClient)(nil)

type HTTP interface {
	Get(ctx context.Context, path string, query url.Values, headers http.Header) (*http.Response, error)
}

type BasicHTTPClient struct {
	endpoint string
	header   http.Header // optional header to use in every request

	log             log.Logger
	client          *http.Client
	maxResponseSize int64
}

func NewBasicHTTPClient(endpoint string, log log.Logger, opts ...BasicHTTPClientOption) *BasicHTTPClient {
	c := &BasicHTTPClient{
		endpoint:        endpoint,
		log:             log,
		client:          &http.Client{Timeout: DefaultTimeoutSeconds * time.Second},
		maxResponseSize: DefaultHTTPMaxResponseSize,
	}

	for _, opt := range opts {
		opt.Apply(c)
	}

	return c
}

type BasicHTTPClientOption interface {
	Apply(c *BasicHTTPClient)
}

type BasicHTTPClientOptionFn func(*BasicHTTPClient)

func (fn BasicHTTPClientOptionFn) Apply(c *BasicHTTPClient) {
	fn(c)
}

func WithHeader(h http.Header) BasicHTTPClientOption {
	return BasicHTTPClientOptionFn(func(c *BasicHTTPClient) {
		c.header = h
	})
}

func WithTransport(t http.RoundTripper) BasicHTTPClientOption {
	return BasicHTTPClientOptionFn(func(c *BasicHTTPClient) {
		c.client.Transport = t
	})
}

var ErrNoEndpoint = errors.New("no endpoint is configured")

// Get sends a GET request for path p. Reading the body of a successful (2xx) response fails with
// ErrResponseTooLarge after DefaultHTTPMaxResponseSize bytes; the body of any other response is
// cut off after 1 KiB.
func (cl *BasicHTTPClient) Get(ctx context.Context, p string, query url.Values, headers http.Header) (*http.Response, error) {
	if cl.endpoint == "" {
		return nil, ErrNoEndpoint
	}
	target, err := url.Parse(cl.endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint URL: %w", err)
	}
	// If we include the raw query in the path-join, it gets url-encoded,
	// and fails to parse as query, and ends up in the url.URL.Path part on the server side.
	// We want to avoid that, and insert the query manually. Real footgun in the url package.
	target = target.JoinPath(p)
	target.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to construct request", err)
	}
	addHTTPHeaders(req.Header, cl.header, headers)
	resp, err := cl.client.Do(req)
	if err != nil {
		return nil, err
	}
	limitResponseBody(resp, cl.maxResponseSize)
	return resp, nil
}

func addHTTPHeaders(header http.Header, hs ...http.Header) {
	for _, h := range hs {
		for key, values := range h {
			for _, value := range values {
				header.Add(key, value)
			}
		}
	}
}
