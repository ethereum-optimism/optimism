package integration_tests

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/stretchr/testify/require"
)

func TestMaxConcurrentRPCs(t *testing.T) {
	var (
		mu                sync.Mutex
		concurrentRPCs    int
		maxConcurrentRPCs int
	)
	handler := func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		concurrentRPCs++
		if maxConcurrentRPCs < concurrentRPCs {
			maxConcurrentRPCs = concurrentRPCs
		}
		mu.Unlock()

		time.Sleep(time.Second * 2)
		BatchedResponseHandler(200, goodResponse)(w, r)

		mu.Lock()
		concurrentRPCs--
		mu.Unlock()
	}
	// We don't use the MockBackend because it serializes requests to the handler
	slowBackend := httptest.NewServer(http.HandlerFunc(handler))
	defer slowBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", slowBackend.URL))

	config := ReadConfig("max_rpc_conns")
	client := NewProxydClient("http://127.0.0.1:8545")
	_, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	type resWithCodeErr struct {
		res  []byte
		code int
		err  error
	}
	resCh := make(chan *resWithCodeErr)
	for i := 0; i < 3; i++ {
		go func() {
			res, code, err := client.SendRPC("eth_chainId", nil)
			resCh <- &resWithCodeErr{
				res:  res,
				code: code,
				err:  err,
			}
		}()
	}
	res1 := <-resCh
	res2 := <-resCh
	res3 := <-resCh

	require.NoError(t, res1.err)
	require.NoError(t, res2.err)
	require.NoError(t, res3.err)
	require.Equal(t, 200, res1.code)
	require.Equal(t, 200, res2.code)
	require.Equal(t, 200, res3.code)
	RequireEqualJSON(t, []byte(goodResponse), res1.res)
	RequireEqualJSON(t, []byte(goodResponse), res2.res)
	RequireEqualJSON(t, []byte(goodResponse), res3.res)

	require.EqualValues(t, 2, maxConcurrentRPCs)
}
