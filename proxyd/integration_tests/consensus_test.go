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

type nodeContext struct {
	mockBackend *MockBackend
	backend     *proxyd.Backend
	handler     *ms.MockedHandler
}

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

	// convenient mapping to access the nodes by name
	nodes := map[string]nodeContext{
		"node1": {
			mockBackend: node1,
			backend:     bg.Backends[0],
			handler:     &h1,
		},
		"node2": {
			mockBackend: node2,
			backend:     bg.Backends[1],
			handler:     &h2,
		},
	}

	reset := func() {
		for _, node := range nodes {
			node.handler.ResetOverrides()
			node.mockBackend.Reset()
		}
		bg.Consensus.ClearListeners()
		bg.Consensus.Reset()
	}

	// poll for updated consensus
	update := func() {
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)
	}

	override := func(node string, method string, block string, response string) {
		nodes[node].handler.AddOverride(&ms.MethodTemplate{
			Method:   method,
			Block:    block,
			Response: response,
		})
	}

	overrideBlock := func(node string, blockRequest string, blockResponse string) {
		override(node,
			"eth_getBlockByNumber",
			blockRequest,
			buildResponse(map[string]string{
				"number": blockResponse,
				"hash":   "hash_" + blockResponse,
			}))
	}

	overrideBlockHash := func(node string, blockRequest string, number string, hash string) {
		override(node,
			"eth_getBlockByNumber",
			blockRequest,
			buildResponse(map[string]string{
				"number": number,
				"hash":   hash,
			}))
	}

	overridePeerCount := func(node string, count int) {
		override(node, "net_peerCount", "", buildResponse(hexutil.Uint64(count).String()))
	}

	overrideNotInSync := func(node string) {
		override(node, "eth_syncing", "", buildResponse(map[string]string{
			"startingblock": "0x0",
			"currentblock":  "0x0",
			"highestblock":  "0x100",
		}))
	}

	useOnlyNode1 := func() {
		overridePeerCount("node2", 0)
		update()

		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.Equal(t, 1, len(consensusGroup))
		require.Contains(t, consensusGroup, nodes["node1"].backend)
		node1.Reset()
	}

	t.Run("initial consensus", func(t *testing.T) {
		reset()

		// unknown consensus at init
		require.Equal(t, "0x0", bg.Consensus.GetLatestBlockNumber().String())

		// first poll
		update()

		// as a default we use:
		// - latest at 0x101 [257]
		// - safe at 0xe1 [225]
		// - finalized at 0xc1 [193]

		// consensus at block 0x101
		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())
		require.Equal(t, "0xe1", bg.Consensus.GetSafeBlockNumber().String())
		require.Equal(t, "0xc1", bg.Consensus.GetFinalizedBlockNumber().String())
	})

	t.Run("prevent using a backend with low peer count", func(t *testing.T) {
		reset()
		overridePeerCount("node1", 0)
		update()

		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.NotContains(t, consensusGroup, nodes["node1"].backend)
		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 1, len(consensusGroup))
	})

	t.Run("prevent using a backend lagging behind", func(t *testing.T) {
		reset()
		// node2 is 8+1 blocks ahead of node1 (0x101 + 8+1 = 0x10a)
		overrideBlock("node2", "latest", "0x10a")
		update()

		// since we ignored node1, the consensus should be at 0x10a
		require.Equal(t, "0x10a", bg.Consensus.GetLatestBlockNumber().String())
		require.Equal(t, "0xe1", bg.Consensus.GetSafeBlockNumber().String())
		require.Equal(t, "0xc1", bg.Consensus.GetFinalizedBlockNumber().String())

		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.NotContains(t, consensusGroup, nodes["node1"].backend)
		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 1, len(consensusGroup))
	})

	t.Run("prevent using a backend lagging behind - one before limit", func(t *testing.T) {
		reset()
		// node2 is 8 blocks ahead of node1 (0x101 + 8 = 0x109)
		overrideBlock("node2", "latest", "0x109")
		update()

		// both nodes are in consensus with the lowest block
		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())
		require.Equal(t, "0xe1", bg.Consensus.GetSafeBlockNumber().String())
		require.Equal(t, "0xc1", bg.Consensus.GetFinalizedBlockNumber().String())
		require.Equal(t, 2, len(bg.Consensus.GetConsensusGroup()))
	})

	t.Run("prevent using a backend not in sync", func(t *testing.T) {
		reset()
		// make node1 not in sync
		overrideNotInSync("node1")
		update()

		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.NotContains(t, consensusGroup, nodes["node1"].backend)
		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 1, len(consensusGroup))
	})

	t.Run("advance consensus", func(t *testing.T) {
		reset()

		// as a default we use:
		// - latest at 0x101 [257]
		// - safe at 0xe1 [225]
		// - finalized at 0xc1 [193]

		update()

		// all nodes start at block 0x101
		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())

		// advance latest on node2 to 0x102
		overrideBlock("node2", "latest", "0x102")

		update()

		// consensus should stick to 0x101, since node1 is still lagging there
		bg.Consensus.UpdateBackendGroupConsensus(ctx)
		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())

		// advance latest on node1 to 0x102
		overrideBlock("node1", "latest", "0x102")

		update()

		// all nodes now at 0x102
		require.Equal(t, "0x102", bg.Consensus.GetLatestBlockNumber().String())
	})

	t.Run("should use lowest safe and finalized", func(t *testing.T) {
		reset()
		overrideBlock("node2", "finalized", "0xc2")
		overrideBlock("node2", "safe", "0xe2")
		update()

		require.Equal(t, "0xc1", bg.Consensus.GetFinalizedBlockNumber().String())
		require.Equal(t, "0xe1", bg.Consensus.GetSafeBlockNumber().String())
	})

	t.Run("advance safe and finalized", func(t *testing.T) {
		reset()
		overrideBlock("node1", "finalized", "0xc2")
		overrideBlock("node1", "safe", "0xe2")
		overrideBlock("node2", "finalized", "0xc2")
		overrideBlock("node2", "safe", "0xe2")
		update()

		require.Equal(t, "0xc2", bg.Consensus.GetFinalizedBlockNumber().String())
		require.Equal(t, "0xe2", bg.Consensus.GetSafeBlockNumber().String())
	})

	t.Run("ban backend if tags are messed - safe < finalized", func(t *testing.T) {
		reset()
		overrideBlock("node1", "finalized", "0xb1")
		overrideBlock("node1", "safe", "0xa1")
		update()

		require.Equal(t, "0xc1", bg.Consensus.GetFinalizedBlockNumber().String())
		require.Equal(t, "0xe1", bg.Consensus.GetSafeBlockNumber().String())

		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.NotContains(t, consensusGroup, nodes["node1"].backend)
		require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 1, len(consensusGroup))
	})

	t.Run("ban backend if tags are messed - latest < safe", func(t *testing.T) {
		reset()
		overrideBlock("node1", "safe", "0xb1")
		overrideBlock("node1", "latest", "0xa1")
		update()

		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())
		require.Equal(t, "0xc1", bg.Consensus.GetFinalizedBlockNumber().String())
		require.Equal(t, "0xe1", bg.Consensus.GetSafeBlockNumber().String())

		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.NotContains(t, consensusGroup, nodes["node1"].backend)
		require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 1, len(consensusGroup))
	})

	t.Run("ban backend if tags are messed - safe dropped", func(t *testing.T) {
		reset()
		update()
		overrideBlock("node1", "safe", "0xb1")
		update()

		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())
		require.Equal(t, "0xc1", bg.Consensus.GetFinalizedBlockNumber().String())
		require.Equal(t, "0xe1", bg.Consensus.GetSafeBlockNumber().String())

		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.NotContains(t, consensusGroup, nodes["node1"].backend)
		require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 1, len(consensusGroup))
	})

	t.Run("ban backend if tags are messed - finalized dropped", func(t *testing.T) {
		reset()
		update()
		overrideBlock("node1", "finalized", "0xa1")
		update()

		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())
		require.Equal(t, "0xc1", bg.Consensus.GetFinalizedBlockNumber().String())
		require.Equal(t, "0xe1", bg.Consensus.GetSafeBlockNumber().String())

		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.NotContains(t, consensusGroup, nodes["node1"].backend)
		require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 1, len(consensusGroup))
	})

	t.Run("recover after safe and finalized dropped", func(t *testing.T) {
		reset()
		useOnlyNode1()
		overrideBlock("node1", "latest", "0xd1")
		overrideBlock("node1", "safe", "0xb1")
		overrideBlock("node1", "finalized", "0x91")
		update()

		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.NotContains(t, consensusGroup, nodes["node1"].backend)
		require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 0, len(consensusGroup))

		// unban and see if it recovers
		bg.Consensus.Unban(nodes["node1"].backend)
		update()

		consensusGroup = bg.Consensus.GetConsensusGroup()
		require.Contains(t, consensusGroup, nodes["node1"].backend)
		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 1, len(consensusGroup))

		require.Equal(t, "0xd1", bg.Consensus.GetLatestBlockNumber().String())
		require.Equal(t, "0x91", bg.Consensus.GetFinalizedBlockNumber().String())
		require.Equal(t, "0xb1", bg.Consensus.GetSafeBlockNumber().String())
	})

	t.Run("latest dropped below safe, then recovered", func(t *testing.T) {
		reset()
		useOnlyNode1()
		overrideBlock("node1", "latest", "0xd1")
		update()

		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.NotContains(t, consensusGroup, nodes["node1"].backend)
		require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 0, len(consensusGroup))

		// unban and see if it recovers
		bg.Consensus.Unban(nodes["node1"].backend)
		overrideBlock("node1", "safe", "0xb1")
		overrideBlock("node1", "finalized", "0x91")
		update()

		consensusGroup = bg.Consensus.GetConsensusGroup()
		require.Contains(t, consensusGroup, nodes["node1"].backend)
		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 1, len(consensusGroup))

		require.Equal(t, "0xd1", bg.Consensus.GetLatestBlockNumber().String())
		require.Equal(t, "0x91", bg.Consensus.GetFinalizedBlockNumber().String())
		require.Equal(t, "0xb1", bg.Consensus.GetSafeBlockNumber().String())
	})

	t.Run("latest dropped below safe, and stayed inconsistent after ban", func(t *testing.T) {
		reset()
		useOnlyNode1()
		overrideBlock("node1", "latest", "0xd1")
		update()

		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.NotContains(t, consensusGroup, nodes["node1"].backend)
		require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 0, len(consensusGroup))

		// unban and see if it recovers - it should not since the blocks stays the same
		bg.Consensus.Unban(nodes["node1"].backend)
		update()

		// should be banned again
		consensusGroup = bg.Consensus.GetConsensusGroup()
		require.NotContains(t, consensusGroup, nodes["node1"].backend)
		require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.Equal(t, 0, len(consensusGroup))
	})

	t.Run("broken consensus", func(t *testing.T) {
		reset()
		listenerCalled := false
		bg.Consensus.AddListener(func() {
			listenerCalled = true
		})
		update()

		// all nodes start at block 0x101
		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())

		// advance latest on both nodes to 0x102
		overrideBlock("node1", "latest", "0x102")
		overrideBlock("node2", "latest", "0x102")

		update()

		// at 0x102
		require.Equal(t, "0x102", bg.Consensus.GetLatestBlockNumber().String())

		// make node2 diverge on hash
		overrideBlockHash("node2", "0x102", "0x102", "wrong_hash")

		update()

		// should resolve to 0x101, since 0x102 is out of consensus at the moment
		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())

		// everybody serving traffic
		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.Equal(t, 2, len(consensusGroup))
		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))

		// onConsensusBroken listener was called
		require.True(t, listenerCalled)
	})

	t.Run("broken consensus with depth 2", func(t *testing.T) {
		reset()
		listenerCalled := false
		bg.Consensus.AddListener(func() {
			listenerCalled = true
		})
		update()

		// all nodes start at block 0x101
		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())

		// advance latest on both nodes to 0x102
		overrideBlock("node1", "latest", "0x102")
		overrideBlock("node2", "latest", "0x102")

		update()

		// at 0x102
		require.Equal(t, "0x102", bg.Consensus.GetLatestBlockNumber().String())

		// advance latest on both nodes to 0x3
		overrideBlock("node1", "latest", "0x103")
		overrideBlock("node2", "latest", "0x103")

		update()

		// at 0x103
		require.Equal(t, "0x103", bg.Consensus.GetLatestBlockNumber().String())

		// make node2 diverge on hash for blocks 0x102 and 0x103
		overrideBlockHash("node2", "0x102", "0x102", "wrong_hash_0x102")
		overrideBlockHash("node2", "0x103", "0x103", "wrong_hash_0x103")

		update()

		// should resolve to 0x101
		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())

		// everybody serving traffic
		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.Equal(t, 2, len(consensusGroup))
		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))

		// onConsensusBroken listener was called
		require.True(t, listenerCalled)
	})

	t.Run("fork in advanced block", func(t *testing.T) {
		reset()
		listenerCalled := false
		bg.Consensus.AddListener(func() {
			listenerCalled = true
		})
		update()

		// all nodes start at block 0x101
		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())

		// make nodes 1 and 2 advance in forks, i.e. they have same block number with different hashes
		overrideBlockHash("node1", "0x102", "0x102", "node1_0x102")
		overrideBlockHash("node2", "0x102", "0x102", "node2_0x102")
		overrideBlockHash("node1", "0x103", "0x103", "node1_0x103")
		overrideBlockHash("node2", "0x103", "0x103", "node2_0x103")
		overrideBlockHash("node1", "latest", "0x103", "node1_0x103")
		overrideBlockHash("node2", "latest", "0x103", "node2_0x103")

		update()

		// should resolve to 0x101, the highest common ancestor
		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())

		// everybody serving traffic
		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.Equal(t, 2, len(consensusGroup))
		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))

		// onConsensusBroken listener should not be called
		require.False(t, listenerCalled)
	})

	t.Run("load balancing should hit both backends", func(t *testing.T) {
		reset()
		update()

		require.Equal(t, 2, len(bg.Consensus.GetConsensusGroup()))

		// reset request counts
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
			_, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x101", false})
			require.NoError(t, err)
			require.Equal(t, 200, statusCode)
			numberReqs--
		}

		msg := fmt.Sprintf("n1 %d, n2 %d", len(node1.Requests()), len(node2.Requests()))
		require.GreaterOrEqual(t, len(node1.Requests()), 50, msg)
		require.GreaterOrEqual(t, len(node2.Requests()), 50, msg)
	})

	t.Run("load balancing should not hit if node is not healthy", func(t *testing.T) {
		reset()
		useOnlyNode1()

		// reset request counts
		node1.Reset()
		node2.Reset()

		require.Equal(t, 0, len(node1.Requests()))
		require.Equal(t, 0, len(node2.Requests()))

		numberReqs := 10
		for numberReqs > 0 {
			_, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x101", false})
			require.NoError(t, err)
			require.Equal(t, 200, statusCode)
			numberReqs--
		}

		msg := fmt.Sprintf("n1 %d, n2 %d", len(node1.Requests()), len(node2.Requests()))
		require.Equal(t, len(node1.Requests()), 10, msg)
		require.Equal(t, len(node2.Requests()), 0, msg)
	})

	t.Run("rewrite response of eth_blockNumber", func(t *testing.T) {
		reset()
		update()

		totalRequests := len(node1.Requests()) + len(node2.Requests())
		require.Equal(t, 2, len(bg.Consensus.GetConsensusGroup()))

		resRaw, statusCode, err := client.SendRPC("eth_blockNumber", nil)
		require.NoError(t, err)
		require.Equal(t, 200, statusCode)

		var jsonMap map[string]interface{}
		err = json.Unmarshal(resRaw, &jsonMap)
		require.NoError(t, err)
		require.Equal(t, "0x101", jsonMap["result"])

		// no extra request hit the backends
		require.Equal(t, totalRequests, len(node1.Requests())+len(node2.Requests()))
	})

	t.Run("rewrite request of eth_getBlockByNumber for latest", func(t *testing.T) {
		reset()
		useOnlyNode1()

		_, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"latest"})
		require.NoError(t, err)
		require.Equal(t, 200, statusCode)

		var jsonMap map[string]interface{}
		err = json.Unmarshal(node1.Requests()[0].Body, &jsonMap)
		require.NoError(t, err)
		require.Equal(t, "0x101", jsonMap["params"].([]interface{})[0])
	})

	t.Run("rewrite request of eth_getBlockByNumber for finalized", func(t *testing.T) {
		reset()
		useOnlyNode1()

		_, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"finalized"})
		require.NoError(t, err)
		require.Equal(t, 200, statusCode)

		var jsonMap map[string]interface{}
		err = json.Unmarshal(node1.Requests()[0].Body, &jsonMap)
		require.NoError(t, err)
		require.Equal(t, "0xc1", jsonMap["params"].([]interface{})[0])
	})

	t.Run("rewrite request of eth_getBlockByNumber for safe", func(t *testing.T) {
		reset()
		useOnlyNode1()

		_, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"safe"})
		require.NoError(t, err)
		require.Equal(t, 200, statusCode)

		var jsonMap map[string]interface{}
		err = json.Unmarshal(node1.Requests()[0].Body, &jsonMap)
		require.NoError(t, err)
		require.Equal(t, "0xe1", jsonMap["params"].([]interface{})[0])
	})

	t.Run("rewrite request of eth_getBlockByNumber - out of range", func(t *testing.T) {
		reset()
		useOnlyNode1()

		resRaw, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x300"})
		require.NoError(t, err)
		require.Equal(t, 400, statusCode)

		var jsonMap map[string]interface{}
		err = json.Unmarshal(resRaw, &jsonMap)
		require.NoError(t, err)
		require.Equal(t, -32019, int(jsonMap["error"].(map[string]interface{})["code"].(float64)))
		require.Equal(t, "block is out of range", jsonMap["error"].(map[string]interface{})["message"])
	})

	t.Run("batched rewrite", func(t *testing.T) {
		reset()
		useOnlyNode1()

		resRaw, statusCode, err := client.SendBatchRPC(
			NewRPCReq("1", "eth_getBlockByNumber", []interface{}{"latest"}),
			NewRPCReq("2", "eth_getBlockByNumber", []interface{}{"0x102"}),
			NewRPCReq("3", "eth_getBlockByNumber", []interface{}{"0xe1"}))
		require.NoError(t, err)
		require.Equal(t, 200, statusCode)

		var jsonMap []map[string]interface{}
		err = json.Unmarshal(resRaw, &jsonMap)
		require.NoError(t, err)
		require.Equal(t, 3, len(jsonMap))

		// rewrite latest to 0x101
		require.Equal(t, "0x101", jsonMap[0]["result"].(map[string]interface{})["number"])

		// out of bounds for block 0x102
		require.Equal(t, -32019, int(jsonMap[1]["error"].(map[string]interface{})["code"].(float64)))
		require.Equal(t, "block is out of range", jsonMap[1]["error"].(map[string]interface{})["message"])

		// dont rewrite for 0xe1
		require.Equal(t, "0xe1", jsonMap[2]["result"].(map[string]interface{})["number"])
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
