package faultproofs

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/interop"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func StartInteropFaultDisputeSystem(t *testing.T, opts ...faultDisputeConfigOpts) (interop.SuperSystem, *disputegame.FactoryHelper, *ethclient.Client) {
	fdc := new(faultDisputeConfig)
	for _, opt := range opts {
		opt(fdc)
	}
	recipe := interopgen.InteropDevRecipe{
		L1ChainID:        900100,
		L2s:              []interopgen.InteropDevL2Recipe{{ChainID: 900200}, {ChainID: 900201}},
		GenesisTimestamp: uint64(time.Now().Unix() + 3), // start chain 3 seconds from now
	}
	worldResources := interop.WorldResourcePaths{
		FoundryArtifacts: "../../packages/contracts-bedrock/forge-artifacts",
		SourceMap:        "../../packages/contracts-bedrock",
	}
	superCfg := interop.SuperSystemConfig{
		SupportTimeTravel: true,
	}

	hdWallet, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	l1User := devkeys.ChainUserKeys(new(big.Int).SetUint64(recipe.L1ChainID))(0)
	privKey, err := hdWallet.Secret(l1User)
	require.NoError(t, err)
	s2 := interop.NewSuperSystem(t, &recipe, worldResources, superCfg)
	factory := disputegame.NewFactoryHelper(t, context.Background(), disputegame.NewSuperDisputeSystem(s2),
		disputegame.WithFactoryPrivKey(privKey))
	return s2, factory, s2.L1GethClient()
}
