package fault

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
)

type Orchestrator struct {
	agents []Agent
	claims []Claim
	steps  []StepCallData

	// tracking when to exit
	claimLen, stepLen, step int
}

func NewOrchestrator(maxDepth uint64, traces []TraceProvider, names []string, agreeWithProposedOutput []bool, root Claim) Orchestrator {
	o := Orchestrator{
		agents: make([]Agent, len(traces)),
		claims: []Claim{root},
		steps:  make([]StepCallData, 0),
	}
	log.Info("Starting game", "root_letter", string(root.Value[31:]))
	for i, trace := range traces {
		o.agents[i] = NewAgent(&o, int(maxDepth), trace, &o, agreeWithProposedOutput[i], log.New("role", names[i]))
	}
	return o
}

func (o *Orchestrator) Respond(_ context.Context, response Claim) error {
	response.ContractIndex = len(o.claims)
	o.claims = append(o.claims, response)
	return nil
}

func (o *Orchestrator) Step(_ context.Context, stepData StepCallData) error {
	log.Info("Step recorded", "step", stepData)
	o.steps = append(o.steps, stepData)
	return nil
}

func (o *Orchestrator) FetchClaims(ctx context.Context) ([]Claim, error) {
	c := make([]Claim, len(o.claims))
	copy(c, o.claims)
	return c, nil
}

func (o *Orchestrator) Start() {
	for {
		for _, a := range o.agents {
			_ = a.Act()
		}
		if o.shouldExit() {
			log.Info("exiting")
			return
		}
	}
}

func (o *Orchestrator) shouldExit() bool {
	cl := o.claimLen
	sl := o.stepLen

	o.claimLen = len(o.claims)
	o.stepLen = len(o.steps)

	noProgress := o.claimLen == cl && o.stepLen == sl
	if noProgress {
		o.step = o.step + 1
	} else {
		o.step = 0
	}
	return noProgress && o.step == 1
}
