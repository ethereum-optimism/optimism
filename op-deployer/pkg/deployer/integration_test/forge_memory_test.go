package integration_test

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
)

// TestForgeRunMemory fails when a forge run of a genesis apply, which runs every L1 and L2 genesis
// script on the local contracts build, nears forge.RunMemory, the memory that a forge slot counts
// on, so that the slots do not let forge runs outgrow the memory of a CI container. It is not
// parallel: the parallel tests' forge runs would make its result depend on timing, and on Linux this
// process's own peak, which the parallel tests raise, hides forge's.
func TestForgeRunMemory(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("the peak memory of a process is known only on Linux and macOS")
	}
	opts, _, _ := setupGenesisChain(t, devnet.DefaultChainID)
	require.NoError(t, deployer.ApplyPipeline(t.Context(), opts))

	peak := forge.PeakRunMemory()
	require.NotZero(t, peak, "forge's peak memory is unknown: this process peaked higher")
	// Transparent huge pages raise forge's peak, so the log says whether they are on.
	thp, _ := os.ReadFile("/sys/kernel/mm/transparent_hugepage/enabled")
	t.Logf("forge peaked at %d MiB of a %d MiB slot; transparent huge pages: %s", peak>>20, forge.RunMemory>>20, strings.TrimSpace(string(thp)))
	const step = 64 << 20
	enough := (peak*10/9 + step - 1) / step * step
	require.LessOrEqualf(t, peak, uint64(forge.RunMemory)/10*9,
		"a forge run peaked at %d MiB, above 90%% of forge.RunMemory (%d MiB): if op-deployer warned that it cannot copy the contracts build without the AST, fix that; otherwise raise RunMemory to %d MiB",
		peak>>20, forge.RunMemory>>20, enough>>20)
}
