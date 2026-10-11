package monitor

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// chainLiquidity is the running state of one chain's ETHLiquidity. Its size is constant: sums,
// not lists of events.
type chainLiquidity struct {
	// baselineBlock is the block the chain's ETHLiquidity is measured from: genesis, or the block
	// a chain that activated interop by upgrade activated it at.
	baselineBlock uint64
	// baseline is the balance at baselineBlock, read on the first check.
	baseline *big.Int
	// through is the last block the sums below include; they cover (baselineBlock, through].
	// throughHash is its hash, to notice a reorg below it.
	through     uint64
	throughHash common.Hash
	// funded sums LiquidityFunded amounts: operator top-ups.
	funded *big.Int
	// sent sums SendETH amounts: the ETH the bridge burned into ETHLiquidity.
	sent *big.Int
	// released sums RelayETH and RefundETH amounts: the ETH the bridge minted out of it.
	released *big.Int
	// surplus is the balance-derived net less the event-derived net at the last check. ETH
	// forced into ETHLiquidity raises it; nothing legitimate lowers it.
	surplus *big.Int
}

func (st *chainLiquidity) clone() *chainLiquidity {
	c := *st
	c.funded, c.sent, c.released, c.surplus = new(big.Int).Set(st.funded), new(big.Int).Set(st.sent),
		new(big.Int).Set(st.released), new(big.Int).Set(st.surplus)
	return &c
}

// liquidityTripwire checks ETH conservation across the cluster. Only the SuperchainETHBridge moves
// ETH in and out of ETHLiquidity, through burn on a send and mint on a relay or a refund. Per chain
// and at one L2 timestamp, it checks that:
//   - every LiquidityBurned is paired with a SendETH of the amount in the same transaction, and
//     every LiquidityMinted with a RelayETH or a RefundETH;
//   - the balance less its baseline less LiquidityFunded top-ups equals the SendETH sum less the
//     RelayETH and RefundETH sums. ETH forced into ETHLiquidity (a self-destruct, or a relay to
//     ETHLiquidity itself) raises the balance with no event, so the balance may exceed the events
//     by a surplus, but that surplus must never shrink: a shrink is ETH that left with no event;
//   - summed over the cluster, SendETH less RelayETH and RefundETH, the ETH in flight, is never
//     negative.
//
// A message both relayed and refunded is caught by the expiry and refund checks, not here.
type liquidityTripwire struct {
	clients map[eth.ChainID]ExpiryClient
	head    eth.BlockLabel
	chains  map[eth.ChainID]*chainLiquidity
	// catchUpBlocks bounds the blocks one check adds to a chain's sums, so catching up from an
	// old baseline spreads over several checks instead of stalling the checker.
	catchUpBlocks uint64
}

func newChainLiquidity(baselineBlock uint64) *chainLiquidity {
	return &chainLiquidity{baselineBlock: baselineBlock, funded: new(big.Int), sent: new(big.Int),
		released: new(big.Int), surplus: new(big.Int)}
}

func newLiquidityTripwire(clients map[eth.ChainID]ExpiryClient, head eth.BlockLabel, baselineBlocks map[eth.ChainID]uint64) *liquidityTripwire {
	chains := make(map[eth.ChainID]*chainLiquidity, len(clients))
	for id := range clients {
		chains[id] = newChainLiquidity(baselineBlocks[id])
	}
	return &liquidityTripwire{clients: clients, head: head, chains: chains, catchUpBlocks: 10 * maxLogRange}
}

// liquidityReading is the result of one tripwire check.
type liquidityReading struct {
	timestamp uint64
	// inFlight is the cluster's ETH sent and not yet relayed or refunded, or nil if some chain
	// could not be read up to timestamp.
	inFlight   *big.Int
	violations map[eth.ChainID][]violation
}

var liquidityTopics = []common.Hash{sendETHTopic, relayETHTopic, refundETHTopic,
	liquidityBurnedTopic, liquidityMintedTopic, liquidityFundedTopic}

// check advances every chain towards the latest L2 timestamp at or below all their heads. Each
// chain's progress and violations stand on their own; the cluster's ETH in flight is only
// computed once every chain has reached that timestamp.
func (lt *liquidityTripwire) check(ctx context.Context) (*liquidityReading, error) {
	ids := make([]eth.ChainID, 0, len(lt.clients))
	for id := range lt.clients {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b eth.ChainID) int { return a.Cmp(b) })

	heads := make(map[eth.ChainID]eth.BlockInfo, len(ids))
	var ts uint64
	for i, id := range ids {
		head, err := lt.clients[id].InfoByLabel(ctx, lt.head)
		if err != nil {
			return nil, fmt.Errorf("read %s head of chain %s: %w", lt.head, id, err)
		}
		heads[id] = head
		if i == 0 || head.Time() < ts {
			ts = head.Time()
		}
	}

	reading := &liquidityReading{timestamp: ts, violations: make(map[eth.ChainID][]violation)}
	inFlight := new(big.Int)
	complete := true
	var errs error
	for _, id := range ids {
		st, violations, caughtUp, err := lt.advance(ctx, id, ts, heads[id])
		if err != nil {
			errs = errors.Join(errs, err)
			complete = false
			continue
		}
		lt.chains[id] = st
		reading.violations[id] = violations
		complete = complete && caughtUp
		inFlight.Add(inFlight, new(big.Int).Sub(st.sent, st.released))
	}
	if complete {
		reading.inFlight = inFlight
	}
	return reading, errs
}

// advance returns a chain's state at its last block at or before `ts`, or as far towards it as
// one check goes, with the violations found on the way. It does not change the stored state.
func (lt *liquidityTripwire) advance(ctx context.Context, id eth.ChainID, ts uint64, head eth.BlockInfo) (*chainLiquidity, []violation, bool, error) {
	c := lt.clients[id]
	st := lt.chains[id].clone()
	target, err := blockAtOrBefore(ctx, c, ts, head)
	if err != nil {
		return nil, nil, false, fmt.Errorf("find block at time %d on chain %s: %w", ts, id, err)
	}
	if target.NumberU64() < st.baselineBlock {
		return nil, nil, false, fmt.Errorf("%w: chain %s is at block %d, before its liquidity baseline block %d", errNotReady, id, target.NumberU64(), st.baselineBlock)
	}
	if st.baseline != nil {
		// Start over if the last block counted was reorged out.
		last, err := c.InfoByNumber(ctx, st.through)
		if err != nil {
			return nil, nil, false, err
		}
		if last.Hash() != st.throughHash {
			st = newChainLiquidity(st.baselineBlock)
		}
	}
	if st.baseline == nil {
		baselineInfo, err := c.InfoByNumber(ctx, st.baselineBlock)
		if err != nil {
			return nil, nil, false, err
		}
		baseline, err := c.BalanceAt(ctx, predeploys.ETHLiquidityAddr, new(big.Int).SetUint64(st.baselineBlock))
		if err != nil {
			return nil, nil, false, fmt.Errorf("read ETHLiquidity balance of chain %s at baseline block %d: %w", id, st.baselineBlock, err)
		}
		st.baseline, st.through, st.throughHash = baseline, st.baselineBlock, baselineInfo.Hash()
	}
	if target.NumberU64() < st.through {
		return nil, nil, false, fmt.Errorf("%w: chain %s went back from block %d to %d", errNotReady, id, st.through, target.NumberU64())
	}

	end := target
	if target.NumberU64() > st.through+lt.catchUpBlocks {
		if end, err = c.InfoByNumber(ctx, st.through+lt.catchUpBlocks); err != nil {
			return nil, nil, false, err
		}
	}
	var violations []violation
	if end.NumberU64() > st.through {
		logs, err := filterLogsChunked(ctx, c, ethereum.FilterQuery{
			Addresses: []common.Address{predeploys.SuperchainETHBridgeAddr, predeploys.ETHLiquidityAddr},
			Topics:    [][]common.Hash{liquidityTopics},
		}, st.through+1, end.NumberU64())
		if err != nil {
			return nil, nil, false, err
		}
		violations = st.addLogs(logs)
	}
	st.through, st.throughHash = end.NumberU64(), end.Hash()

	balance, err := c.BalanceAt(ctx, predeploys.ETHLiquidityAddr, new(big.Int).SetUint64(end.NumberU64()))
	if err != nil {
		return nil, nil, false, fmt.Errorf("read ETHLiquidity balance of chain %s at block %d: %w", id, end.NumberU64(), err)
	}
	balanceNet := new(big.Int).Sub(balance, st.baseline)
	balanceNet.Sub(balanceNet, st.funded)
	eventNet := new(big.Int).Sub(st.sent, st.released)
	surplus := new(big.Int).Sub(balanceNet, eventNet)
	if surplus.Cmp(st.surplus) < 0 {
		violations = append(violations, violationf(reasonUnexplainedOutflow,
			"ETHLiquidity on chain %s at block %d holds %s wei less than its bridge events account for (surplus went from %s to %s)",
			id, end.NumberU64(), new(big.Int).Sub(st.surplus, surplus), st.surplus, surplus))
	}
	st.surplus = surplus
	return st, violations, end.NumberU64() == target.NumberU64(), nil
}

// addLogs adds a block range's bridge and liquidity events to the sums, and checks that every
// burn and mint of a transaction is paired with a bridge event of the amount.
func (st *chainLiquidity) addLogs(logs []types.Log) []violation {
	type txAmounts struct{ burned, minted, sent, released []*big.Int }
	byTx := make(map[common.Hash]*txAmounts)
	var order []common.Hash
	for i := range logs {
		l := &logs[i]
		if len(l.Topics) == 0 || len(l.Data) < 32 {
			continue
		}
		tx, ok := byTx[l.TxHash]
		if !ok {
			tx = &txAmounts{}
			byTx[l.TxHash] = tx
			order = append(order, l.TxHash)
		}
		amount := new(big.Int).SetBytes(l.Data[:32])
		switch {
		case l.Address == predeploys.ETHLiquidityAddr && l.Topics[0] == liquidityFundedTopic:
			st.funded.Add(st.funded, amount)
		case l.Address == predeploys.ETHLiquidityAddr && l.Topics[0] == liquidityBurnedTopic:
			tx.burned = append(tx.burned, amount)
		case l.Address == predeploys.ETHLiquidityAddr && l.Topics[0] == liquidityMintedTopic:
			tx.minted = append(tx.minted, amount)
		case l.Address == predeploys.SuperchainETHBridgeAddr && l.Topics[0] == sendETHTopic:
			tx.sent = append(tx.sent, amount)
			st.sent.Add(st.sent, amount)
		case l.Address == predeploys.SuperchainETHBridgeAddr && (l.Topics[0] == relayETHTopic || l.Topics[0] == refundETHTopic):
			tx.released = append(tx.released, amount)
			st.released.Add(st.released, amount)
		}
	}
	var violations []violation
	for _, hash := range order {
		tx := byTx[hash]
		if !sameAmounts(tx.burned, tx.sent) {
			violations = append(violations, violationf(reasonUnpairedBurn,
				"tx %s burned %v in ETHLiquidity but the bridge sent %v", hash, tx.burned, tx.sent))
		}
		if !sameAmounts(tx.minted, tx.released) {
			violations = append(violations, violationf(reasonUnpairedMint,
				"tx %s minted %v from ETHLiquidity but the bridge relayed or refunded %v", hash, tx.minted, tx.released))
		}
	}
	return violations
}

func sameAmounts(a, b []*big.Int) bool {
	if len(a) != len(b) {
		return false
	}
	a, b = slices.Clone(a), slices.Clone(b)
	slices.SortFunc(a, (*big.Int).Cmp)
	slices.SortFunc(b, (*big.Int).Cmp)
	return slices.EqualFunc(a, b, func(x, y *big.Int) bool { return x.Cmp(y) == 0 })
}
