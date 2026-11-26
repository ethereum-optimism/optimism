package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// skipOpNodeOnlyPresetOnKonaNode skips when the selected CL runs Kona. The supernode presets
// call it, through selectInteropVerifier, because with op-supernode they start only op-node and
// op-supernode CLs. Use sysgo.SkipOnKonaNode for individual tests that exercise a Kona client gap.
func skipOpNodeOnlyPresetOnKonaNode(t devtest.T) {
	if sysgo.ResolveMixedL2CLKind(t).RunsKona() {
		t.Skip("skipping on kona: this preset starts only op-node and op-supernode CLs whatever DEVSTACK_L2CL_KIND says")
	}
}

// selectInteropVerifier picks the interop verifier of a supernode preset from
// DEVSTACK_SUPERNODE_KIND and records it in cfg, or skips the test with the reason the preset
// cannot run here. Every supernode preset calls it before it starts anything.
//
// With op-supernode, the default, the preset starts op-node and op-supernode CLs, so it skips
// under a kona CL kind. With lokahi, each chain runs a kona-node that follows a lokahi process,
// so the preset skips under op-node, and where it needs what the lokahi backend lacks:
// lokahiUnsupported names that for the whole preset, and an option op-supernode alone has skips
// as well.
func selectInteropVerifier(t devtest.T, cfg *sysgo.PresetConfig, lokahiUnsupported string) {
	if sysgo.ResolveSupernodeKind(t) != sysgo.SupernodeLokahi {
		skipOpNodeOnlyPresetOnKonaNode(t)
		return
	}
	if !sysgo.ResolveMixedL2CLKind(t).RunsKona() {
		t.Skip("skipping on op-node: with DEVSTACK_SUPERNODE_KIND=lokahi this preset starts only kona CLs whatever DEVSTACK_L2CL_KIND says")
	}
	if lokahiUnsupported != "" {
		t.Skipf("skipping on lokahi: %s", lokahiUnsupported)
	}
	if cfg.InteropLogBackfillDepth != 0 {
		t.Skip("skipping on lokahi: the log backfill depth configures op-supernode's log backfill, which lokahi does not have")
	}
	cfg.InteropVerifier = sysgo.SupernodeLokahi
}
