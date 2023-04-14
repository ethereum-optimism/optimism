package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientWithCookies(t *testing.T) {
	testcases := []struct {
		name    string
		cookies bool
	}{
		{
			name:    "cookies disabled",
			cookies: false,
		}, {
			name:    "cookies enabled",
			cookies: true,
		},
	}
	for _, test := range testcases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cookieSet := false
			cookieChecked := false
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cookie, err := r.Cookie("name")
				if test.cookies && cookieSet {
					assert.NoError(t, err)
					assert.Equal(t, "value", cookie.Value)
				} else {
					assert.ErrorIs(t, err, http.ErrNoCookie)
				}
				cookieChecked = cookieSet
				http.SetCookie(w, &http.Cookie{Name: "name", Value: "value"})
				cookieSet = true
				_, err = w.Write([]byte("{\"jsonrpc\":\"2.0\",\"id\":0,\"result\":\"0x5\"}"))
				assert.NoError(t, err)
			}))
			defer s.Close()

			ctx := context.Background()
			cfg := CLIConfig{
				Addr:    s.URL,
				Cookies: test.cookies,
			}
			client, err := NewClient(ctx, cfg)
			assert.NoError(t, err)
			for i := 0; i < 2; i++ {
				_, err = client.ChainID(ctx)
				assert.NoError(t, err)
			}
			assert.True(t, cookieSet)
			assert.True(t, cookieChecked)
		})
	}
}

func TestClientWithHeaders(t *testing.T) {
	testcases := []struct {
		name    string
		headers http.Header
	}{
		{
			name: "no headers",
		}, {
			name:    "accept header",
			headers: http.Header{"Accept": []string{"application/gzip"}},
		},
	}
	for _, test := range testcases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			headetChecked := false
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.headers != nil {
					for k, v := range test.headers {
						assert.Equal(t, v, r.Header.Values(k))
					}
				}
				headetChecked = true
				_, err := w.Write([]byte("{\"jsonrpc\":\"2.0\",\"id\":0,\"result\":\"0x5\"}"))
				assert.NoError(t, err)
			}))
			defer s.Close()

			ctx := context.Background()
			cfg := CLIConfig{
				Addr:    s.URL,
				Headers: test.headers,
			}
			client, err := NewClient(ctx, cfg)
			assert.NoError(t, err)
			_, err = client.ChainID(ctx)
			assert.NoError(t, err)
			assert.True(t, headetChecked)
		})
	}
}
