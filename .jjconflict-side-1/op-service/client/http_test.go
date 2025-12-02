package client

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
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
