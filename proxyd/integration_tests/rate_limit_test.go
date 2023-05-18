package integration_tests

import (
	"encoding/json"
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

const frontendOverLimitResponse = `{"error":{"code":-32016,"message":"over rate limit with special message"},"id":null,"jsonrpc":"2.0"}`
const frontendOverLimitResponseWithID = `{"error":{"code":-32016,"message":"over rate limit with special message"},"id":999,"jsonrpc":"2.0"}`

var ethChainID = "eth_chainId"

func TestFrontendMaxRPSLimit(t *testing.T) {
	goodBackend := NewMockBackend(BatchedResponseHandler(200, goodResponse))
	defer goodBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))

	config := ReadConfig("frontend_rate_limit")
	_, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	t.Run("non-exempt over limit", func(t *testing.T) {
		client := NewProxydClient("http://127.0.0.1:8545")
		limitedRes, codes := spamReqs(t, client, ethChainID, 429, 3)
		require.Equal(t, 1, codes[429])
		require.Equal(t, 2, codes[200])
		RequireEqualJSON(t, []byte(frontendOverLimitResponse), limitedRes)
	})

	t.Run("exempt user agent over limit", func(t *testing.T) {
		h := make(http.Header)
		h.Set("User-Agent", "exempt_agent")
		client := NewProxydClientWithHeaders("http://127.0.0.1:8545", h)
		_, codes := spamReqs(t, client, ethChainID, 429, 3)
		require.Equal(t, 3, codes[200])
	})

	t.Run("exempt origin over limit", func(t *testing.T) {
		h := make(http.Header)
		h.Set("Origin", "exempt_origin")
		client := NewProxydClientWithHeaders("http://127.0.0.1:8545", h)
		_, codes := spamReqs(t, client, ethChainID, 429, 3)
		require.Equal(t, 3, codes[200])
	})

	t.Run("multiple xff", func(t *testing.T) {
		h1 := make(http.Header)
		h1.Set("X-Forwarded-For", "0.0.0.0")
		h2 := make(http.Header)
		h2.Set("X-Forwarded-For", "1.1.1.1")
		client1 := NewProxydClientWithHeaders("http://127.0.0.1:8545", h1)
		client2 := NewProxydClientWithHeaders("http://127.0.0.1:8545", h2)
		_, codes := spamReqs(t, client1, ethChainID, 429, 3)
		require.Equal(t, 1, codes[429])
		require.Equal(t, 2, codes[200])
		_, code, err := client2.SendRPC(ethChainID, nil)
		require.Equal(t, 200, code)
		require.NoError(t, err)
		time.Sleep(time.Second)
		_, code, err = client2.SendRPC(ethChainID, nil)
		require.Equal(t, 200, code)
		require.NoError(t, err)
	})

	time.Sleep(time.Second)

	t.Run("RPC override", func(t *testing.T) {
		client := NewProxydClient("http://127.0.0.1:8545")
		limitedRes, codes := spamReqs(t, client, "eth_foobar", 429, 2)
		// use 2 and 1 here since the limit for eth_foobar is 1
		require.Equal(t, 1, codes[429])
		require.Equal(t, 1, codes[200])
		RequireEqualJSON(t, []byte(frontendOverLimitResponseWithID), limitedRes)
	})

	time.Sleep(time.Second)

	t.Run("RPC override in batch", func(t *testing.T) {
		client := NewProxydClient("http://127.0.0.1:8545")
		req := NewRPCReq("123", "eth_foobar", nil)
		out, code, err := client.SendBatchRPC(req, req, req)
		require.NoError(t, err)
		var res []proxyd.RPCRes
		require.NoError(t, json.Unmarshal(out, &res))

		expCode := proxyd.ErrOverRateLimit.Code
		require.Equal(t, 200, code)
		require.Equal(t, 3, len(res))
		require.Nil(t, res[0].Error)
		require.Equal(t, expCode, res[1].Error.Code)
		require.Equal(t, expCode, res[2].Error.Code)
	})

	time.Sleep(time.Second)

	t.Run("RPC override in batch exempt", func(t *testing.T) {
		h := make(http.Header)
		h.Set("User-Agent", "exempt_agent")
		client := NewProxydClientWithHeaders("http://127.0.0.1:8545", h)
		req := NewRPCReq("123", "eth_foobar", nil)
		out, code, err := client.SendBatchRPC(req, req, req)
		require.NoError(t, err)
		var res []proxyd.RPCRes
		require.NoError(t, json.Unmarshal(out, &res))

		require.Equal(t, 200, code)
		require.Equal(t, 3, len(res))
		require.Nil(t, res[0].Error)
		require.Nil(t, res[1].Error)
		require.Nil(t, res[2].Error)
	})

	time.Sleep(time.Second)

	t.Run("global RPC override", func(t *testing.T) {
		h := make(http.Header)
		h.Set("User-Agent", "exempt_agent")
		client := NewProxydClientWithHeaders("http://127.0.0.1:8545", h)
		limitedRes, codes := spamReqs(t, client, "eth_baz", 429, 2)
		// use 1 and 1 here since the limit for eth_baz is 1
		require.Equal(t, 1, codes[429])
		require.Equal(t, 1, codes[200])
		RequireEqualJSON(t, []byte(frontendOverLimitResponseWithID), limitedRes)
	})
}

func spamReqs(t *testing.T, client *ProxydHTTPClient, method string, limCode int, n int) ([]byte, map[int]int) {
	resCh := make(chan *resWithCode)
	for i := 0; i < n; i++ {
		go func() {
			res, code, err := client.SendRPC(method, nil)
			require.NoError(t, err)
			resCh <- &resWithCode{
				code: code,
				res:  res,
			}
		}()
	}

	codes := make(map[int]int)
	var limitedRes []byte
	for i := 0; i < n; i++ {
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
