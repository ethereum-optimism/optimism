package node_utils

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const DefaultL1ID = 900
const DefaultL2ID = 901

// --- Generic RPC request/response types -------------------------------------

// ---------------------------------------------------------------------------

const (
	DEFAULT_TIMEOUT = 10 * time.Second
)

func GetNodeRPCEndpoint(node *dsl.L2CLNode) client.RPC {
	return node.Escape().ClientRPC()
}

func SendRPCRequest[T any](clientRPC client.RPC, method string, resOutput *T, params ...any) error {
	ctx, cancel := context.WithTimeout(context.Background(), DEFAULT_TIMEOUT)
	defer cancel()

	return clientRPC.CallContext(ctx, &resOutput, method, params...)
}

func MatchedWithinRange(t devtest.T, baseNode, refNode dsl.L2CLNode, delta uint64, lvl types.SafetyLevel, attempts int) dsl.CheckFunc {
	logger := t.Logger()
	chainID := baseNode.ChainID()

	return func() error {
		base := baseNode.ChainSyncStatus(chainID, lvl)
		ref := refNode.ChainSyncStatus(chainID, lvl)
		logger.Info("Expecting node to match with reference", "base", base.Number, "ref", ref.Number)
		return retry.Do0(t.Ctx(), attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				base = baseNode.ChainSyncStatus(chainID, lvl)
				ref = refNode.ChainSyncStatus(chainID, lvl)
				if ref.Number <= base.Number+delta || ref.Number >= base.Number-delta {
					logger.Info("Node matched", "ref_id", refNode, "base_id", baseNode, "ref", ref.Number, "base", base.Number, "delta", delta)

					// We get the same block from the head and tail node
					var headNode dsl.L2CLNode
					var tailNode eth.BlockID
					if ref.Number > base.Number {
						headNode = refNode
						tailNode = base
					} else {
						headNode = baseNode
						tailNode = ref
					}

					baseBlock, err := headNode.Escape().RollupAPI().OutputAtBlock(t.Ctx(), tailNode.Number)
					if err != nil {
						return err
					}

					t.Require().Equal(baseBlock.BlockRef.Number, tailNode.Number, "expected block number to match")
					t.Require().Equal(baseBlock.BlockRef.Hash, tailNode.Hash, "expected block hash to match")

					return nil
				}
				logger.Info("Node sync status", "base", base.Number, "ref", ref.Number)
				return fmt.Errorf("expected head to match: %s", lvl)
			})
	}
}
