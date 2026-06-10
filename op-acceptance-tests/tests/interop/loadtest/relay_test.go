package loadtest

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/backpressure"
)

// RelaySpammer initiates messages on one chain and executes them on the other.
type RelaySpammer struct {
	t      devtest.T
	source *L2
	dest   *L2
}

var _ backpressure.Task = (*RelaySpammer)(nil)

func NewRelaySpammer(t devtest.T, source, dest *L2) *RelaySpammer {
	return &RelaySpammer{
		t:      t,
		source: source,
		dest:   dest,
	}
}

func (r *RelaySpammer) Do(ctx context.Context) error {
	startE2E := time.Now()

	startInit := startE2E
	rng := rand.New(rand.NewSource(1234))
	initTx, err := r.source.Include(r.t, planCall(r.t, interop.RandomInitTrigger(rng, r.source.EventLogger, rng.Intn(2), rng.Intn(5))))
	if err != nil {
		return fmt.Errorf("include init msg: %w", err)
	}
	messageLatency.WithLabelValues(r.source.Config.ChainID.String(), "init").Observe(time.Since(startInit).Seconds())
	initMsg, err := initMsgFromReceipt(r.t, r.source, initTx.Receipt)
	if err != nil {
		return err
	}

	startExec := time.Now()
	if _, err = r.dest.Include(r.t, planExecMsg(r.t, initMsg, r.dest.BlockTime, r.dest.EL.Escape().EthClient())); err != nil {
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
	s := backpressure.NewSteady(t.Logger(), l2B.BlockTime, l2B.EL.Escape().EthClient(), l2B.Config.ElasticityMultiplier())
	aimd := backpressure.NewAIMD(backpressure.WithAIMDObserver(aimdObserver{}))
	s.Run(t.Ctx(), aimd, NewRelaySpammer(t, l2A, l2B))
}

// TestRelayBurst runs the Relay spammer on a Burst schedule. See TestRelaySteady for more details
// on the Relay spammer.
func TestRelayBurst(gt *testing.T) {
	t, l2A, l2B := setupLoadTest(gt)
	burst := backpressure.NewBurst(t.Logger())
	burst.Run(t.Ctx(), backpressure.NewAIMD(), NewRelaySpammer(t, l2A, l2B))
}
