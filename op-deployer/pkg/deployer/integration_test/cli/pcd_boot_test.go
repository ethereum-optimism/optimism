package cli

import (
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestCLIPCDBoot(t *testing.T) {
	opRethPath := requirePCDOpReth(t)
	prestatePath := pcdPrestateArtifactPath(t)
	prestate := requirePCDPrestate(t, prestatePath)

	// Backdate L1 so the L2 genesis time is in the past when the node starts.
	// The sequencer cannot create block 1 before that time.
	backdate := time.Duration(pcdGenesisTimeOffset)*time.Second + time.Hour
	requestedL1Timestamp := uint64(time.Now().Add(-backdate).Unix())
	chainIDs := []common.Hash{uint256.NewInt(1).Bytes32()}
	journey := newPCDJourneyFixtureWithAnvilOptions(
		t,
		chainIDs,
		devnet.WithTimestamp(requestedL1Timestamp),
		devnet.WithHardfork("prague"),
	)
	l1Genesis, err := journey.l1Client.BlockByNumber(t.Context(), big.NewInt(0))
	require.NoError(t, err)
	require.Equal(t, requestedL1Timestamp, l1Genesis.Time(), "Anvil must use the requested back-dated timestamp")
	t.Logf("PCD runtime inputs: op-reth=%s Anvil hardfork=prague requested_timestamp=%d observed_timestamp=%d", opRethPath, requestedL1Timestamp, l1Genesis.Time())

	journey.bootstrapOPCM()
	journey.runInit(embedded.GameTypeSuperCannonKona)
	journey.runPrepare()
	journey.runInspect()
	journey.writeDependencySet()
	journey.runPrestate(prestate)
	journey.runContinue()

	committedWorkdir := journey.cloneCommittedWorkdir()
	artifacts := pcdArtifactPaths(committedWorkdir, chainIDs)
	require.Len(t, artifacts, 1)
	dependencySetPath := filepath.Join(committedWorkdir, "interop-depset.json")
	result := bootPCDFromArtifacts(t, pcdBootConfig{
		l1RPC:             journey.l1RPC,
		l1Client:          journey.l1Client,
		genesisPath:       artifacts[0].genesisPath,
		rollupPath:        artifacts[0].rollupPath,
		dependencySetPath: dependencySetPath,
	})

	require.Equalf(
		t,
		result.expectedGenesisHash,
		result.block0.Hash(),
		"L2 block 0 hash differs for committed genesis artifact %s: expected %s, observed %s",
		artifacts[0].genesisPath,
		result.expectedGenesisHash,
		result.block0.Hash(),
	)
	require.Equalf(
		t,
		result.genesisTime,
		result.block0.Time(),
		"L2 block 0 timestamp differs for committed rollup artifact %s: expected %d, observed %d",
		artifacts[0].rollupPath,
		result.genesisTime,
		result.block0.Time(),
	)
	require.Equalf(
		t,
		result.genesisTime+result.l2BlockTime,
		result.block1.Time(),
		"L2 block 1 timestamp differs for committed rollup artifact %s: expected %d, observed %d",
		artifacts[0].rollupPath,
		result.genesisTime+result.l2BlockTime,
		result.block1.Time(),
	)
	t.Logf(
		"PCD runtime boot reached L2 block 1 in %s from genesis=%s rollup=%s depset=%s",
		result.bootToBlock1,
		artifacts[0].genesisPath,
		artifacts[0].rollupPath,
		dependencySetPath,
	)
}
