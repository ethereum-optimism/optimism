package fault

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type Orchestrator struct {
	agents    []Agent
	outputChs []chan Claim
	responses chan Claim
}

func NewOrchestrator(maxDepth uint64, traces []TraceProvider, names []string, root, counter Claim) Orchestrator {
	o := Orchestrator{
		responses: make(chan Claim, 100),
		outputChs: make([]chan Claim, len(traces)),
		agents:    make([]Agent, len(traces)),
	}
	PrettyPrintAlphabetClaim("init", root)
	PrettyPrintAlphabetClaim("init", counter)
	for i, trace := range traces {
		game := NewGameState(root)
		_ = game.Put(counter)
		o.agents[i] = NewAgent(game, int(maxDepth), trace, &o, log.New("role", names[i]))
		o.outputChs[i] = make(chan Claim)
	}
	return o
}

func (o *Orchestrator) Respond(_ context.Context, response Claim) error {
	o.responses <- response
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

func PrettyPrintAlphabetClaim(name string, claim Claim) {
	value := claim.Value
	idx := value[30]
	letter := value[31]
	par_letter := claim.Parent.Value[31]
	if claim.IsRoot() {
		fmt.Printf("%s\ttrace %v letter %c\n", name, idx, letter)
	} else {
		fmt.Printf("%s\ttrace %v letter %c is attack %v parent letter %c\n", name, idx, letter, !claim.DefendsParent(), par_letter)
	}

}
