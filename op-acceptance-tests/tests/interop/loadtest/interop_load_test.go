package loadtest

import (
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const numInitTxsEnvVar = "NAT_LOADTEST_INITTXS"

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleInterop())
}

type L2 struct {
	Chain  *dsl.L2Network
	EL     *dsl.L2ELNode
	Funder *dsl.Funder
}

func TestLoad(gt *testing.T) {
	if testing.Short() {
		gt.Skip("skipping load test in short mode")
	}
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)

	var numInitTxs uint64
	if numInitTxsStr, ok := os.LookupEnv(numInitTxsEnvVar); ok {
		var err error
		numInitTxs, err = strconv.ParseUint(numInitTxsStr, 10, 64)
		t.Require().NoError(err)
	}
	t.Gate().NotZero(numInitTxs, "load test only makes sense when "+numInitTxsEnvVar+" is nonzero")

	l2ELA := sys.L2ChainA.PublicRPC()
	L2A := &L2{
		Chain:  sys.L2ChainA,
		EL:     l2ELA,
		Funder: dsl.NewFunder(sys.Wallet, sys.FaucetA, l2ELA),
	}
	l2ELB := sys.L2ChainB.PublicRPC()
	L2B := &L2{
		Chain:  sys.L2ChainB,
		EL:     l2ELB,
		Funder: dsl.NewFunder(sys.Wallet, sys.FaucetB, l2ELB),
	}

	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		SpamInteropTxs(t, &wg, numInitTxs, L2A, L2B, sys.Supervisor)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		SpamInteropTxs(t, &wg, numInitTxs, L2B, L2A, sys.Supervisor)
	}()
}

func SpamInteropTxs(t devtest.T, wg *sync.WaitGroup, numInitTxs uint64, source *L2, dest *L2, supervisor *dsl.Supervisor) {
	msgsCh := make(chan []types.Message, 100)
	defer close(msgsCh)

	// Spam executing messages.
	wg.Add(1)
	go func() {
		defer wg.Done()
		executors := []Executor{
			NewValidExecutor(dest.Funder, dest.EL, supervisor),
			NewDelayedExecutor(NewValidExecutor(dest.Funder, dest.EL, supervisor), wg, time.Minute),
			NewInvalidExecutor(dest.Funder, dest.EL, makeInvalidChainID),
			NewInvalidExecutor(dest.Funder, dest.EL, makeInvalidBlockNumber),
			NewInvalidExecutor(dest.Funder, dest.EL, makeInvalidLogIndex),
			NewInvalidExecutor(dest.Funder, dest.EL, makeInvalidOrigin),
			NewInvalidExecutor(dest.Funder, dest.EL, makeInvalidPayloadHash),
			NewInvalidExecutor(dest.Funder, dest.EL, makeInvalidTimestamp),
		}
		for msgs := range msgsCh {
			for _, executor := range executors {
				wg.Add(1)
				go func(executor Executor, msgs []types.Message) {
					defer wg.Done()
					executor.Execute(t, msgs)
				}(executor, msgs)
			}
		}
	}()

	// Spam initiating messages.
	eventLogger := source.Funder.NewFundedEOA(eth.OneEther).DeployEventLogger()
	initiators := []Initiator{
		NewManyMsgsInitiator(source.Funder, source.EL, eventLogger),
		NewLargeMsgInitiator(source.Funder, source.EL, eventLogger),
	}
	for i := range numInitTxs {
		msgsCh <- initiators[i%uint64(len(initiators))].Initiate(t)
	}
}
