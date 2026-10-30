package faultproofs

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-e2e/interop"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

var InteropL1ChainID = new(big.Int).SetUint64(900100)

func StartInteropFaultDisputeSystem(t *testing.T, opts ...faultDisputeConfigOpts) (interop.SuperSystem, *disputegame.FactoryHelper, *ethclient.Client) {
	fdc := new(faultDisputeConfig)
	for _, opt := range opts {
		opt(fdc)
	}

	sysConfigOpts := new(e2esys.SystemConfigOpts)
	for _, sysOpt := range fdc.sysOpts {
		sysOpt(sysConfigOpts)
	}

	world := interop.WorldConfig{
		L1ChainID:         bigs.Uint64Strict(InteropL1ChainID),
		L2ChainIDs:        []uint64{900200, 900201},
		ContractArtifacts: "../../packages/contracts-bedrock/forge-artifacts",
	}
	superCfg := interop.SuperSystemConfig{
		SupportTimeTravel: true,
		BatcherUsesBlobs:  fdc.batcherUsesBlobs,
	}

	hdWallet, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	privKey, err := hdWallet.Secret(devkeys.ChainUserKeys(InteropL1ChainID)(0))
	require.NoError(t, err)
	s2 := interop.NewSuperSystem(t, world, superCfg)
	factory := disputegame.NewFactoryHelper(t, context.Background(), disputegame.NewSuperDisputeSystem(s2, sysConfigOpts),
		disputegame.WithFactoryPrivKey(privKey))

	ctx := context.Background()
	// wait for the supervisor to sync genesis
	// Also ensures the L1 has advanced past genesis which can otherwise cause gas estimation problems
	genesisTimestamp := s2.L1Genesis().Timestamp
	var lastError error
	err = wait.For(ctx, 1*time.Minute, func() (bool, error) {
		status, err := s2.SupernodeClient().SyncStatus(ctx)
		if err != nil {
			lastError = err
			return false, nil
		}
		return status.SafeTimestamp > genesisTimestamp && status.CurrentL1.Number > 0, nil
	})
	require.NoErrorf(t, err, "failed to wait for supernode to sync genesis: %v", lastError)

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
