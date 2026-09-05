package interopsmoke

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPlanSmoke(t *testing.T) {
	t.Run("no tests named is the whole suite, with the suite's defaults", func(t *testing.T) {
		plan, err := planSmoke(Config{})
		require.NoError(t, err)
		require.True(t, plan.suite)
		require.Equal(t, "All Tests", plan.name)
		var keys []string
		for _, test := range plan.tests {
			keys = append(keys, test.key)
		}
		require.Equal(t, suiteTests, keys)
		require.NotContains(t, keys, TestChainedInvalidMessage, "the cascade test doubles the suite's worst case")
		require.Equal(t, uint(1), plan.cfg.InvalidBlocks)
		require.Equal(t, uint(1), plan.cfg.InvalidTxPerBlock)
		require.Equal(t, directionBoth, plan.cfg.Direction)
		require.Equal(t, defaultReorgTimeout, plan.cfg.ReorgTimeout)
	})

	t.Run("all is the same plan", func(t *testing.T) {
		plan, err := planSmoke(Config{Tests: []string{TestAll}})
		require.NoError(t, err)
		require.True(t, plan.suite)
		require.Equal(t, "All Tests", plan.name)
	})

	t.Run("a suite keeps what the caller did set", func(t *testing.T) {
		plan, err := planSmoke(Config{InvalidBlocks: 3, InvalidTxPerBlock: 2, Direction: directionAToB, ReorgTimeout: time.Minute})
		require.NoError(t, err)
		require.Equal(t, uint(3), plan.cfg.InvalidBlocks)
		require.Equal(t, uint(2), plan.cfg.InvalidTxPerBlock)
		require.Equal(t, directionAToB, plan.cfg.Direction)
		require.Equal(t, time.Minute, plan.cfg.ReorgTimeout)
	})

	t.Run("a single test is its own report and is not defaulted", func(t *testing.T) {
		plan, err := planSmoke(Config{Tests: []string{TestInvalidMessage}})
		require.NoError(t, err)
		require.False(t, plan.suite)
		require.Equal(t, "Invalid Exec Message (reorg)", plan.name)
		// An explicit zero has to reach the test that rejects it, rather than becoming a one.
		require.Zero(t, plan.cfg.InvalidBlocks)
		require.Zero(t, plan.cfg.InvalidTxPerBlock)
		require.Zero(t, plan.cfg.ReorgTimeout)
		require.Empty(t, plan.cfg.Direction)
	})

	t.Run("several named tests report per test", func(t *testing.T) {
		plan, err := planSmoke(Config{Tests: []string{TestIdentity, TestTransfer}})
		require.NoError(t, err)
		require.True(t, plan.suite)
		require.Equal(t, "Selected Tests", plan.name)
		require.Len(t, plan.tests, 2)
	})

	t.Run("a private pair renames what would otherwise be untrue", func(t *testing.T) {
		plan, err := planSmoke(Config{Tests: []string{TestTransfer}, PrivatePairB: true})
		require.NoError(t, err)
		require.Equal(t, "Native-Unit Transfers", plan.name)
	})

	t.Run("a private pair's suite runs the one direction it can be held to", func(t *testing.T) {
		plan, err := planSmoke(Config{PrivatePairB: true})
		require.NoError(t, err)
		require.Equal(t, directionBToA, plan.cfg.Direction)
	})

	t.Run("all cannot be combined", func(t *testing.T) {
		_, err := planSmoke(Config{Tests: []string{TestAll, TestIdentity}})
		require.ErrorContains(t, err, "whole suite")
	})

	t.Run("an unknown test is named back", func(t *testing.T) {
		_, err := planSmoke(Config{Tests: []string{"sideways"}})
		require.ErrorContains(t, err, `unknown smoke test "sideways"`)
	})
}

func TestPlanRunReportsSkips(t *testing.T) {
	skipping := smokeTest{key: "skipping", name: "Skipping", fn: func(*smokeEnv) error {
		return &smokeSkip{reason: "does not apply here"}
	}}
	failing := smokeTest{key: "failing", name: "Failing", fn: func(*smokeEnv) error {
		return errors.New("boom")
	}}

	t.Run("a skip in a suite is printed and does not fail the run", func(t *testing.T) {
		var out bytes.Buffer
		env := &smokeEnv{stderr: &out}
		require.NoError(t, smokePlan{suite: true, tests: []smokeTest{skipping}}.run(env))
		require.Contains(t, out.String(), "SKIP: does not apply here")
		require.NotContains(t, out.String(), "PASS")
	})

	t.Run("a skip on its own is the run's result, for Run to report", func(t *testing.T) {
		var out bytes.Buffer
		env := &smokeEnv{stderr: &out}
		var skip *smokeSkip
		require.ErrorAs(t, smokePlan{tests: []smokeTest{skipping}}.run(env), &skip)
		require.Equal(t, "does not apply here", skip.reason)
		require.Empty(t, out.String(), "a single test does not report itself twice")
	})

	t.Run("a failure still fails the run", func(t *testing.T) {
		var out bytes.Buffer
		env := &smokeEnv{stderr: &out}
		err := smokePlan{suite: true, tests: []smokeTest{skipping, failing}}.run(env)
		require.ErrorContains(t, err, "Failing")
		require.NotContains(t, err.Error(), "Skipping")
	})
}

func TestPrivatePairDispositions(t *testing.T) {
	t.Run("the ETH bridge is skipped with its reason", func(t *testing.T) {
		var skip *smokeSkip
		require.ErrorAs(t, smokeBridge(&smokeEnv{privatePairB: true}), &skip)
		require.Contains(t, skip.reason, "native ETH interop is disabled")
	})

	t.Run("the cascade test is refused", func(t *testing.T) {
		err := smokeChainedInvalidMessage(&smokeEnv{privatePairB: true, reorgTimeout: time.Minute})
		require.ErrorIs(t, err, errChainedInvalidOnPrivatePair)
	})

	t.Run("an unnamed invalid-message direction becomes the only one that means anything", func(t *testing.T) {
		var out bytes.Buffer
		env := &smokeEnv{stderr: &out, privatePairB: true}
		require.NoError(t, env.usePrivatePairDirection())
		require.Equal(t, directionBToA, env.direction)
		require.Contains(t, out.String(), directionBToA)
	})

	t.Run("the direction that executes on the private chain is refused", func(t *testing.T) {
		for _, direction := range []string{directionAToB, directionBoth} {
			env := &smokeEnv{stderr: &bytes.Buffer{}, privatePairB: true, direction: direction}
			require.ErrorContains(t, env.usePrivatePairDirection(), "never replaced")
		}
	})

	t.Run("b-to-a is left alone", func(t *testing.T) {
		env := &smokeEnv{stderr: &bytes.Buffer{}, privatePairB: true, direction: directionBToA}
		require.NoError(t, env.usePrivatePairDirection())
		require.Equal(t, directionBToA, env.direction)
	})

	t.Run("the waits outlast the resolver's own bound", func(t *testing.T) {
		// The resolver blocks up to five minutes for the rendering to derive the block that names
		// a private message; a shorter wait here would time out about the wrong thing.
		require.Greater(t, privatePairWaitTimeout, 5*time.Minute)
	})
}

func TestRemoteChainWaitBudget(t *testing.T) {
	require.Equal(t, smokeWaitTimeout, (&remoteChain{}).waitBudget())
	require.Equal(t, privatePairWaitTimeout, (&remoteChain{waitTimeout: privatePairWaitTimeout}).waitBudget())
}
