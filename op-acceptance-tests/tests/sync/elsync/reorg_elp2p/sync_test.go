package reorg_elp2p

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestUnsafeReorgToSafe(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	require := t.Require()
	logger := t.Logger()

	delta := uint64(10)
	dsl.CheckAll(t,
		sys.L2CLB.AdvancedFn(types.LocalUnsafe, delta, 30),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 30),
	)

	sys.L2CLB.Stop()
	sys.L2ELB.DisconnectPeerWith(sys.L2EL)

	genesis := sys.L2ELB.BlockRefByNumber(0)
	unsafe := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	safe := sys.L2ELB.BlockRefByLabel(eth.Safe)
	require.GreaterOrEqual(unsafe.Number, safe.Number)

	// Make unsafe head diff between sequencer EL and verifier EL
	sys.L2CL.AdvancedFn(types.LocalUnsafe, 3, 30)

	logger.Info("Unsafe blocks exists which not yet promoted to safe", "unsafe", unsafe.Number, "safe", safe.Number)

	// Trigger Reorg to block diverged from sequencer
	res := sys.L2ELB.NewPayloadWithFault(sys.L2EL, safe.Number).IsValid()
	sys.L2ELB.ForkchoiceUpdateRaw(res.BlockHash, res.BlockHash, genesis.Hash, nil).IsValid()

	require.Equal(safe.Number, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)
	require.Equal(safe.Number, sys.L2ELB.BlockRefByLabel(eth.Safe).Number)
	require.NotEqual(safe.Hash, sys.L2ELB.BlockRefByLabel(eth.Safe).Hash)

	// Trigger Reorg to sequencer produced block
	sys.L2ELB.NewPayload(sys.L2EL, safe.Number).IsValid()
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, safe.Number, safe.Number, 0, nil).IsValid()
	require.Equal(safe.Number, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)
	require.Equal(safe.Number, sys.L2ELB.BlockRefByLabel(eth.Safe).Number)
	require.Equal(safe.Hash, sys.L2ELB.BlockRefByLabel(eth.Safe).Hash)

	// Peer again for EL Sync preparation
	sys.L2ELB.PeerWith(sys.L2EL)

	// Trigger EL Sync to fill in the gap
	target := sys.L2EL.BlockRefByNumber(unsafe.Number + 1)
	targetHash := target.Hash
	targetHash[0] = targetHash[0] + 1 // inject fault
	// op-geth logs
	// 	t=2025-10-15T00:40:26.803+0900 lvl=warn msg="Fetching the unknown forkchoice head from network" hash=0x84da371982ce4edcc1eb2fe7cbf2f886265619637645e64a25e13898275b9a30 global=true
	//  t=2025-10-15T00:40:26.803+0900 lvl=debug msg="Attempting to retrieve sync target" peer=dabe984226b4e6ab1e1debfd4dd94d669aba83288d3e95038fef7df26e2d0f16 hash=0x84da371982ce4edcc1eb2fe7cbf2f886265619637645e64a25e13898275b9a30 global=true
	//  t=2025-10-15T00:40:26.803+0900 lvl=debug msg="Fetching batch of headers" id=dabe984226b4e6ab1e1debfd4dd94d669aba83288d3e95038fef7df26e2d0f16 conn=staticdial count=1 fromhash=0x84da371982ce4edcc1eb2fe7cbf2f886265619637645e64a25e13898275b9a30 skip=0 reverse=false global=true
	//  t=2025-10-15T00:40:26.803+0900 lvl=warn msg="Could not retrieve unknown head from peers" global=true
	sys.L2ELB.ForkchoiceUpdateRaw(targetHash, targetHash, genesis.Hash, nil).WaitUntilValid(10)
}
