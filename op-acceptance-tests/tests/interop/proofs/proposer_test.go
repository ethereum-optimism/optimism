package proofs

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

var (
	gameCountFn        = w3.MustNewFunc("gameCount()", "uint256")
	gameAtIndexFn      = w3.MustNewFunc("gameAtIndex(uint256)", "uint32, uint64, address")
	rootClaimFn        = w3.MustNewFunc("rootClaim()", "bytes32")
	l2SequenceNumberFn = w3.MustNewFunc("l2SequenceNumber()", "uint256")
)

func TestProposer(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)

	// The DGF is shared across all L2 networks. So pick the first one.
	disputeGameFactoryAddr := sys.L2Networks()[0].DisputeGameFactoryProxyAddr()
	l1Client := sys.L1EL.EthClient()

	var gameCount *big.Int
	newFuncCallDSL(t, l1Client, gameCountFn).
		WithReturns(&gameCount).
		Call(disputeGameFactoryAddr)

	t.Require().Eventually(func() bool {
		var newGameCount *big.Int
		newFuncCallDSL(t, l1Client, gameCountFn).
			WithReturns(&newGameCount).
			Call(disputeGameFactoryAddr)
		check := newGameCount.Cmp(gameCount) > 0
		t.Logf("waiting for game count to increase. current=%d new=%d", gameCount, newGameCount)
		return check
	}, time.Minute*10, time.Second*5)

	var gameType uint32
	var timestamp uint64
	var gameAddress common.Address
	newFuncCallDSL(t, l1Client, gameAtIndexFn).
		WithArgs(gameCount).
		WithReturns(&gameType, &timestamp, &gameAddress).
		Call(disputeGameFactoryAddr)

	var rootClaim [32]byte
	newFuncCallDSL(t, l1Client, rootClaimFn).WithReturns(&rootClaim).Call(gameAddress)

	var l2SequenceNumber *big.Int
	newFuncCallDSL(t, l1Client, l2SequenceNumberFn).WithReturns(&l2SequenceNumber).Call(gameAddress)

	superRoot := sys.Supervisor.FetchSuperRootAtTimestamp(l2SequenceNumber.Uint64())
	t.Require().Equal(superRoot.SuperRoot[:], rootClaim[:])
}
