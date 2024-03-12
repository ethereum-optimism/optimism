package test

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type PlannedTestDef struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"params"`

	Sub []*PlannedTestDef `json:"sub"`

	sync.RWMutex `json:"-"`
}

func (p *PlannedTestDef) Param(k string) (v string, ok bool) {
	p.RLock()
	defer p.RUnlock()
	if p.Params == nil {
		return
	}
	v, ok = p.Params[k]
	return
}

func (p *PlannedTestDef) SetParam(k, v string) {
	p.Lock()
	defer p.Unlock()
	if p.Params == nil {
		p.Params = make(map[string]string)
	}
	p.Params[k] = v
}

func (p *PlannedTestDef) AddSub(sub *PlannedTestDef) {
	p.Lock()
	defer p.Unlock()
	p.Sub = append(p.Sub, sub)
}

type PlanDef struct {
	Tests      []*PlannedTestDef `json:"tests"`
	ImportPath string            `json:"importPath"`
	sync.Mutex `json:"-"`
}

// plan is the accumulated collection of test-plans.
// This is a global, but test-specific, thus initialized per test-binary (per Go package).
// This maps test-name to plan definition.
// Once all tests are completed, a post-processing function should be called by MainStart.
var plan = &PlanDef{}

// SavePlan adds the plan of a single Go test to the package test plan.
func SavePlan(testPlan *PlannedTestDef) {
	checkMain()
	plan.Lock()
	defer plan.Unlock()
	plan.Tests = append(plan.Tests, testPlan)
}

// WritePlans writes the currently accumulated package test-plan to disk
func WritePlans(dest string) error {
	plan.Lock()
	defer plan.Unlock()

	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to open test-plan file: %w", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(plan); err != nil {
		return fmt.Errorf("failed to write/encode plan: %w", err)
	}
	return nil
}
