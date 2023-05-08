package challenger

import (
	"context"
	"errors"
	"math/big"
	_ "net/http/pprof"
	"time"

	goEth "github.com/ethereum/go-ethereum"
	common "github.com/ethereum/go-ethereum/common"
	goTypes "github.com/ethereum/go-ethereum/core/types"

	flags "github.com/ethereum-optimism/optimism/op-challenger/flags"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"
)

var supportedL2OutputVersion = eth.Bytes32{}

// oracle executes the Challenger's oracle hook.
// This function is intended to be spawned in a goroutine.
// It will run until the Challenger's context is cancelled.
//
// The oracle hook listen's for `OutputProposed` events from the L2 Output Oracle contract.
// When it receives an event, it will validate the output against the trusted rollup node.
// If an output is invalid, it will create a dispute game of the given configuration.
func (c *Challenger) oracle() {
	defer c.wg.Done()

	// The `OutputProposed` event is encoded as:
	// 0: bytes32 indexed outputRoot,
	// 1: uint256 indexed l2OutputIndex,
	// 2: uint256 indexed l2BlockNumber,
	// 3: uint256 l1Timestamp

	// Listen for `OutputProposed` events from the L2 Output Oracle contract
	event := c.l2ooABI.Events["OutputProposed"]
	query := goEth.FilterQuery{
		Topics: [][]common.Hash{
			{event.ID},
		},
	}

	logs := make(chan goTypes.Log)
	sub, err := c.l1Client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		c.log.Error("failed to subscribe to logs", "err", err)
		return
	}

	for {
		select {
		case err := <-sub.Err():
			c.log.Error("failed to subscribe to logs", "err", err)
			return

		case vLog := <-logs:
			l2BlockNumber := new(big.Int).SetBytes(vLog.Topics[3][:])
			expected := vLog.Topics[1]
			c.log.Info("Validating output", "l2BlockNumber", l2BlockNumber, "outputRoot", expected.Hex())
			isValid, rootClaim, err := c.ValidateOutput(c.ctx, l2BlockNumber, (eth.Bytes32)(expected))
			if err != nil || isValid {
				break
			}

			c.metr.RecordInvalidOutput(
				eth.L2BlockRef{
					Hash:   vLog.Topics[0],
					Number: l2BlockNumber.Uint64(),
				},
			)
			c.log.Debug("Creating dispute game for", "l2BlockNumber", l2BlockNumber, "rootClaim", rootClaim)
			cCtx, cancel := context.WithTimeout(c.ctx, 10*time.Minute)
			_, err = c.createDisputeGame(cCtx, c.gameType, rootClaim, l2BlockNumber)
			if err != nil {
				c.log.Error("Failed to challenge transaction", "err", err)
				cancel()
				break
			}
			c.metr.RecordDisputeGameCreated(eth.L2BlockRef{
				Hash:   common.Hash(*rootClaim),
				Number: l2BlockNumber.Uint64(),
			})
			cancel()
		case <-c.done:
			return
		}
	}
}

// createDisputeGame creates a dispute game.
// It will create a dispute game of type `gameType` with the given output and l2BlockNumber.
// The `gameType` must be a valid dispute game type as defined by the `GameType` enum in the DisputeGameFactory contract.
func (c *Challenger) createDisputeGame(ctx context.Context, gameType flags.GameType, output *eth.Bytes32, l2BlockNumber *big.Int) (*common.Address, error) {
	c.log.Info("dispute game creation not implemented", "gameType", gameType, "output", output, "l2BlockNumber", l2BlockNumber)
	return nil, errors.New("dispute game creation not implemented")
}

// ValidateOutput checks that a given output is expected via a trusted rollup node rpc.
// It returns: if the output is correct, error
func (c *Challenger) ValidateOutput(ctx context.Context, l2BlockNumber *big.Int, expected eth.Bytes32) (bool, *eth.Bytes32, error) {
	ctx, cancel := context.WithTimeout(ctx, c.networkTimeout)
	defer cancel()
	output, err := c.rollupClient.OutputAtBlock(ctx, l2BlockNumber.Uint64())
	if err != nil {
		c.log.Error("failed to fetch output for l2BlockNumber %d: %w", l2BlockNumber, err)
		return true, nil, err
	}
	if output.Version != supportedL2OutputVersion {
		c.log.Error("unsupported l2 output version: %s", output.Version)
		return true, nil, errors.New("unsupported l2 output version")
	}
	// If the block numbers don't match, we should try to fetch the output again
	if output.BlockRef.Number != l2BlockNumber.Uint64() {
		c.log.Error("invalid blockNumber: next blockNumber is %v, blockNumber of block is %v", l2BlockNumber, output.BlockRef.Number)
		return true, nil, errors.New("invalid blockNumber")
	}
	if output.OutputRoot == expected {
		c.metr.RecordValidOutput(output.BlockRef)
	}
	return output.OutputRoot != expected, &output.OutputRoot, nil
}
