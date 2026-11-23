package client

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/stretchr/testify/require"
)

func TestBasicHTTPClient(t *testing.T) {
	called := make(chan *http.Request, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		called <- r
	}))
	defer ts.Close()

	// generate deep clones
	mkhdr := func() http.Header {
		return http.Header{
			"Foo":        []string{"bar", "baz"},
			"Superchain": []string{"op"},
		}
	}
	opt := WithHeader(mkhdr())
	c := NewBasicHTTPClient(ts.URL, testlog.Logger(t, slog.LevelInfo), opt)

	const ep = "/api/version"
	query := url.Values{
		"key": []string{"123"},
	}
	getheader := http.Header{
		"Fruits":     []string{"apple"},
		"Superchain": []string{"base"},
	}
	resp, err := c.Get(context.Background(), ep, query, getheader)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	req := <-called
	require.Equal(t, ep, req.URL.Path)
	require.Equal(t, query, req.URL.Query())
	require.ElementsMatch(t, req.Header.Values("Foo"), mkhdr()["Foo"])
	require.ElementsMatch(t, req.Header.Values("Fruits"), getheader["Fruits"])
	require.ElementsMatch(t, req.Header.Values("Superchain"), []string{"op", "base"})
}

func TestBasicHTTPClientResponseLimit(t *testing.T) {
	const limit = 1 << 10
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		size, err := strconv.Atoi(r.URL.Query().Get("size"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusInternalServerError)
		}
		_, _ = w.Write(make([]byte, size))
	}))
	defer ts.Close()
	c := NewBasicHTTPClient(ts.URL, testlog.Logger(t, slog.LevelInfo))
	c.maxResponseSize = limit

	get := func(path string, size int) ([]byte, error) {
		resp, err := c.Get(context.Background(), path, url.Values{"size": {strconv.Itoa(size)}}, nil)
		require.NoError(t, err)
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}

	body, err := get("/ok", limit)
	require.NoError(t, err)
	require.Equal(t, limit, len(body))

	_, err = get("/ok", limit+1)
	require.ErrorIs(t, err, ErrResponseTooLarge)

	body, err = get("/fail", 4*maxErrorBodySize)
	require.NoError(t, err)
	require.Equal(t, maxErrorBodySize, len(body))
}

func TestAddHTTPHeaders(t *testing.T) {
	for _, test := range []struct {
		desc      string
		expheader http.Header
		headers   []http.Header
	}{
		{
			desc:      "all-empty",
			expheader: http.Header{},
			headers:   nil,
		},
		{
			desc:      "1-header-and-nils",
			expheader: http.Header{"Foo": []string{"bar"}},
			headers: []http.Header{
				nil,
				{"Foo": []string{"bar"}},
				nil,
			},
		},
		{
			desc: "2-headers",
			expheader: http.Header{
				"Foo":   []string{"bar", "baz"},
				"Super": []string{"chain"},
				"Fruit": []string{"apple"},
			},
			headers: []http.Header{
				{
					"Foo":   []string{"bar"},
					"Super": []string{"chain"},
				},
				{
					"Foo":   []string{"baz"},
					"Fruit": []string{"apple"},
				},
			},
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			h := make(http.Header)
			addHTTPHeaders(h, test.headers...)
			require.Equal(t, test.expheader, h)
		})
	}
}
