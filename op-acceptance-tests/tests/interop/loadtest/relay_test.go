package loadtest

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// Relayer is a Spammer that initiates messages on one chain and executes them on the other.
type Relayer struct {
	source *L2
	dest   *L2
}

var _ Spammer = (*Relayer)(nil)

func NewRelayer(source, dest *L2) *Relayer {
	return &Relayer{
		source: source,
		dest:   dest,
	}
}

func (r *Relayer) Spam(t devtest.T) error {
	startE2E := time.Now()

	startInit := startE2E
	rng := rand.New(rand.NewSource(1234))
	initTx, err := r.source.Include(t, planCall(t, interop.RandomInitTrigger(rng, r.source.EventLogger, rng.Intn(2), rng.Intn(5))))
	if err != nil {
		return fmt.Errorf("include init msg: %w", err)
	}
	messageLatency.WithLabelValues(r.source.Config.ChainID.String(), "init").Observe(time.Since(startInit).Seconds())
	initMsg, err := initMsgFromReceipt(t, r.source, initTx.Receipt)
	if err != nil {
		return err
	}

	startExec := time.Now()
	if _, err = r.dest.Include(t, planExecMsg(t, initMsg, r.dest.BlockTime, r.dest.EL.Escape().EthClient())); err != nil {
		return err
	}
	endExec := time.Now()
	messageLatency.WithLabelValues(r.dest.Config.ChainID.String(), "exec").Observe(endExec.Sub(startExec).Seconds())

	messageLatency.WithLabelValues("all", "e2e").Observe(endExec.Sub(startE2E).Seconds())
	return nil
}

// TestRelaySteady runs the Relay spammer on a Steady schedule. A single execution of the Relay
// spammer sends one initating message on the source chain and one corresponding executing message
// on the destination chain.
func TestRelaySteady(gt *testing.T) {
	t, l2A, l2B := setupLoadTest(gt)
	s := NewSteady(l2B.EL.Escape().EthClient(), l2B.Config.ElasticityMultiplier(), l2B.BlockTime, WithAIMDObserver(aimdObserver{}))
	s.Run(t, NewRelayer(l2A, l2B))
}

// TestRelayBurst runs the Relay spammer on a Burst schedule. See TestRelaySteady for more details
// on the Relay spammer.
func TestRelayBurst(gt *testing.T) {
	t, l2A, l2B := setupLoadTest(gt)
	burst := NewBurst(l2B.BlockTime, WithAIMDObserver(aimdObserver{}))
	burst.Run(t, NewRelayer(l2A, l2B))
}
