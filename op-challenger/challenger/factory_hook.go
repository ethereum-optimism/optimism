package challenger

import (
	"context"
	"math/big"
	_ "net/http/pprof"

	bindings "github.com/ethereum-optimism/optimism/op-challenger/contracts/bindings"
	goEth "github.com/ethereum/go-ethereum"
	bind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	common "github.com/ethereum/go-ethereum/common"
	goTypes "github.com/ethereum/go-ethereum/core/types"

	txmgr "github.com/ethereum-optimism/optimism/op-service/txmgr"
)

// factory executes the Challenger's factory hook.
// This function is intended to be spawned in a goroutine.
// It will run until the Challenger's context is cancelled.
//
// The factory hook listens for `DisputeGameCreated` events from the dispute game factory contract.
// When it receives an event, it will check the dispute game parameters and submit a challenge if valid.
func (c *Challenger) factory() {
	defer c.wg.Done()

	// `DisputeGameCreated`` event encoded as:
	// 0: address indexed disputeProxy
	// 1: GameType indexed gameType
	// 2: Claim indexed rootClaim

	// Listen for `DisputeGameCreated` events from the Dispute Game Factory contract
	event := c.dgfABI.Events["DisputeGameCreated"]
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
			// Parse the `DisputeGameCreated` event
			adgProxy := common.BytesToAddress(vLog.Topics[1][:])
			gameType := new(big.Int).SetBytes(vLog.Topics[2][:])
			rootClaim := vLog.Topics[3]

			// Dispatch on the game type
			switch gameType.Uint64() {
			case AttestationDisputeGameType:
				err := c.attestationChallenge(adgProxy, rootClaim)
				if err != nil {
					c.log.Error("failed to challenge attestation dispute game", "err", err)
				}
			}
		case <-c.done:
			return
		}
	}
}

// attestationChallenge sends a challenge to the given attestation dispute game.
func (c *Challenger) attestationChallenge(adgProxy common.Address, rootClaim common.Hash) error {
	cCtx, cCancel := context.WithTimeout(c.ctx, c.networkTimeout)
	defer cCancel()
	adgContract, err := bindings.NewMockAttestationDisputeGameCaller(adgProxy, c.l1Client)
	if err != nil {
		return err
	}
	l2BlockNumber, err := adgContract.L2BLOCKNUMBER(&bind.CallOpts{Context: cCtx})
	if err != nil {
		return err
	}

	// Create an eip-712 signature for the contested output
	signature, err := c.signOutput(l2BlockNumber, rootClaim)
	if err != nil {
		return err
	}

	data, err := c.adgABI.Pack("challenge", signature)
	if err != nil {
		return err
	}
	receipt, err := c.txMgr.Send(cCtx, txmgr.TxCandidate{
		TxData:   data,
		To:       adgProxy,
		GasLimit: 0,
		From:     c.from,
	})
	if err != nil {
		return err
	}
	c.log.Info("attestation challenge tx successful", "tx_hash", receipt.TxHash)
	c.metr.RecordChallengeSent(l2BlockNumber, rootClaim)
	return nil
}
