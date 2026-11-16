package verify

import (
	"log/slog"
	"math/big"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGetBundleFromFileSkipsPrivateChains(t *testing.T) {
	settlementID := common.BigToHash(big.NewInt(901))
	privateID := common.BigToHash(big.NewInt(902))
	contracts := addresses.OpChainContracts{}
	contracts.SystemConfigProxy = common.HexToAddress("0xaa")
	contracts.OptimismPortalProxy = common.HexToAddress("0xbb")

	st := &state.State{AppliedIntent: &state.Intent{
		GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OPPrivateFlag),
		},
		Chains: []*state.ChainIntent{
			{ID: settlementID, DeployOverrides: map[string]any{"settlementChainId": 901}},
			{ID: privateID, DeployOverrides: map[string]any{"settlementChainId": 901}},
		},
	}}
	st.SetChainContracts(settlementID, contracts, true)
	st.SetChainContracts(privateID, contracts, true)
	stateFile := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, st.WriteToFile(stateFile))

	logger, logs := testlog.CaptureLogger(t, slog.LevelInfo)
	bundle, err := GetBundleFromFile(logger, stateFile)
	require.NoError(t, err)

	require.Equal(t, contracts.SystemConfigProxy, bundle["opchain_"+settlementID.Hex()+"_"+fieldNameToContractName("SystemConfigProxy")])
	for name := range bundle {
		require.False(t, strings.HasPrefix(name, "opchain_"+privateID.Hex()), "private chain contract %s is verified with its settlement chain", name)
	}
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(slog.LevelInfo),
		testlog.NewMessageContainsFilter("Skipping private chain"),
		testlog.NewAttributesContainsFilter("chain", privateID.Hex()),
	))
}
