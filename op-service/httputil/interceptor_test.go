package httputil

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeHTTPTransport struct {
	resp *http.Response
	err  error
}

func (f *fakeHTTPTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return f.resp, f.err
}

func TestHTTPInterceptor(t *testing.T) {
	out := "start"
	basic := HTTPInterceptorFunc(func(req *http.Request, inner http.RoundTripper) (resp *http.Response, err error) {
		out += " " + req.Header.Get("X_TEST_PRE")
		resp, err = inner.RoundTrip(req)
		out += " " + resp.Header.Get("X_TEST_POST")
		return
	})
	req := &http.Request{Header: make(http.Header)}
	req.Header.Set("X_TEST_PRE", "foo")

	resp := &http.Response{Header: make(http.Header), Body: io.NopCloser(bytes.NewReader([]byte{}))}
	resp.Header.Set("X_TEST_POST", "bar")

	inner := &fakeHTTPTransport{resp: resp, err: nil}

	got, err := basic.Intercept(req, inner)
	require.NoError(t, err)
	_ = got.Body.Close() // Make lint happy
	require.Equal(t, resp, got)

	require.Equal(t, "start foo bar", out)
}

func TestNestedContextHTTPInterceptor(t *testing.T) {
	out := "start"
	foo := HTTPInterceptorFunc(func(req *http.Request, inner http.RoundTripper) (resp *http.Response, err error) {
		out += " " + req.Header.Get("X_TEST_PRE_FOO")
		resp, err = inner.RoundTrip(req)
		out += " " + resp.Header.Get("X_TEST_POST_FOO")
		return
	})
	bar := HTTPInterceptorFunc(func(req *http.Request, inner http.RoundTripper) (resp *http.Response, err error) {
		out += " " + req.Header.Get("X_TEST_PRE_BAR")
		resp, err = inner.RoundTrip(req)
		out += " " + resp.Header.Get("X_TEST_POST_BAR")
		return
	})

	ctx := context.Background()
	ctx = NewInterceptorContext(ctx, foo)
	ctx = NewInterceptorContext(ctx, bar)

	req, err := http.NewRequestWithContext(ctx, "GET", "test", bytes.NewReader([]byte{}))
	require.NoError(t, err)
	req.Header.Set("X_TEST_PRE_FOO", "pre-foo")
	req.Header.Set("X_TEST_PRE_BAR", "pre-bar")

	resp := &http.Response{Header: make(http.Header), Body: io.NopCloser(bytes.NewReader([]byte{})), StatusCode: 200}
	resp.Header.Set("X_TEST_POST_FOO", "post-foo")
	resp.Header.Set("X_TEST_POST_BAR", "post-bar")

	inner := &fakeHTTPTransport{resp: resp, err: nil}

	cl := &http.Client{Transport: InterceptorRoundTripper{Inner: inner}}
	got, err := cl.Do(req)
	require.NoError(t, err)
	_ = got.Body.Close()
	require.Equal(t, 200, got.StatusCode)

	require.Equal(t, "start pre-bar pre-foo post-foo post-bar", out)
}
