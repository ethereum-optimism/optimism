package integration_tests

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/stretchr/testify/require"
)

type resWithCode struct {
	code int
	res  []byte
}

const frontendOverLimitResponse = `{"error":{"code":-32016,"message":"over rate limit"},"id":null,"jsonrpc":"2.0"}`

func TestBackendMaxRPSLimit(t *testing.T) {
	goodBackend := NewMockBackend(BatchedResponseHandler(200, goodResponse))
	defer goodBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))

	config := ReadConfig("backend_rate_limit")
	client := NewProxydClient("http://127.0.0.1:8545")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	limitedRes, codes := spamReqs(t, client, 503)
	require.Equal(t, 2, codes[200])
	require.Equal(t, 1, codes[503])
	RequireEqualJSON(t, []byte(noBackendsResponse), limitedRes)
}

func TestFrontendMaxRPSLimit(t *testing.T) {
	goodBackend := NewMockBackend(BatchedResponseHandler(200, goodResponse))
	defer goodBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))

	config := ReadConfig("frontend_rate_limit")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	t.Run("non-exempt over limit", func(t *testing.T) {
		client := NewProxydClient("http://127.0.0.1:8545")
		limitedRes, codes := spamReqs(t, client, 429)
		require.Equal(t, 1, codes[429])
		require.Equal(t, 2, codes[200])
		RequireEqualJSON(t, []byte(frontendOverLimitResponse), limitedRes)
	})

	t.Run("exempt user agent over limit", func(t *testing.T) {
		h := make(http.Header)
		h.Set("User-Agent", "exempt_agent")
		client := NewProxydClientWithHeaders("http://127.0.0.1:8545", h)
		_, codes := spamReqs(t, client, 429)
		require.Equal(t, 3, codes[200])
	})

	t.Run("exempt origin over limit", func(t *testing.T) {
		h := make(http.Header)
		h.Set("Origin", "exempt_origin")
		client := NewProxydClientWithHeaders("http://127.0.0.1:8545", h)
		_, codes := spamReqs(t, client, 429)
		fmt.Println(codes)
		require.Equal(t, 3, codes[200])
	})

	t.Run("multiple xff", func(t *testing.T) {
		h1 := make(http.Header)
		h1.Set("X-Forwarded-For", "0.0.0.0")
		h2 := make(http.Header)
		h2.Set("X-Forwarded-For", "1.1.1.1")
		client1 := NewProxydClientWithHeaders("http://127.0.0.1:8545", h1)
		client2 := NewProxydClientWithHeaders("http://127.0.0.1:8545", h2)
		_, codes := spamReqs(t, client1, 429)
		require.Equal(t, 1, codes[429])
		require.Equal(t, 2, codes[200])
		_, code, err := client2.SendRPC("eth_chainId", nil)
		require.Equal(t, 200, code)
		require.NoError(t, err)
		time.Sleep(time.Second)
		_, code, err = client2.SendRPC("eth_chainId", nil)
		require.Equal(t, 200, code)
		require.NoError(t, err)
	})
}

func spamReqs(t *testing.T, client *ProxydHTTPClient, limCode int) ([]byte, map[int]int) {
	resCh := make(chan *resWithCode)
	for i := 0; i < 3; i++ {
		go func() {
			res, code, err := client.SendRPC("eth_chainId", nil)
			require.NoError(t, err)
			resCh <- &resWithCode{
				code: code,
				res:  res,
			}
		}()
	}

	codes := make(map[int]int)
	var limitedRes []byte
	for i := 0; i < 3; i++ {
		res := <-resCh
		code := res.code
		if codes[code] == 0 {
			codes[code] = 1
		} else {
			codes[code] += 1
		}

		if code == limCode {
			limitedRes = res.res
		}
	}

	return limitedRes, codes
}
