package validate

import (
	"bytes"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestValidateAutoSkipsPrivateChains(t *testing.T) {
	settlementID := common.BigToHash(big.NewInt(901))
	privateID := common.BigToHash(big.NewInt(902))
	st := &state.State{AppliedIntent: &state.Intent{
		GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OPPrivateFlag),
		},
		Chains: []*state.ChainIntent{
			{ID: settlementID, DeployOverrides: map[string]any{"settlementChainId": 901}},
			{ID: privateID, DeployOverrides: map[string]any{"settlementChainId": 901}},
		},
	}}
	st.SetChainContracts(settlementID, addresses.OpChainContracts{}, true)
	st.SetChainContracts(privateID, addresses.OpChainContracts{}, true)
	workdir := t.TempDir()
	require.NoError(t, st.WriteToFile(filepath.Join(workdir, "state.json")))

	// Without --fail, contracts the unreachable L1 cannot confirm are only warnings.
	var out bytes.Buffer
	app := cli.NewApp()
	app.Writer = &out
	app.Commands = Commands()
	require.NoError(t, app.Run([]string{"op-deployer", "auto", "--l1-rpc-url", "http://127.0.0.1:1", "--workdir", workdir}))

	logs := out.String()
	require.Contains(t, logs, `msg="Validating chain" chain-id=`+settlementID.Hex())
	require.NotContains(t, logs, `msg="Validating chain" chain-id=`+privateID.Hex())
	require.Contains(t, logs, `msg="Skipping private chain, it shares the L1 contracts of its settlement chain and is validated with it" chain-id=`+privateID.Hex()+` settlement-chain-id=`+settlementID.Hex())
}
