package deployer

import (
	"context"
	"log/slog"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestPrepare_RejectsPrivateChains checks that prepare refuses an intent with an OP Private
// private chain, read back from intent.toml.
func TestPrepare_RejectsPrivateChains(t *testing.T) {
	_, _, dk := shared.DefaultPrivkey(t)
	l1ChainID := big.NewInt(900)
	loc, _ := testutil.LocalArtifacts(t)

	intent, st := shared.NewIntent(t, l1ChainID, dk, uint256.NewInt(1), loc, loc, standard.GasLimit)
	intent.GlobalDeployOverrides = map[string]any{
		"devFeatureBitmap": devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OPPrivateFlag),
	}
	intent.Chains[0].DeployOverrides = map[string]any{"settlementChainId": 1}
	private := shared.NewChainIntent(t, dk, l1ChainID, uint256.NewInt(2), standard.GasLimit)
	private.DeployOverrides = map[string]any{"settlementChainId": 1}
	intent.Chains = append(intent.Chains, private)

	workdir := t.TempDir()
	require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))
	require.NoError(t, st.WriteToFile(filepath.Join(workdir, "state.json")))

	err := Prepare(context.Background(), PrepareConfig{
		Workdir:    workdir,
		Logger:     testlog.Logger(t, slog.LevelWarn),
		PrivateKey: testPrivKey,
		L1RPCUrl:   testL1RPCUrl,
	})
	require.ErrorContains(t, err, "private chain "+private.ID.Hex()+": prepare does not support OP Private private chains")
}
