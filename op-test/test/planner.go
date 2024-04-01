package test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"slices"
	"strings"
	"sync"
	"testing"

	"golang.org/x/exp/slog"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// Plan is the default entry-point to use for op-test tests.
// It wraps the Go test framework to provide test utils and parametrization features.
//
// Test packages using op-test require a TestMain(m *testing.M) function that calls Main(m).
func Plan(t *testing.T, fn func(t Planner)) {
	checkMain()

	ctx := packageCtx()
	t.Run("main", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		settings := GetTestSettings(ctx)

		var testPlan *PlannedTestDef
		if settings.presetPlan != nil {
			testPlan = settings.presetPlan.GetPlan(t.Name())
			if testPlan == nil {
				t.Skip("preset package-plan was specified, but no plan for test-case was found")
			}
		} else {
			testPlan = &PlannedTestDef{
				Name: t.Name(),
			}
		}

		imp := &testImpl{
			T:            t,
			ctx:          ctx,
			logLvl:       slog.LevelError,
			plan:         testPlan,
			buildingPlan: settings.buildingPlan,
			runningPlan:  settings.runningPlan,
		}
		fn(imp)

		SavePlan(imp.plan)
	})
}

type parameterSelection struct {
	name    string
	options []string
}

// testImpl wraps the regular Go test framework to implement the Testing interface.
type testImpl struct {
	*testing.T

	// nil if no parent-test
	parent *testImpl

	// index of the test, compared to its sibling tests, assuming there is a parent test
	subIndex uint64

	// number of sub-tests that we have passed so far
	currentSubTests uint64

	// ctx is scoped to the execution of this test-scope.
	ctx context.Context

	plan *PlannedTestDef

	// extend plan if true, leave as-is if false
	buildingPlan bool
	// Run the test functions if true, only traverse if false.
	// If buildingPlan is true, then immediately execute after building the plan.
	runningPlan bool

	logLvl slog.Level

	loggerOnce sync.Once
	logger     log.Logger

	// First-seen parameterSelection, which can be exhausted at the end of the test.
	parameterSelection *parameterSelection
}

var _ Planner = (*testImpl)(nil)

// Ctx implements Testing.Ctx
func (imp *testImpl) Ctx() context.Context {
	return imp.ctx
}

// Logger implements Testing.Logger
func (imp *testImpl) Logger() log.Logger {
	imp.loggerOnce.Do(func() {
		imp.logger = testlog.Logger(imp, imp.logLvl)
	})
	return imp.logger
}

// Parameter implements Testing.Parameter
func (imp *testImpl) Parameter(name string) (value string, ok bool) {
	// recurse up the test-stack, to look for the parameter
	p := imp
	for p != nil {
		if p.plan == nil {
			p = p.parent
			continue
		}
		v, ok := p.plan.Param(name)
		if !ok {
			p = p.parent
			continue
		}
		return v, true
	}
	return "", false
}

// Run implements Planner.Run
func (imp *testImpl) Run(name string, fn func(t Executor)) {
	imp.orderedSubTest(name, func(t *testImpl) {
		if !t.runningPlan {
			t.Skip("not running")
		}
		t.Log("test!", t.Name())
		//fn(t) TODO
	})
}

// Plan implements Planner.Plan
func (imp *testImpl) Plan(name string, fn func(t Planner)) {
	imp.orderedSubTest(name, func(t *testImpl) {
		fn(t)
	})
}

func (imp *testImpl) orderedSubTest(name string, fn func(t *testImpl)) {
	imp.currentSubTests += 1

	var subPlan *PlannedTestDef
	if imp.buildingPlan { // don't consume existing plan if we are building the plan
		subPlan = &PlannedTestDef{Name: name}
		imp.plan.AddSub(subPlan)
	} else {
		// if we have a plan, take the sub-test entry
		require.LessOrEqual(imp.T, imp.currentSubTests, uint64(len(imp.plan.Sub)))
		subPlan = imp.plan.Sub[imp.currentSubTests-1]
	}
	imp.subTest(subPlan, fn)
}

func (imp *testImpl) subTest(subPlan *PlannedTestDef, fn func(t *testImpl)) {
	ctx := imp.Ctx()
	imp.T.Run(subPlan.Name, func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		subScope := &testImpl{
			parent:       imp,
			T:            t,
			ctx:          ctx,
			logLvl:       imp.logLvl,
			plan:         subPlan,
			buildingPlan: imp.buildingPlan,
			runningPlan:  imp.runningPlan,
		}

		fn(subScope)
	})
}

// Select implements Testing.Select
func (imp *testImpl) Select(name string, options []string, fn func(t Planner)) {
	// Check if the choice was already made
	current, ok := imp.Parameter(name)
	hasWildcard := slices.Contains(options, "*")
	if ok {
		if !hasWildcard && !slices.Contains(options, current) {
			imp.T.Fatalf("presented with choice %q, with options %q, but already assumed %q",
				name, strings.Join(options, ", "), current)
		}
		fn(imp)
		return
	}

	// get the parameter selector
	selector := GetParameterSelector(imp.ctx)
	// select what option(s) we should go with
	selectedOptions := selector.Select(name, options)
	if len(selectedOptions) == 0 {
		imp.T.Skipf("None of the options for parameter %q where selected, skipping test!", name)
	}
	if !hasWildcard {
		// verify the selected options are valid (a subset of the suggested options)
		seen := make(map[string]struct{})
		for _, opt := range options {
			seen[opt] = struct{}{}
		}
		for _, opt := range selectedOptions {
			if _, ok := seen[opt]; !ok {
				imp.T.Fatalf("Test selector selected option %q for %q, but it is was not in the set of selectable options!", opt, name)
			}
		}
	}

	imp.orderedSubTest(name, func(t *testImpl) {
		for _, opt := range options {
			subName := name + "=" + opt
			// TODO: maybe hash the option-value, if it's too large to encode in the test-name
			subPlan := &PlannedTestDef{Name: subName}
			subPlan.SetParam(name, opt)
			t.plan.AddSub(subPlan)
			t.subTest(subPlan, func(t *testImpl) {
				fn(t)
			})
		}
	})
}

type Settings struct {
	buildingPlan bool
	runningPlan  bool
	presetPlan   *PlanDef
}

// We make the Settings available through the ctx, instead of a global,
// so the Settings logic itself can be overridden and tested.
type testSettingsCtxKey struct{}

func GetTestSettings(ctx context.Context) *Settings {
	sel := ctx.Value(testSettingsCtxKey{})
	if sel == nil {
		return nil
	}
	v, ok := sel.(*Settings)
	if !ok {
		panic(fmt.Errorf("bad test settings: %v", v))
	}
	return v
}
