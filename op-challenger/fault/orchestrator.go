package fault

import (
	"context"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type Orchestrator struct {
	agents    []Agent
	outputChs []chan Claim
	responses chan Claim
}

func NewOrchestrator(maxDepth uint64, traces []TraceProvider, names []string, root Claim) Orchestrator {
	o := Orchestrator{
		responses: make(chan Claim, 100),
		outputChs: make([]chan Claim, len(traces)),
		agents:    make([]Agent, len(traces)),
	}
	log.Info("Starting game", "root_letter", string(root.Value[31:]))
	for i, trace := range traces {
		game := NewGameState(root, maxDepth)
		o.agents[i] = NewAgent(game, int(maxDepth), trace, &o, log.New("role", names[i]))
		o.outputChs[i] = make(chan Claim)
	}
	return o
}

func (o *Orchestrator) Respond(_ context.Context, response Claim) error {
	o.responses <- response
	return nil
}

func (o *Orchestrator) Step(ctx context.Context, stepData StepCallData) error {
	return nil
}

func (o *Orchestrator) Start() {
	for i := 0; i < len(o.agents); i++ {
		go runAgent(&o.agents[i], o.outputChs[i])
	}
	o.responderThread()
}

func runAgent(agent *Agent, claimCh <-chan Claim) {
	for {
		agent.PerformActions()
		// Note: Should drain the channel here
		claim := <-claimCh
		_ = agent.AddClaim(claim)

	}
}

func (o *Orchestrator) responderThread() {
	timer := time.NewTimer(200 * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case resp := <-o.responses:
			timer.Reset(200 * time.Millisecond)
			for _, ch := range o.outputChs {
				// Copy it. Should be immutable, but be sure.
				resp := resp
				ch <- resp
			}
		case <-timer.C:
			os.Exit(0)
		}

	}
}
