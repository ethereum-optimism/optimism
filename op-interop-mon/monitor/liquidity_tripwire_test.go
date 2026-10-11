package monitor

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func (s *expiryScenario) tripwire(baselines map[eth.ChainID]uint64) *liquidityTripwire {
	return newLiquidityTripwire(map[eth.ChainID]ExpiryClient{s.a.id: s.a, s.b.id: s.b}, eth.Finalized, baselines)
}

func (s *expiryScenario) setHead(n uint64) {
	s.a.safe, s.b.safe = n, n
}

// adjustBalance changes a chain's ETHLiquidity balance by `delta` from block n on.
func adjustBalance(c *fakeChain, n uint64, delta *big.Int) {
	c.setBalance(predeploys.ETHLiquidityAddr, n, new(big.Int).Add(valueAt(c.balances[predeploys.ETHLiquidityAddr], n), delta))
}

// relaySend relays the scenario's send on B in block n: mint, RelayETH.
func (s *expiryScenario) relaySend(n uint64) {
	tx := txHash("relay on B")
	s.b.addLog(n, tx, liquidityLog(liquidityMintedTopic, predeploys.SuperchainETHBridgeAddr, s.amount))
	s.b.addLog(n, tx, relayETHLog(s.sender, s.to, s.amount, s.a.id))
	adjustBalance(s.b, n, new(big.Int).Neg(s.amount))
}

func checkTripwire(t *testing.T, lt *liquidityTripwire) (*big.Int, map[eth.ChainID][]string) {
	r, err := lt.check(context.Background())
	require.NoError(t, err)
	out := make(map[eth.ChainID][]string)
	for id, vs := range r.violations {
		if len(vs) > 0 {
			out[id] = reasons(vs)
		}
	}
	return r.inFlight, out
}

func TestLiquidityTripwire(t *testing.T) {
	t.Run("send in flight, then refunded", func(t *testing.T) {
		s := newExpiryScenario()
		lt := s.tripwire(nil)
		s.setHead(s.sendBlock)
		inFlight, violations := checkTripwire(t, lt)
		require.Empty(t, violations)
		require.Equal(t, s.amount, inFlight, "the send is in flight")
		s.setHead(s.refundBlock)
		inFlight, violations = checkTripwire(t, lt)
		require.Empty(t, violations)
		require.Zero(t, inFlight.Sign(), "the refund returned it")
	})

	t.Run("send relayed", func(t *testing.T) {
		s := newExpiryScenario()
		s.a.removeLogs(refundETHTopic)
		s.a.removeLogs(liquidityMintedTopic)
		s.a.balances[predeploys.ETHLiquidityAddr] = s.a.balances[predeploys.ETHLiquidityAddr][:2]
		s.relaySend(s.sendBlock + 2)
		s.setHead(s.sendBlock + 2)
		inFlight, violations := checkTripwire(t, s.tripwire(nil))
		require.Empty(t, violations)
		require.Zero(t, inFlight.Sign())
	})

	t.Run("send both refunded and relayed", func(t *testing.T) {
		// Both mints look legitimate on their own chain; only the cluster total shows the
		// double mint.
		s := newExpiryScenario()
		lt := s.tripwire(nil)
		s.setHead(s.refundBlock)
		_, violations := checkTripwire(t, lt)
		require.Empty(t, violations)
		s.relaySend(s.refundBlock + 2)
		s.setHead(s.refundBlock + 2)
		inFlight, violations := checkTripwire(t, lt)
		require.Empty(t, violations)
		require.Equal(t, new(big.Int).Neg(s.amount), inFlight, "a mint without a burn makes the total negative")
	})

	t.Run("mint with no bridge event", func(t *testing.T) {
		s := newExpiryScenario()
		lt := s.tripwire(nil)
		s.setHead(s.sendBlock)
		_, violations := checkTripwire(t, lt)
		require.Empty(t, violations)
		// Less than the ETH in flight, so the cluster total alone would not show it.
		adjustBalance(s.a, s.sendBlock+1, big.NewInt(-1))
		s.setHead(s.sendBlock + 1)
		inFlight, violations := checkTripwire(t, lt)
		require.Equal(t, map[eth.ChainID][]string{s.a.id: {reasonUnexplainedOutflow}}, violations)
		require.Equal(t, 1, inFlight.Sign(), "the cluster total is still positive")
	})

	t.Run("mint paired with no bridge event", func(t *testing.T) {
		s := newExpiryScenario()
		s.a.addLog(s.sendBlock+1, txHash("rogue mint"), liquidityLog(liquidityMintedTopic, predeploys.SuperchainETHBridgeAddr, big.NewInt(5)))
		adjustBalance(s.a, s.sendBlock+1, big.NewInt(-5))
		s.setHead(s.sendBlock + 1)
		_, violations := checkTripwire(t, s.tripwire(nil))
		require.Equal(t, map[eth.ChainID][]string{s.a.id: {reasonUnpairedMint, reasonUnexplainedOutflow}}, violations)
	})

	t.Run("burn paired with no bridge event", func(t *testing.T) {
		s := newExpiryScenario()
		s.a.removeLogs(sendETHTopic)
		s.setHead(s.sendBlock)
		_, violations := checkTripwire(t, s.tripwire(nil))
		require.Equal(t, map[eth.ChainID][]string{s.a.id: {reasonUnpairedBurn}}, violations)
	})

	t.Run("operator top-up and forced ETH are not violations", func(t *testing.T) {
		s := newExpiryScenario()
		lt := s.tripwire(nil)
		s.setHead(s.sendBlock)
		_, violations := checkTripwire(t, lt)
		require.Empty(t, violations)
		topUp := big.NewInt(params.Ether)
		s.a.addLog(s.sendBlock+1, txHash("fund"), liquidityLog(liquidityFundedTopic, common.HexToAddress("0xf0"), topUp))
		adjustBalance(s.a, s.sendBlock+1, topUp)
		adjustBalance(s.b, s.sendBlock+1, big.NewInt(7)) // a self-destruct into ETHLiquidity
		s.setHead(s.sendBlock + 1)
		inFlight, violations := checkTripwire(t, lt)
		require.Empty(t, violations)
		require.Equal(t, s.amount, inFlight)
	})

	t.Run("forced ETH does not hide a later mint with no event", func(t *testing.T) {
		s := newExpiryScenario()
		lt := s.tripwire(nil)
		adjustBalance(s.a, s.sendBlock+1, big.NewInt(7))
		s.setHead(s.sendBlock + 1)
		_, violations := checkTripwire(t, lt)
		require.Empty(t, violations)
		adjustBalance(s.a, s.sendBlock+2, big.NewInt(-7))
		s.setHead(s.sendBlock + 2)
		_, violations = checkTripwire(t, lt)
		require.Equal(t, map[eth.ChainID][]string{s.a.id: {reasonUnexplainedOutflow}}, violations)
	})

	t.Run("reads every chain at the same timestamp", func(t *testing.T) {
		s := newExpiryScenario()
		s.a.removeLogs(refundETHTopic)
		s.a.removeLogs(liquidityMintedTopic)
		s.a.balances[predeploys.ETHLiquidityAddr] = s.a.balances[predeploys.ETHLiquidityAddr][:2]
		// B relays at a time A has not reached: the relay must not be counted yet, or the
		// cluster would look like it relayed more than it sent.
		s.relaySend(s.sendBlock + 2)
		s.a.safe, s.b.safe = s.sendBlock, s.sendBlock+10
		lt := s.tripwire(nil)
		inFlight, violations := checkTripwire(t, lt)
		require.Empty(t, violations)
		require.Equal(t, s.amount, inFlight)
	})

	t.Run("baseline block of a chain that activated interop by upgrade", func(t *testing.T) {
		s := newExpiryScenario()
		s.setHead(s.sendBlock)
		inFlight, violations := checkTripwire(t, s.tripwire(map[eth.ChainID]uint64{s.a.id: s.sendBlock}))
		require.Empty(t, violations)
		require.Zero(t, inFlight.Sign(), "the send happened at the baseline block")
	})

	t.Run("a chain that cannot be read does not hold back the others", func(t *testing.T) {
		s := newExpiryScenario()
		lt := s.tripwire(map[eth.ChainID]uint64{s.b.id: s.sendBlock + 5})
		s.setHead(s.sendBlock + 2)
		r, err := lt.check(context.Background())
		require.ErrorIs(t, err, errNotReady, "B is before its baseline block")
		require.Nil(t, r.inFlight, "no cluster total without every chain")
		require.Equal(t, s.sendBlock+2, lt.chains[s.a.id].through, "A's progress is kept")
	})

	t.Run("catches up over several checks", func(t *testing.T) {
		s := newExpiryScenario()
		lt := s.tripwire(nil)
		lt.catchUpBlocks = 10
		s.setHead(s.refundBlock)
		checks := 0
		for {
			checks++
			r, err := lt.check(context.Background())
			require.NoError(t, err)
			require.Empty(t, r.violations[s.a.id])
			require.Empty(t, r.violations[s.b.id])
			if r.inFlight != nil {
				require.Zero(t, r.inFlight.Sign())
				break
			}
		}
		require.Equal(t, 5, checks, "45 blocks at 10 blocks per check")
	})

	t.Run("starts over after a reorg below its progress", func(t *testing.T) {
		s := newExpiryScenario()
		lt := s.tripwire(nil)
		s.setHead(s.sendBlock)
		_, violations := checkTripwire(t, lt)
		require.Empty(t, violations)
		// The send block is replaced by one without the send.
		s.a.headers[s.sendBlock].Extra = []byte("reorged")
		s.a.removeLogs(sendETHTopic)
		s.a.removeLogs(liquidityBurnedTopic)
		s.a.balances[predeploys.ETHLiquidityAddr] = s.a.balances[predeploys.ETHLiquidityAddr][:1]
		inFlight, violations := checkTripwire(t, lt)
		require.Empty(t, violations, "a reorg is not a violation")
		require.Zero(t, inFlight.Sign())
	})
}
