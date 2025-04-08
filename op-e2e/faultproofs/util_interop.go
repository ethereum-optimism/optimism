package faultproofs

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-e2e/interop"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

var InteropL1ChainID = new(big.Int).SetUint64(900100)

func StartInteropFaultDisputeSystem(t *testing.T, opts ...faultDisputeConfigOpts) (interop.SuperSystem, *disputegame.FactoryHelper, *ethclient.Client) {
	fdc := new(faultDisputeConfig)
	for _, opt := range opts {
		opt(fdc)
	}
	recipe := interopgen.InteropDevRecipe{
		L1ChainID:        InteropL1ChainID.Uint64(),
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

	ctx := context.Background()
	// wait for the supervisor to sync genesis
	var lastError error
	err = wait.For(ctx, 1*time.Minute, func() (bool, error) {
		status, err := s2.SupervisorClient().SyncStatus(ctx)
		if err != nil {
			lastError = err
			return false, nil
		}
		return status.SafeTimestamp != 0, nil
	})
	require.NoErrorf(t, err, "failed to wait for supervisor to sync genesis: %v", lastError)

	return s2, factory, s2.L1GethClient()
}

func aliceKey(t *testing.T) *ecdsa.PrivateKey {
	hdWallet, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	challengerKey, err := hdWallet.Secret(devkeys.ChainUserKeys(InteropL1ChainID)(1))
	require.NoError(t, err)
	return challengerKey
}

func malloryKey(t *testing.T) *ecdsa.PrivateKey {
	hdWallet, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	malloryKey, err := hdWallet.Secret(devkeys.ChainUserKeys(InteropL1ChainID)(2))
	require.NoError(t, err)
	return malloryKey
}
