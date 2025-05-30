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

	numInitTxs := uint64(1)
	if numInitTxsStr, ok := os.LookupEnv(numInitTxsEnvVar); ok {
		var err error
		numInitTxs, err = strconv.ParseUint(numInitTxsStr, 10, 64)
		t.Require().NoError(err)
	}

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
		SpamInteropTxs(t, numInitTxs, L2A, L2B, sys.Supervisor)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		SpamInteropTxs(t, numInitTxs, L2B, L2A, sys.Supervisor)
	}()
}

func SpamInteropTxs(t devtest.T, numInitTxs uint64, source *L2, dest *L2, supervisor *dsl.Supervisor) {
	var relayWg sync.WaitGroup
	defer relayWg.Wait()
	msgsCh := make(chan []types.Message, 100)
	defer close(msgsCh) // Must be defer'd after the relayWg.Wait() above.

	// Spam executing messages.
	relayWg.Add(1)
	go func() {
		defer relayWg.Done()
		dest.Funder.NewFundedEOA(eth.MillionEther.Mul(100))
		relayers := []Relayer{
			NewValidRelayer(dest.Funder, dest.EL, supervisor),
			NewDelayedRelayer(NewValidRelayer(dest.Funder, dest.EL, supervisor), &relayWg, time.Minute),
			NewInvalidRelayer(dest.Funder, dest.EL, makeInvalidChainID),
			NewInvalidRelayer(dest.Funder, dest.EL, makeInvalidBlockNumber),
			NewInvalidRelayer(dest.Funder, dest.EL, makeInvalidLogIndex),
			NewInvalidRelayer(dest.Funder, dest.EL, makeInvalidOrigin),
			NewInvalidRelayer(dest.Funder, dest.EL, makeInvalidPayloadHash),
			NewInvalidRelayer(dest.Funder, dest.EL, makeInvalidTimestamp),
		}
		for msgs := range msgsCh {
			for _, relayer := range relayers {
				relayWg.Add(1)
				go func() {
					defer relayWg.Done()
					relayer.Relay(t, msgs)
				}()
			}
			if len(msgs) >= cap(msgs)/2 {
				t.Logger().Warn("Messages buffer is at least half full", "len", len(msgs), "cap", cap(msgs))
			}
		}
	}()

	// Spam initiating messages.
	eventLogger := source.Funder.NewFundedEOA(eth.OneEther).DeployEventLogger()
	initiators := []Initiator{
		NewManyMsgsInitiator(source.Funder, source.EL, eventLogger),
		NewLargeMsgInitiator(source.Funder, source.EL, eventLogger),
	}
	var initWg sync.WaitGroup
	defer initWg.Wait()
	for i := range numInitTxs {
		initWg.Add(1)
		go func() {
			defer initWg.Done()
			msgsCh <- initiators[i%uint64(len(initiators))].Initiate(t)
		}()
	}
}
