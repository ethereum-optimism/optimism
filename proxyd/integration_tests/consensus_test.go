package integration_tests

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"testing"

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
	defer shutdown()

	bg := svr.BackendGroups["node"]
	require.NotNil(t, bg)
	require.NotNil(t, bg.Consensus)

	t.Run("initial consensus", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()

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

	t.Run("advance consensus", func(t *testing.T) {
		h1.ResetOverrides()
		h2.ResetOverrides()

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
			Response: buildResponse("0x2", "hash2"),
		})

		// poll for group consensus
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// consensus should stick to 0x1, since node1 is still lagging there
		bg.Consensus.UpdateBackendGroupConsensus(ctx)
		require.Equal(t, "0x1", bg.Consensus.GetConsensusBlockNumber().String())

		// advance latest on node1 to 0x2
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildResponse("0x2", "hash2"),
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
			Response: buildResponse("0x2", "hash2"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildResponse("0x2", "hash2"),
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
			Response: buildResponse("0x2", "wrong_hash"),
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
			Response: buildResponse("0x2", "hash2"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildResponse("0x2", "hash2"),
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
			Response: buildResponse("0x3", "hash3"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildResponse("0x3", "hash3"),
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
			Response: buildResponse("0x2", "wrong_hash2"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "0x3",
			Response: buildResponse("0x3", "wrong_hash3"),
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
			Response: buildResponse("0x2", "node1_0x2"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "0x2",
			Response: buildResponse("0x2", "node2_0x2"),
		})
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "0x3",
			Response: buildResponse("0x3", "node1_0x3"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "0x3",
			Response: buildResponse("0x3", "node2_0x3"),
		})
		h1.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildResponse("0x3", "node1_0x3"),
		})
		h2.AddOverride(&ms.MethodTemplate{
			Method:   "eth_getBlockByNumber",
			Block:    "latest",
			Response: buildResponse("0x3", "node2_0x3"),
		})

		// poll for group consensus
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)

		// should resolve to 0x1, the highest common ancestor
		require.Equal(t, "0x1", bg.Consensus.GetConsensusBlockNumber().String())
	})
}

func buildResponse(number string, hash string) string {
	return fmt.Sprintf(`{
      "jsonrpc": "2.0",
      "id": 67,
      "result": {
        "number": "%s",
		"hash": "%s"
      }
    }`, number, hash)
}
