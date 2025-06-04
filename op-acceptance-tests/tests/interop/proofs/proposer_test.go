package proofs

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
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

	deployment := sys.L2Networks()[0].Escape().Deployment()
	disputeGameFactoryAddr := deployment.DisputeGameFactoryProxyAddr()

	l1Client := sys.L1EL.Escape().EthClient()

	var gameCount *big.Int
	newFuncCallDSL(t, l1Client, gameCountFn).
		WithReturns(&gameCount).
		Call(disputeGameFactoryAddr)

	waitCtx, cancel := context.WithTimeout(t.Ctx(), time.Minute*30)
	err := wait.For(waitCtx, time.Second*5, func() (bool, error) {
		var newGameCount *big.Int
		newFuncCallDSL(t, l1Client, gameCountFn).
			WithReturns(&newGameCount).
			Call(disputeGameFactoryAddr)
		if newGameCount.Cmp(gameCount) > 0 {
			t.Logf("game count increased from %d to %d", gameCount, newGameCount)
			return true, nil
		}
		return false, nil
	})
	cancel()
	t.Require().NoError(err, "waiting for game count to increase")

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
