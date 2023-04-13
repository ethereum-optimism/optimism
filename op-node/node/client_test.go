package node

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
)

func TestL1EndpointConfigCookies(t *testing.T) {
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
				_, err = w.Write([]byte("{\"jsonrpc\":\"2.0\",\"id\":0,\"result\":\"\"}"))
				assert.NoError(t, err)
			}))
			defer s.Close()

			ctx := context.Background()
			l1 := L1EndpointConfig{
				L1NodeAddr: s.URL,
				Cookies:    test.cookies,
			}
			client, _, err := l1.Setup(ctx, log.New(ctx), &rollup.Config{})
			assert.NoError(t, err)
			for i := 0; i < 2; i++ {
				err = client.CallContext(ctx, new(string), "fake_method")
				assert.NoError(t, err)
			}
			assert.True(t, cookieSet)
			assert.True(t, cookieChecked)
		})
	}
}
