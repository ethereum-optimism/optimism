package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/proxyd"
	ms "github.com/ethereum-optimism/optimism/proxyd/tools/mockserver/handler"
	"github.com/stretchr/testify/require"
)

func TestConsensus(t *testing.T) {
	node1 := NewMockBackend(nil)
	defer node1.Close()
	node2 := NewMockBackend(nil)
	defer node2.Close()

	dir, err := os.Getwd()
	require.NoError(t, err)

	responses := path.Join(dir, "testdata/consensus_responses.yml")

	h1 := ms.MockedHandler{
		Overrides:    []*ms.MethodTemplate{},
		Autoload:     true,
		AutoloadFile: responses,
	}
	h2 := ms.MockedHandler{
		Overrides:    []*ms.MethodTemplate{},
		Autoload:     true,
		AutoloadFile: responses,
	}

	require.NoError(t, os.Setenv("NODE1_URL", node1.URL()))
	require.NoError(t, os.Setenv("NODE2_URL", node2.URL()))

	node1.SetHandler(http.HandlerFunc(h1.Handler))
	node2.SetHandler(http.HandlerFunc(h2.Handler))

	config := ReadConfig("consensus")
	ctx := context.Background()
	svr, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	client := NewProxydClient("http://127.0.0.1:8545")
	defer shutdown()

	bg := svr.BackendGroups["node"]
	require.NotNil(t, bg)
	require.NotNil(t, bg.Consensus)

	t.Run("initial consensus", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		// unknown consensus at init
		require.Equal(t, "0x0", bg.Consensus.GetConsensusBlockNumber().String())

		// first poll
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// consensus at block 0x1
		require.Equal(t, "0x1", bg.Consensus.GetConsensusBlockNumber().String())
	})

	t.Run("prevent using a backend with low peer count", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		h1.AddOverride(&ms.MethodTemplate{
			Method:   "net_peerCount",
			Block:    "",
			Response: buildPeerCountResponse(1),
		})

		be := backend(bg, "node1")
		require.NotNil(t, be)

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)
		consensusGroup := bg.Consensus.GetConsensusGroup()

		require.NotContains(t, consensusGroup, be)
		require.Equal(t, 1, len(consensusGroup))
	})

	t.Run("prevent using a backend lagging behind", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x1", "hash1"),
		})

		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x100", "hash0x100"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "0x100",
			Response: buildGetBlockResponse("0x100", "hash0x100"),
		})

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// since we ignored node1, the consensus should be at 0x100
		require.Equal(t, "0x100", bg.Consensus.GetConsensusBlockNumber().String())

		consensusGroup := bg.Consensus.GetConsensusGroup()

		be := backend(bg, "node1")
		require.NotNil(t, be)
		require.NotContains(t, consensusGroup, be)
		require.Equal(t, 1, len(consensusGroup))
	})

	t.Run("prevent using a backend not in sync", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		// advance latest on node2 to 0x2
		h1.AddOverride(&ms.MethodTemplate{
			Method: "eth_syncing",
			Block:  "",
			Response: buildResponse(map[string]string{
				"startingblock": "0x0",
				"currentblock":  "0x0",
				"highestblock":  "0x100",
			}),
		})

		be := backend(bg, "node1")
		require.NotNil(t, be)

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)
		consensusGroup := bg.Consensus.GetConsensusGroup()

		require.NotContains(t, consensusGroup, be)
		require.Equal(t, 1, len(consensusGroup))
	})

	t.Run("advance consensus", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// all nodes start at block 0x1
		require.Equal(t, "0x1", bg.Consensus.GetConsensusBlockNumber().String())

		// advance latest on node2 to 0x2
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x2", "hash2"),
		})

		// poll for group consensus
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}

		// consensus should stick to 0x1, since node1 is still lagging there
		bg.Consensus.UpdateBackendGroupConsensus(ctx)
		require.Equal(t, "0x1", bg.Consensus.GetConsensusBlockNumber().String())

		// advance latest on node1 to 0x2
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x2", "hash2"),
		})

		// poll for group consensus
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// should stick to 0x2, since now all nodes are at 0x2
		require.Equal(t, "0x2", bg.Consensus.GetConsensusBlockNumber().String())
	})

	t.Run("broken consensus", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// all nodes start at block 0x1
		require.Equal(t, "0x1", bg.Consensus.GetConsensusBlockNumber().String())

		// advance latest on both nodes to 0x2
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x2", "hash2"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x2", "hash2"),
		})

		// poll for group consensus
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// at 0x2
		require.Equal(t, "0x2", bg.Consensus.GetConsensusBlockNumber().String())

		// make node2 diverge on hash
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "0x2",
			Response: buildGetBlockResponse("0x2", "wrong_hash"),
		})

		// poll for group consensus
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// should resolve to 0x1, since 0x2 is out of consensus at the moment
		require.Equal(t, "0x1", bg.Consensus.GetConsensusBlockNumber().String())

		// later, when impl events, listen to broken consensus event
	})

	t.Run("broken consensus with depth 2", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// all nodes start at block 0x1
		require.Equal(t, "0x1", bg.Consensus.GetConsensusBlockNumber().String())

		// advance latest on both nodes to 0x2
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x2", "hash2"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x2", "hash2"),
		})

		// poll for group consensus
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// at 0x2
		require.Equal(t, "0x2", bg.Consensus.GetConsensusBlockNumber().String())

		// advance latest on both nodes to 0x3
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x3", "hash3"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x3", "hash3"),
		})

		// poll for group consensus
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// at 0x3
		require.Equal(t, "0x3", bg.Consensus.GetConsensusBlockNumber().String())

		// make node2 diverge on hash for blocks 0x2 and 0x3
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "0x2",
			Response: buildGetBlockResponse("0x2", "wrong_hash2"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "0x3",
			Response: buildGetBlockResponse("0x3", "wrong_hash3"),
		})

		// poll for group consensus
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// should resolve to 0x1
		require.Equal(t, "0x1", bg.Consensus.GetConsensusBlockNumber().String())
	})

	t.Run("fork in advanced block", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// all nodes start at block 0x1
		require.Equal(t, "0x1", bg.Consensus.GetConsensusBlockNumber().String())

		// make nodes 1 and 2 advance in forks
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "0x2",
			Response: buildGetBlockResponse("0x2", "node1_0x2"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "0x2",
			Response: buildGetBlockResponse("0x2", "node2_0x2"),
		})
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "0x3",
			Response: buildGetBlockResponse("0x3", "node1_0x3"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "0x3",
			Response: buildGetBlockResponse("0x3", "node2_0x3"),
		})
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x3", "node1_0x3"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x3", "node2_0x3"),
		})

		// poll for group consensus
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// should resolve to 0x1, the highest common ancestor
		require.Equal(t, "0x1", bg.Consensus.GetConsensusBlockNumber().String())
	})

	t.Run("load balancing should hit both backends", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		require.Equal(t, 2, len(bg.Consensus.GetConsensusGroup()))

		node1.Reset()
		node2.Reset()

		require.Equal(t, 0, len(node1.Requests()))
		require.Equal(t, 0, len(node2.Requests()))

		// there is a random component to this test,
		// since our round-robin implementation shuffles the ordering
		// to achieve uniform distribution

		// so we just make 100 requests per backend and expect the number of requests to be somewhat balanced
		// i.e. each backend should be hit minimally by at least 50% of the requests
		consensusGroup := bg.Consensus.GetConsensusGroup()

		numberReqs := len(consensusGroup) * 100
		for numberReqs > 0 {
			_, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x1", false})
			require.NoError(t, err)
			require.Equal(t, 200, statusCode)
			numberReqs--
		}

		msg := fmt.Sprintf("n1 %d, n2 %d", len(node1.Requests()), len(node2.Requests()))
		require.GreaterOrEqual(t, len(node1.Requests()), 50, msg)
		require.GreaterOrEqual(t, len(node2.Requests()), 50, msg)
	})

	t.Run("load balancing should not hit if node is not healthy", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		// node1 should not be serving any traffic
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "net_peerCount",
			Block:    "",
			Response: buildPeerCountResponse(1),
		})

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		require.Equal(t, 1, len(bg.Consensus.GetConsensusGroup()))

		node1.Reset()
		node2.Reset()

		require.Equal(t, 0, len(node1.Requests()))
		require.Equal(t, 0, len(node2.Requests()))

		numberReqs := 10
		for numberReqs > 0 {
			_, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x1", false})
			require.NoError(t, err)
			require.Equal(t, 200, statusCode)
			numberReqs--
		}

		msg := fmt.Sprintf("n1 %d, n2 %d", len(node1.Requests()), len(node2.Requests()))
		require.Equal(t, len(node1.Requests()), 0, msg)
		require.Equal(t, len(node2.Requests()), 10, msg)
	})

	t.Run("rewrite response of eth_blockNumber", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		node1.Reset()
		node2.Reset()
		bg.Consensus.Unban()

		// establish the consensus

		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x2", "hash2"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x2", "hash2"),
		})

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		totalRequests := len(node1.Requests()) + len(node2.Requests())

		require.Equal(t, 2, len(bg.Consensus.GetConsensusGroup()))

		// pretend backends advanced in consensus, but we are still serving the latest value of the consensus
		// until it gets updated again

		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x3", "hash3"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x3", "hash3"),
		})

		resRaw, statusCode, err := client.SendRPC("eth_blockNumber", nil)
		require.NoError(t, err)
		require.Equal(t, 200, statusCode)

		var jsonMap map[string]interface{}
		err = json.Unmarshal(resRaw, &jsonMap)
		require.NoError(t, err)
		require.Equal(t, "0x2", jsonMap["result"])

		// no extra request hit the backends
		require.Equal(t, totalRequests, len(node1.Requests())+len(node2.Requests()))
	})

	t.Run("rewrite request of eth_getBlockByNumber", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		// establish the consensus and ban node2 for now
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x2", "hash2"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "net_peerCount",
			Block:    "",
			Response: buildPeerCountResponse(1),
		})

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		require.Equal(t, 1, len(bg.Consensus.GetConsensusGroup()))

		node1.Reset()

		_, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"latest"})
		require.NoError(t, err)
		require.Equal(t, 200, statusCode)

		var jsonMap map[string]interface{}
		err = json.Unmarshal(node1.Requests()[0].Body, &jsonMap)
		require.NoError(t, err)
		require.Equal(t, "0x2", jsonMap["params"].([]interface{})[0])
	})

	t.Run("rewrite request of eth_getBlockByNumber - out of range", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		// establish the consensus and ban node2 for now
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x2", "hash2"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "net_peerCount",
			Block:    "",
			Response: buildPeerCountResponse(1),
		})

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		require.Equal(t, 1, len(bg.Consensus.GetConsensusGroup()))

		node1.Reset()

		resRaw, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x10"})
		require.NoError(t, err)
		require.Equal(t, 400, statusCode)

		var jsonMap map[string]interface{}
		err = json.Unmarshal(resRaw, &jsonMap)
		require.NoError(t, err)
		require.Equal(t, -32019, int(jsonMap["error"].(map[string]interface{})["code"].(float64)))
		require.Equal(t, "block is out of range", jsonMap["error"].(map[string]interface{})["message"])
	})

	t.Run("batched rewrite", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()
		bg.Consensus.Unban()

		// establish the consensus and ban node2 for now
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildGetBlockResponse("0x2", "hash2"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "net_peerCount",
			Block:    "",
			Response: buildPeerCountResponse(1),
		})

		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		require.Equal(t, 1, len(bg.Consensus.GetConsensusGroup()))

		node1.Reset()

		resRaw, statusCode, err := client.SendBatchRPC(
			NewRPCReq("1", "eth_getBlockByNumber", []interface{}{"latest"}),
			NewRPCReq("2", "eth_getBlockByNumber", []interface{}{"0x10"}),
			NewRPCReq("3", "eth_getBlockByNumber", []interface{}{"0x1"}))
		require.NoError(t, err)
		require.Equal(t, 200, statusCode)

		var jsonMap []map[string]interface{}
		err = json.Unmarshal(resRaw, &jsonMap)
		require.NoError(t, err)
		require.Equal(t, 3, len(jsonMap))

		// rewrite latest to 0x2
		require.Equal(t, "0x2", jsonMap[0]["result"].(map[string]interface{})["number"])

		// out of bounds for block 0x10
		require.Equal(t, -32019, int(jsonMap[1]["error"].(map[string]interface{})["code"].(float64)))
		require.Equal(t, "block is out of range", jsonMap[1]["error"].(map[string]interface{})["message"])

		// dont rewrite for 0x1
		require.Equal(t, "0x1", jsonMap[2]["result"].(map[string]interface{})["number"])
	})
}

func backend(bg *proxyd.BackendGroup, name string) *proxyd.Backend {
	for _, be := range bg.Backends {
		if be.Name == name {
			return be
		}
	}
	return nil
}

func buildPeerCountResponse(count uint64) string {
	return buildResponse(hexutil.Uint64(count).String())
}
func buildGetBlockResponse(number string, hash string) string {
	return buildResponse(map[string]string{
		"number": number,
		"hash":   hash,
	})
}

func buildResponse(result interface{}) string {
	res, err := json.Marshal(proxyd.RPCRes{
		Result: result,
	})
	if err != nil {
		panic(err)
	}
	return string(res)
}
