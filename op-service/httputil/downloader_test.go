package httputil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloader_Download(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "test")
		}))
		t.Cleanup(srv.Close)

		d := new(Downloader)
		out := new(bytes.Buffer)
		err := d.Download(context.Background(), srv.URL, out)
		require.NoError(t, err)
		require.Equal(t, "test", out.String())
	})

	t.Run("above max size", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "test")
		}))
		t.Cleanup(srv.Close)

		d := &Downloader{
			MaxSize: 2,
		}
		out := new(bytes.Buffer)
		err := d.Download(context.Background(), srv.URL, out)
		require.ErrorContains(t, err, "exceeds maximum allowed size")
	})

	t.Run("above max size with fake content length", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Header needs to come before WriteHeader otherwise it will be automatically corrected.
			w.Header().Set("Content-Length", "1")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "test")
		}))
		t.Cleanup(srv.Close)

		d := &Downloader{
			MaxSize: 2,
		}
		out := new(bytes.Buffer)
		err := d.Download(context.Background(), srv.URL, out)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})
}
