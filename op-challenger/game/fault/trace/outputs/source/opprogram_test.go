package source

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-program/client/driver"
	fpp_config "github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestRunOPProgram(t *testing.T) {
	rollupCfg := chaincfg.Sepolia
	l2ChainConfig, err := params.LoadOPStackChainConfig(rollupCfg.L2ChainID.Uint64())
	require.NoError(t, err)
	logger := testlog.Logger(t, log.LvlInfo)
	l1Rpc := "http://l1:8544"
	l1BeaconApi := "http://beacon:9545"
	dataDir := t.TempDir()
	l1Head := common.Hash{0xab}
	l2Head := eth.BlockID{
		Hash:   common.Hash{0x2a},
		Number: 4892,
	}
	l2OutputRoot := common.Hash{0x2b}
	l2Claim := common.Hash{0x2c}
	l2ClaimBlockNumber := uint64(15886)

	cfg := config.NewConfig(common.Address{}, l1Rpc, l1BeaconApi, dataDir, config.TraceTypeCannon)
	cfg.CannonNetwork = "sepolia"
	cfg.RollupRpc = "http://rollup:1234"
	cfg.CannonL2 = "http://l2:4888"

	runProgram := func(t *testing.T, programErr error) (uint64, bool, *fpp_config.Config, error) {
		var actualCfg *fpp_config.Config
		runner := &fppRunner{
			logger: logger,
			cfg:    &cfg,
			runFPP: func(_ context.Context, _ log.Logger, fppConfig *fpp_config.Config) error {
				actualCfg = fppConfig
				return programErr
			},
		}
		maxSafeHead, valid, err := runner.RunProgram(context.Background(), l1Head, l2Head, l2OutputRoot, l2Claim, l2ClaimBlockNumber)
		return maxSafeHead, valid, actualCfg, err
	}

	t.Run("Config", func(t *testing.T) {
		_, _, actual, err := runProgram(t, nil)
		require.NoError(t, err)
		require.NoError(t, actual.Check())
		require.Equal(t, rollupCfg, actual.Rollup)
		require.Equal(t, filepath.Join(dataDir, subdirName), actual.DataDir)
		require.Equal(t, l1Head, actual.L1Head)
		require.Equal(t, l1Rpc, actual.L1URL)
		require.Equal(t, l1BeaconApi, actual.L1BeaconURL)
		require.Equal(t, false, actual.L1TrustRPC)
		// TODO(client-pod#590): Support setting trust rpc
		// TODO(client-pod#590): Support setting rpc kind
		require.Equal(t, l2Head.Hash, actual.L2Head)
		require.Equal(t, l2OutputRoot, actual.L2OutputRoot)
		require.Equal(t, cfg.CannonL2, actual.L2URL)
		require.Equal(t, l2Claim, actual.L2Claim)
		require.Equal(t, l2ClaimBlockNumber, actual.L2ClaimBlockNumber)
		require.Equal(t, l2ChainConfig, actual.L2ChainConfig)
		require.Equal(t, false, actual.IsCustomChainConfig)
	})

	t.Run("ValidOutputRoot", func(t *testing.T) {
		maxSafeHead, valid, _, err := runProgram(t, nil)
		require.NoError(t, err)
		require.Equal(t, uint64(math.MaxUint64), maxSafeHead, "No need to restrict with valid output root")
		require.True(t, valid)
	})

	t.Run("InvalidOutputRoot", func(t *testing.T) {
		maxSafeHead, valid, _, err := runProgram(t, driver.ErrClaimNotValid)
		require.NoError(t, err)
		// TODO(client-pod#416): Verify the final safe head was returned
		require.Equal(t, uint64(math.MaxUint64), maxSafeHead, "No need to restrict with valid output root")
		require.False(t, valid)
	})

	t.Run("DerivationError", func(t *testing.T) {
		expectedErr := errors.New("boom")
		_, _, _, err := runProgram(t, expectedErr)
		require.ErrorIs(t, err, expectedErr)
	})
}
