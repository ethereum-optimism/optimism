package integration_tests

import (
	"context"
	"time"

	// "encoding/json"
	// "fmt"
	"net/http"
	"os"
	"path"
	"testing"

	// "time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/proxyd"
	ms "github.com/ethereum-optimism/optimism/proxyd/tools/mockserver/handler"
	"github.com/stretchr/testify/require"
)

// type nodeContext struct {
// 	backend     *proxyd.Backend   // this is the actual backend impl in proxyd
// 	mockBackend *MockBackend      // this is the fake backend that we can use to mock responses
// 	handler     *ms.MockedHandler // this is where we control the state of mocked responses
// }

func setupBlockHeightZero(t *testing.T) (map[string]nodeContext, *proxyd.BackendGroup, *ProxydHTTPClient, func()) {
	// setup mock servers
	node1 := NewMockBackend(nil)
	node2 := NewMockBackend(nil)

	dir, err := os.Getwd()
	require.NoError(t, err)

	responses := path.Join(dir, "testdata/block_height_zero_responses.yml")

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

	// setup proxyd
	config := ReadConfig("block_height_zero")
	svr, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)

	// expose the proxyd client
	client := NewProxydClient("http://127.0.0.1:8545")

	// expose the backend group
	bg := svr.BackendGroups["node"]
	require.NotNil(t, bg)
	require.NotNil(t, bg.Consensus)
	require.Equal(t, 2, len(bg.Backends)) // should match config

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

	return nodes, bg, client, shutdown
}

func TestBlockHeightZero(t *testing.T) {
	nodes, bg, _, shutdown := setupBlockHeightZero(t)
	defer nodes["node1"].mockBackend.Close()
	defer nodes["node2"].mockBackend.Close()
	defer shutdown()

	ctx := context.Background()

	// poll for updated consensus
	update := func() {
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)
	}

	// convenient methods to manipulate state and mock responses
	reset := func() {
		for _, node := range nodes {
			node.handler.ResetOverrides()
			node.mockBackend.Reset()
		}
		bg.Consensus.ClearListeners()
		bg.Consensus.Reset()
	}

	override := func(node string, method string, block string, response string) {
		if _, ok := nodes[node]; !ok {
			t.Fatalf("node %s does not exist in the nodes map", node)
		}
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

	// overrideBlockHash := func(node string, blockRequest string, number string, hash string) {
	// 	override(node,
	// 		"eth_getBlockByNumber",
	// 		blockRequest,
	// 		buildResponse(map[string]string{
	// 			"number": number,
	// 			"hash":   hash,
	// 		}))
	// }

	overridePeerCount := func(node string, count int) {
		override(node, "net_peerCount", "", buildResponse(hexutil.Uint64(count).String()))
	}

	// overrideNotInSync := func(node string) {
	// 	override(node, "eth_syncing", "", buildResponse(map[string]string{
	// 		"startingblock": "0x0",
	// 		"currentblock":  "0x0",
	// 		"highestblock":  "0x100",
	// 	}))
	// }

	// force ban node2 and make sure node1 is the only one in consensus
	// useOnlyNode1 := func() {
	// 	overridePeerCount("node2", 0)
	// 	update()
	//
	// 	consensusGroup := bg.Consensus.GetConsensusGroup()
	// 	require.Equal(t, 1, len(consensusGroup))
	// 	require.Contains(t, consensusGroup, nodes["node1"].backend)
	// 	nodes["node1"].mockBackend.Reset()
	// }
	//
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

	t.Run("Test Backend BlockHeight Zero Activates at after 5 infratctions", func(t *testing.T) {
		reset()
		overrideBlock("node1", "latest", "0x0")
		for i := 0; i < 6; i++ {
			require.Equal(t, uint(i), nodes["node1"].backend.GetBlockHeightZeroSlidingWindowCount())
			require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
			require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))
			update()
		}
		require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))
	})

	t.Run("Test Backend BlockHeight Zero Does not activates if only four infractions in window", func(t *testing.T) {
		reset()

		delay := nodes["node2"].backend.GetBlockHeightZeroSlidingWindowLength() / 2
		overrideBlock("node2", "latest", "0x0")
		for i := 0; i < 10; i++ {
			// require.Equal(t, uint(i), nodes["node2"].backend.GetBlockHeightZeroSlidingWindowCount())
			require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend), "Expected Node2 Not to be Banned. NOTE: THIS PASS WILL NOT PASS IN DEBUG")
			require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend), "Expected Node2 Not to be Banned. NOTE: THIS PASS WILL NOT PASS IN DEBUG")
			update()
			time.Sleep(delay)
		}
		time.Sleep(delay)
		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))
	})

	t.Run("Test Backend BlockHeight Activates then Deactivates", func(t *testing.T) {
		reset()

		// delay := nodes["node2"].backend.GetBlockHeightZeroSlidingWindowLength()
		overrideBlock("node2", "latest", "0x0")
		for i := 0; i < 10; i++ {
			update()
			if i > 5 {
				require.True(t, bg.Consensus.IsBanned(nodes["node2"].backend), "Expected Node2 Not to be Banned. NOTE: THIS PASS WILL NOT PASS IN DEBUG")
				require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend), "Expected Node2 Not to be Banned. NOTE: THIS PASS WILL NOT PASS IN DEBUG")
			} else {
				require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend), "Expected Node2 Not to be Banned. NOTE: THIS PASS WILL NOT PASS IN DEBUG")
				require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend), "Expected Node2 Not to be Banned. NOTE: THIS PASS WILL NOT PASS IN DEBUG")
			}
		}
		overrideBlock("node2", "latest", "0x1")
		bg.Consensus.Unban(nodes["node2"].backend)
		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))
	})

}
