package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrResponseTooLarge is returned when a response body is read beyond its size limit.
var ErrResponseTooLarge = errors.New("response too large")

// maxErrorBodySize is how much of an unsuccessful (non-2xx) response body is read. Such a body
// only ends up in an error message.
const maxErrorBodySize = 1 << 10

// limitResponseBody bounds what can be read from resp.Body. Reading a successful response
// fails with ErrResponseTooLarge once it passes limit bytes; any other response is silently cut
// off at maxErrorBodySize bytes. net/http has already inflated a gzip-encoded body by the
// time it returns the response, so limit counts inflated bytes.
func limitResponseBody(resp *http.Response, limit int64) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body = &truncatedBody{Reader: io.LimitReader(resp.Body, maxErrorBodySize), Closer: resp.Body}
		return
	}
	resp.Body = &limitedBody{rc: resp.Body, remaining: limit, limit: limit}
}

type truncatedBody struct {
	io.Reader
	io.Closer
}

// limitedBody is like io.LimitReader, but returns ErrResponseTooLarge instead of io.EOF when
// the underlying reader has more than limit bytes.
type limitedBody struct {
	rc        io.ReadCloser
	remaining int64
	limit     int64
}

func (b *limitedBody) Read(p []byte) (int, error) {
	if b.remaining < 0 {
		return 0, b.tooLarge()
	}
	// Read one byte more than allowed, to tell a body of exactly limit bytes from a larger one.
	if int64(len(p)) > b.remaining {
		p = p[:b.remaining+1]
	}
	n, err := b.rc.Read(p)
	if int64(n) <= b.remaining {
		b.remaining -= int64(n)
		return n, err
	}
	n = int(b.remaining)
	b.remaining = -1
	return n, b.tooLarge()
}

func (b *limitedBody) tooLarge() error {
	return fmt.Errorf("%w: exceeds %d bytes", ErrResponseTooLarge, b.limit)
}

func (b *limitedBody) Close() error {
	return b.rc.Close()
}

// limitedTransport applies limitResponseBody to every response of its base transport.
type limitedTransport struct {
	base  http.RoundTripper
	limit int64
}

func (t *limitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	limitResponseBody(resp, t.limit)
	return resp, nil
}
