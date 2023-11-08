package geth

import (
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// fakePoS is a testing-only utility to attach to Geth,
// to build a fake proof-of-stake L1 chain with fixed block time and basic lagging safe/finalized blocks.
type fakePoS struct {
	clock     clock.Clock
	eth       *eth.Ethereum
	log       log.Logger
	blockTime uint64

	withdrawalsIndex uint64

	finalizedDistance uint64
	safeDistance      uint64

	engineAPI *catalyst.ConsensusAPI
	sub       ethereum.Subscription
}

func (f *fakePoS) Start() error {
	if advancing, ok := f.clock.(*clock.AdvancingClock); ok {
		advancing.Start()
	}
	withdrawalsRNG := rand.New(rand.NewSource(450368975843)) // avoid generating the same address as any test
	f.sub = event.NewSubscription(func(quit <-chan struct{}) error {
		// poll every half a second: enough to catch up with any block time when ticks are missed
		t := f.clock.NewTicker(time.Second / 2)
		for {
			select {
			case now := <-t.Ch():
				chain := f.eth.BlockChain()
				head := chain.CurrentBlock()
				finalized := chain.CurrentFinalBlock()
				if finalized == nil { // fallback to genesis if nothing is finalized
					finalized = chain.Genesis().Header()
				}
				safe := chain.CurrentSafeBlock()
				if safe == nil { // fallback to finalized if nothing is safe
					safe = finalized
				}
				if head.Number.Uint64() > f.finalizedDistance { // progress finalized block, if we can
					finalized = f.eth.BlockChain().GetHeaderByNumber(head.Number.Uint64() - f.finalizedDistance)
				}
				if head.Number.Uint64() > f.safeDistance { // progress safe block, if we can
					safe = f.eth.BlockChain().GetHeaderByNumber(head.Number.Uint64() - f.safeDistance)
				}
				// start building the block as soon as we are past the current head time
				if head.Time >= uint64(now.Unix()) {
					continue
				}
				newBlockTime := head.Time + f.blockTime
				if time.Unix(int64(newBlockTime), 0).Add(5 * time.Minute).Before(f.clock.Now()) {
					// We're a long way behind, let's skip some blocks...
					newBlockTime = uint64(f.clock.Now().Unix())
				}
				// create some random withdrawals
				withdrawals := make([]*types.Withdrawal, withdrawalsRNG.Intn(4))
				for i := 0; i < len(withdrawals); i++ {
					withdrawals[i] = &types.Withdrawal{
						Index:     f.withdrawalsIndex + uint64(i),
						Validator: withdrawalsRNG.Uint64() % 100_000_000, // 100 million fake validators
						Address:   testutils.RandomAddress(withdrawalsRNG),
						// in gwei, consensus-layer quirk. withdraw non-zero value up to 50 ETH
						Amount: uint64(withdrawalsRNG.Intn(50_000_000_000) + 1),
					}
				}
				res, err := f.engineAPI.ForkchoiceUpdatedV2(engine.ForkchoiceStateV1{
					HeadBlockHash:      head.Hash(),
					SafeBlockHash:      safe.Hash(),
					FinalizedBlockHash: finalized.Hash(),
				}, &engine.PayloadAttributes{
					Timestamp:             newBlockTime,
					Random:                common.Hash{},
					SuggestedFeeRecipient: head.Coinbase,
					Withdrawals:           withdrawals,
				})
				if err != nil {
					f.log.Error("failed to start building L1 block", "err", err)
					continue
				}
				if res.PayloadID == nil {
					f.log.Error("failed to start block building", "res", res)
					continue
				}
				// wait with sealing, if we are not behind already
				delay := time.Unix(int64(newBlockTime), 0).Sub(f.clock.Now())
				tim := f.clock.NewTimer(delay)
				select {
				case <-tim.Ch():
					// no-op
				case <-quit:
					tim.Stop()
					return nil
				}
				envelope, err := f.engineAPI.GetPayloadV2(*res.PayloadID)
				if err != nil {
					f.log.Error("failed to finish building L1 block", "err", err)
					continue
				}
				if _, err := f.engineAPI.NewPayloadV2(*envelope.ExecutionPayload); err != nil {
					f.log.Error("failed to insert built L1 block", "err", err)
					continue
				}
				if _, err := f.engineAPI.ForkchoiceUpdatedV2(engine.ForkchoiceStateV1{
					HeadBlockHash:      envelope.ExecutionPayload.BlockHash,
					SafeBlockHash:      safe.Hash(),
					FinalizedBlockHash: finalized.Hash(),
				}, nil); err != nil {
					f.log.Error("failed to make built L1 block canonical", "err", err)
					continue
				}
				// Increment global withdrawals index in the CL.
				// The EL doesn't really care about the value,
				// but it's nice to mock something consistent with the CL specs.
				f.withdrawalsIndex += uint64(len(withdrawals))
			case <-quit:
				return nil
			}
		}
	})
	return nil
}

func (f *fakePoS) Stop() error {
	f.sub.Unsubscribe()
	if advancing, ok := f.clock.(*clock.AdvancingClock); ok {
		advancing.Stop()
	}
	return nil
}
