package config

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
)

// TestDeployConfigConcurrentMutationIsolated stresses DeployConfig() from many
// goroutines that each mutate the returned value. Each call must return an
// independent *genesis.DeployConfig: mutations made by one goroutine must not
// be observable by any other, and the cached source must remain unchanged.
//
// Run with -race to catch any shared state that might be silently aliased.
//
// This is the regression guard for CircleCI failures where the matrix runner
// for rust/kona/tests/proofs.TestOperatorFeeConsistency crashed inside
// (*DeployConfig).Copy with "concurrent map writes". The previous
// implementation handed out copies via a json.Marshal of a single shared
// *DeployConfig; the new implementation snapshots the JSON once at init and
// each caller unmarshals its own value, so no state is shared.
func TestDeployConfigConcurrentMutationIsolated(t *testing.T) {
	const testType AllocType = "test-race"

	src := &genesis.DeployConfig{}
	src.L1ChainID = 900
	src.L2ChainID = 901
	src.L2BlockTime = 1
	originalIsthmus := hexutil.Uint64(42)
	src.L2GenesisIsthmusTimeOffset = &originalIsthmus

	raw, err := json.Marshal(src)
	require.NoError(t, err)

	mtx.Lock()
	deployConfigBytesByType[testType] = raw
	mtx.Unlock()
	t.Cleanup(func() {
		mtx.Lock()
		delete(deployConfigBytesByType, testType)
		mtx.Unlock()
	})

	const goroutines = 64
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				dc := DeployConfig(testType)
				// Mutate scalar and pointer fields. If any state is shared
				// across callers, -race will flag this or a later read will
				// observe an unexpected value.
				dc.L1ChainID = uint64(g*iterations + i)
				v := hexutil.Uint64(uint64(g*iterations + i))
				dc.L2GenesisIsthmusTimeOffset = &v

				require.Equal(t, uint64(901), dc.L2ChainID)
				require.NotNil(t, dc.L2GenesisIsthmusTimeOffset)
				require.Equal(t, v, *dc.L2GenesisIsthmusTimeOffset)
			}
		}(g)
	}
	wg.Wait()

	// The cached source must be untouched: a fresh read should still see the
	// original L1ChainID and Isthmus offset.
	dc := DeployConfig(testType)
	require.Equal(t, uint64(900), dc.L1ChainID)
	require.NotNil(t, dc.L2GenesisIsthmusTimeOffset)
	require.Equal(t, originalIsthmus, *dc.L2GenesisIsthmusTimeOffset)
}
