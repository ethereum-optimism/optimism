package presets

import (
	"os"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// Running the STOCK suites against a private-interop pair.
//
// # Why this is an environment variable and not an option
//
// The plug-in thesis is that existing acceptance tests run UNCHANGED against a pair
// (op-private-interop/docs/DESIGN.md, "Testing"). "Unchanged" has to be taken literally to
// mean anything: a test that had `presets.WithPrivateInteropChain()` added to its constructor call
// is a test that was changed, and it proves only that the option compiles.
//
// A test names its preset in its own body and passes its own options, so the only place left to say
// "build this world as a pair" is outside the test. That is exactly what DEVSTACK_L2EL_KIND and
// DEVSTACK_L2CL_KIND already do for the execution and consensus layers, and this follows their
// shape: one variable, read at preset construction, honoured only by presets that can act on it.
//
//	DEVSTACK_PRIVATE_INTEROP=true go test ./tests/interop/contract/...
//
// runs the stock messenger suite with chain B replaced by a private chain and its rendering. With
// the variable unset, nothing here has any effect at all.
//
// # Honoured only where it means something
//
// The flag is applied inside collectSupportedPresetConfig, and only for presets whose supported
// option kinds include the private-interop chain. A single-chain preset, or a two-L2 preset with no
// pair wiring, is left alone rather than silently building something else -- and a preset that DOES
// support pairs but was asked for something a pair cannot provide (the in-process interop filter,
// which the pair's runtime does not wire) skips with a reason instead of running a weaker test
// under a name that promises more.

// DevstackPrivateInteropEnvVar asks every preset that can build a private-interop pair to build one.
const DevstackPrivateInteropEnvVar = "DEVSTACK_PRIVATE_INTEROP"

// DevstackPrivateInteropCadenceEnvVar overrides the pair's range cadence, in private blocks per
// range. Unset means the devstack default.
const DevstackPrivateInteropCadenceEnvVar = "DEVSTACK_PRIVATE_INTEROP_CADENCE"

// ambientPrivateInterop reports whether the environment asks for a pair.
func ambientPrivateInterop() bool {
	on, _ := strconv.ParseBool(os.Getenv(DevstackPrivateInteropEnvVar))
	return on
}

// applyAmbientPrivateInterop turns a supporting preset into a private-interop pair when the
// environment asks for one and the test did not already say so itself.
func applyAmbientPrivateInterop(t devtest.T, presetName string, cfg *sysgo.PresetConfig, supported optionKinds) {
	if cfg.PrivateInterop != nil || !ambientPrivateInterop() {
		return
	}
	if supported&optionKindPrivateInteropChain == 0 {
		t.Logger().Debug("Ignoring the ambient private-interop request: this preset builds no pair",
			"preset", presetName, "env", DevstackPrivateInteropEnvVar)
		return
	}
	if cfg.UseInteropFilter {
		t.Skipf("%s asked for the in-process interop filter, which the private-interop pair's runtime does not wire; "+
			"the filter reads a chain's own blocks, and a pair's public blocks are its rendering's", presetName)
	}
	pi := sysgo.DefaultPrivateInteropConfig()
	if raw := os.Getenv(DevstackPrivateInteropCadenceEnvVar); raw != "" {
		cadence, err := strconv.ParseUint(raw, 10, 64)
		t.Require().NoErrorf(err, "%s must be a number of private blocks per range, got %q", DevstackPrivateInteropCadenceEnvVar, raw)
		pi.MaxBlocksPerRange = cadence
	}
	t.Logger().Info("Building this preset's second L2 as a private-interop pair, on the environment's request",
		"preset", presetName, "env", DevstackPrivateInteropEnvVar, "cadence_blocks", pi.MaxBlocksPerRange)
	cfg.PrivateInterop = &pi
}
