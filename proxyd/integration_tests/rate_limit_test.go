package integration_tests

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/stretchr/testify/require"
)

type resWithCode struct {
	code int
	res  []byte
}

func TestMaxRPSLimit(t *testing.T) {
	goodBackend := NewMockBackend(BatchedResponseHandler(200, goodResponse))
	defer goodBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))

	config := ReadConfig("rate_limit")
	client := NewProxydClient("http://127.0.0.1:8545")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

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

		// 503 because there's only one backend available
		if code == 503 {
			limitedRes = res.res
		}
	}

	require.Equal(t, 2, codes[200])
	require.Equal(t, 1, codes[503])
	RequireEqualJSON(t, []byte(noBackendsResponse), limitedRes)
}
