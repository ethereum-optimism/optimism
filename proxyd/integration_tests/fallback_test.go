package integration_tests

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/proxyd"
	ms "github.com/ethereum-optimism/optimism/proxyd/tools/mockserver/handler"
	"github.com/stretchr/testify/require"
)

func setup_failover(t *testing.T) (map[string]nodeContext, *proxyd.BackendGroup, *ProxydHTTPClient, func(), []time.Time, []time.Time) {
	// setup mock servers
	node1 := NewMockBackend(nil)
	node2 := NewMockBackend(nil)

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

	// setup proxyd
	config := ReadConfig("fallback")
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
		"normal": {
			mockBackend: node1,
			backend:     bg.Backends[0],
			handler:     &h1,
		},
		"fallback": {
			mockBackend: node2,
			backend:     bg.Backends[1],
			handler:     &h2,
		},
	}
	normalTimestamps := []time.Time{}
	fallbackTimestamps := []time.Time{}

	return nodes, bg, client, shutdown, normalTimestamps, fallbackTimestamps
}

func TestFallback(t *testing.T) {
	nodes, bg, client, shutdown, normalTimestamps, fallbackTimestamps := setup_failover(t)
	defer nodes["normal"].mockBackend.Close()
	defer nodes["fallback"].mockBackend.Close()
	defer shutdown()

	ctx := context.Background()

	// Use Update to Advance the Candidate iteration
	update := func() {
		for _, be := range bg.Primaries() {
			bg.Consensus.UpdateBackend(ctx, be)
		}

		for _, be := range bg.Fallbacks() {
			healthyCandidates := bg.Consensus.FilterCandidates(bg.Primaries())
			if len(healthyCandidates) == 0 {
				bg.Consensus.UpdateBackend(ctx, be)
			}
		}

		bg.Consensus.UpdateBackendGroupConsensus(ctx)
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

	containsNode := func(backends []*proxyd.Backend, name string) bool {
		for _, be := range backends {
			// Note: Currently checks for name but would like to expose fallback better
			if be.Name == name {
				return true
			}
		}
		return false
	}

	// TODO: Improvement instead of simple array,
	// ensure normal and backend are returned in strict order
	recordLastUpdates := func(backends []*proxyd.Backend) []time.Time {
		lastUpdated := []time.Time{}
		for _, be := range backends {
			lastUpdated = append(lastUpdated, bg.Consensus.GetLastUpdate(be))
		}
		return lastUpdated
	}

	// convenient methods to manipulate state and mock responses
	reset := func() {
		for _, node := range nodes {
			node.handler.ResetOverrides()
			node.mockBackend.Reset()
		}
		bg.Consensus.ClearListeners()
		bg.Consensus.Reset()

		normalTimestamps = []time.Time{}
		fallbackTimestamps = []time.Time{}
	}

	/*
		triggerFirstNormalFailure: will trigger consensus group into fallback mode
		old consensus group should be returned one time, and fallback group should be enabled
		Fallback will be returned subsequent update
	*/
	triggerFirstNormalFailure := func() {
		overridePeerCount("normal", 0)
		update()
		require.True(t, containsNode(bg.Consensus.GetConsensusGroup(), "fallback"))
		require.False(t, containsNode(bg.Consensus.GetConsensusGroup(), "normal"))
		require.Equal(t, 1, len(bg.Consensus.GetConsensusGroup()))
		nodes["fallback"].mockBackend.Reset()
	}

	t.Run("Test fallback Mode will not be exited, unless state changes", func(t *testing.T) {
		reset()
		triggerFirstNormalFailure()
		for i := 0; i < 10; i++ {
			update()
			require.False(t, containsNode(bg.Consensus.GetConsensusGroup(), "normal"))
			require.True(t, containsNode(bg.Consensus.GetConsensusGroup(), "fallback"))
			require.Equal(t, 1, len(bg.Consensus.GetConsensusGroup()))
		}
	})

	t.Run("Test Healthy mode will not be exited unless state changes", func(t *testing.T) {
		reset()
		for i := 0; i < 10; i++ {
			update()
			require.Equal(t, 1, len(bg.Consensus.GetConsensusGroup()))
			require.False(t, containsNode(bg.Consensus.GetConsensusGroup(), "fallback"))
			require.True(t, containsNode(bg.Consensus.GetConsensusGroup(), "normal"))

			_, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x101", false})

			require.Equal(t, 200, statusCode)
			require.Nil(t, err, "error not nil")
			require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())
			require.Equal(t, "0xe1", bg.Consensus.GetSafeBlockNumber().String())
			require.Equal(t, "0xc1", bg.Consensus.GetFinalizedBlockNumber().String())
		}
		// TODO: Remove these, just here so compiler doesn't complain
		overrideNotInSync("normal")
		overrideBlock("normal", "safe", "0xb1")
		overrideBlockHash("fallback", "0x102", "0x102", "wrong_hash")
	})

	t.Run("trigger normal failure, subsequent update return failover in consensus group, and fallback mode enabled", func(t *testing.T) {
		reset()
		triggerFirstNormalFailure()
		update()
		require.Equal(t, 1, len(bg.Consensus.GetConsensusGroup()))
		require.True(t, containsNode(bg.Consensus.GetConsensusGroup(), "fallback"))
		require.False(t, containsNode(bg.Consensus.GetConsensusGroup(), "normal"))
	})

	t.Run("trigger healthy -> fallback, update -> healthy", func(t *testing.T) {
		reset()
		update()
		require.Equal(t, 1, len(bg.Consensus.GetConsensusGroup()))
		require.True(t, containsNode(bg.Consensus.GetConsensusGroup(), "normal"))
		require.False(t, containsNode(bg.Consensus.GetConsensusGroup(), "fallback"))

		triggerFirstNormalFailure()
		update()
		require.Equal(t, 1, len(bg.Consensus.GetConsensusGroup()))
		require.True(t, containsNode(bg.Consensus.GetConsensusGroup(), "fallback"))
		require.False(t, containsNode(bg.Consensus.GetConsensusGroup(), "normal"))

		overridePeerCount("normal", 5)
		update()
		require.Equal(t, 1, len(bg.Consensus.GetConsensusGroup()))
		require.True(t, containsNode(bg.Consensus.GetConsensusGroup(), "normal"))
		require.False(t, containsNode(bg.Consensus.GetConsensusGroup(), "fallback"))
	})

	t.Run("Ensure fallback is not updated when in normal mode", func(t *testing.T) {
		reset()
		for i := 0; i < 10; i++ {
			update()
			ts := recordLastUpdates(bg.Backends)
			normalTimestamps = append(normalTimestamps, ts[0])
			fallbackTimestamps = append(fallbackTimestamps, ts[1])

			require.False(t, normalTimestamps[i].IsZero())
			require.True(t, fallbackTimestamps[i].IsZero())

			require.True(t, containsNode(bg.Consensus.GetConsensusGroup(), "normal"))
			require.False(t, containsNode(bg.Consensus.GetConsensusGroup(), "fallback"))

			// consensus at block 0x101
			require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())
			require.Equal(t, "0xe1", bg.Consensus.GetSafeBlockNumber().String())
			require.Equal(t, "0xc1", bg.Consensus.GetFinalizedBlockNumber().String())
		}
	})

	/*
	 Set Normal backend to Fail -> both backends should be updated
	*/
	t.Run("Ensure both nodes are quieried in fallback mode", func(t *testing.T) {
		reset()
		triggerFirstNormalFailure()
		for i := 0; i < 10; i++ {
			update()
			ts := recordLastUpdates(bg.Backends)
			normalTimestamps = append(normalTimestamps, ts[0])
			fallbackTimestamps = append(fallbackTimestamps, ts[1])

			// Both Nodes should be updated again
			require.False(t, normalTimestamps[i].IsZero())
			require.False(t, fallbackTimestamps[i].IsZero(),
				fmt.Sprintf("Error: Fallback timestamp: %v was not queried on iteratio %d", fallbackTimestamps[i], i),
			)
			if i > 0 {
				require.Greater(t, normalTimestamps[i], normalTimestamps[i-1])
				require.Greater(t, fallbackTimestamps[i], fallbackTimestamps[i-1])
			}
		}
	})

	t.Run("Ensure both nodes are quieried in fallback mode", func(t *testing.T) {
		reset()
		triggerFirstNormalFailure()
		for i := 0; i < 10; i++ {
			update()
			ts := recordLastUpdates(bg.Backends)
			normalTimestamps = append(normalTimestamps, ts[0])
			fallbackTimestamps = append(fallbackTimestamps, ts[1])

			// Both Nodes should be updated again
			require.False(t, normalTimestamps[i].IsZero())
			require.False(t, fallbackTimestamps[i].IsZero(),
				fmt.Sprintf("Error: Fallback timestamp: %v was not queried on iteratio %d", fallbackTimestamps[i], i),
			)
			if i > 0 {
				require.Greater(t, normalTimestamps[i], normalTimestamps[i-1])
				require.Greater(t, fallbackTimestamps[i], fallbackTimestamps[i-1])
			}
		}
	})
	t.Run("Healthy -> Fallback -> Healthy with timestamps", func(t *testing.T) {
		reset()
		for i := 0; i < 10; i++ {
			update()
			ts := recordLastUpdates(bg.Backends)
			normalTimestamps = append(normalTimestamps, ts[0])
			fallbackTimestamps = append(fallbackTimestamps, ts[1])

			// Normal is queried, fallback is not
			require.False(t, normalTimestamps[i].IsZero())
			require.True(t, fallbackTimestamps[i].IsZero(),
				fmt.Sprintf("Error: Fallback timestamp: %v was not queried on iteratio %d", fallbackTimestamps[i], i),
			)
			if i > 0 {
				require.Greater(t, normalTimestamps[i], normalTimestamps[i-1])
				// Fallbacks should be zeros
				require.Equal(t, fallbackTimestamps[i], fallbackTimestamps[i-1])
			}
		}

		offset := 10
		triggerFirstNormalFailure()
		for i := 0; i < 10; i++ {
			update()
			ts := recordLastUpdates(bg.Backends)
			normalTimestamps = append(normalTimestamps, ts[0])
			fallbackTimestamps = append(fallbackTimestamps, ts[1])

			// Both Nodes should be updated again
			require.False(t, normalTimestamps[i+offset].IsZero())
			require.False(t, fallbackTimestamps[i+offset].IsZero())

			require.Greater(t, normalTimestamps[i+offset], normalTimestamps[i+offset-1])
			require.Greater(t, fallbackTimestamps[i+offset], fallbackTimestamps[i+offset-1])
		}

		overridePeerCount("normal", 5)
		offset = 20
		for i := 0; i < 10; i++ {
			update()
			ts := recordLastUpdates(bg.Backends)
			normalTimestamps = append(normalTimestamps, ts[0])
			fallbackTimestamps = append(fallbackTimestamps, ts[1])

			// Normal Node will be updated
			require.False(t, normalTimestamps[i+offset].IsZero())
			require.Greater(t, normalTimestamps[i+offset], normalTimestamps[i+offset-1])

			// fallback should not be updating
			if offset+i > 21 {
				require.Equal(t, fallbackTimestamps[i+offset], fallbackTimestamps[i+offset-1])
			}
		}
	})
}
