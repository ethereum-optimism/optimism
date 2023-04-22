package challenger

import (
	"context"
	"math/big"
	_ "net/http/pprof"
	"time"

	goEth "github.com/ethereum/go-ethereum"
	common "github.com/ethereum/go-ethereum/common"
	goTypes "github.com/ethereum/go-ethereum/core/types"

	txmgr "github.com/ethereum-optimism/optimism/op-service/txmgr"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"
)

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
			contract, err := c.createDisputeGame(cCtx, AttestationDisputeGameType, rootClaim, l2BlockNumber)
			if err != nil {
				c.log.Error("Failed to challenge transaction", "err", err)
				cancel()
				break
			}
			c.metr.RecordDisputeGameCreated(l2BlockNumber, common.Hash(*rootClaim), *contract)
			cancel()
		case <-c.done:
			return
		}
	}
}

// createDisputeGame creates a dispute game.
// It will create a dispute game of type `gameType` with the given output and l2BlockNumber.
// The `gameType` must be a valid dispute game type as defined by the `GameType` enum in the DisputeGameFactory contract.
func (c *Challenger) createDisputeGame(ctx context.Context, gameType uint8, output *eth.Bytes32, l2BlockNumber *big.Int) (*common.Address, error) {
	cCtx, cCancel := context.WithTimeout(ctx, c.networkTimeout)
	defer cCancel()
	data, err := c.dgfABI.Pack("create", gameType, output, common.BigToHash(l2BlockNumber))
	if err != nil {
		return nil, err
	}
	receipt, err := c.txMgr.Send(cCtx, txmgr.TxCandidate{
		TxData:   data,
		To:       c.dgfContractAddr,
		GasLimit: 0,
		From:     c.from,
	})
	if err != nil {
		return nil, err
	}
	c.log.Info("challenger successfully sent tx", "tx_hash", receipt.TxHash, "output", output, "l2BlockNumber", l2BlockNumber)
	c.log.Info("challenger dispute game contract", "contract", receipt.ContractAddress)
	return &receipt.ContractAddress, nil
}
