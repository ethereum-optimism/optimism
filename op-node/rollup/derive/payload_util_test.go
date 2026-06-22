package derive

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestPayloadToSystemConfigUpgradeGas covers the reconstruction of the system
// config gas limit from an L2 payload across a Karst (L2CM) activation.
//
// The Karst activation block's gas limit carries the one-time NUT-bundle upgrade
// gas on top of the system config gas limit. That upgrade gas applies only to the
// activation block, so PayloadToSystemConfig must strip it back out — otherwise
// the inflated limit would be read back as the system config gas limit and
// persist on every block after the activation block.
func TestPayloadToSystemConfigUpgradeGas(t *testing.T) {
	const (
		baseGasLimit = uint64(30_000_000)
		blockTime    = uint64(2)
		karstTime    = uint64(1000)
	)

	karstGas, err := UpgradeGas(forks.Karst)
	require.NoError(t, err)
	require.NotZero(t, karstGas, "Karst NUT bundle must reserve upgrade gas")

	mkCfg := func() *rollup.Config {
		cfg := &rollup.Config{
			BlockTime:              blockTime,
			L1ChainID:              big.NewInt(101),
			L2ChainID:              big.NewInt(102),
			DepositContractAddress: common.Address{0xbb},
			L1SystemConfigAddress:  common.Address{0xcc},
		}
		// Activate everything through Isthmus at genesis (so Holocene is active,
		// Jovian is not) and schedule Karst at karstTime. This isolates the gas
		// stripping while keeping the L1-info tx and extra-data formats simple.
		cfg.ActivateAtGenesis(forks.Isthmus)
		kt := karstTime
		cfg.KarstTime = &kt
		return cfg
	}

	sysCfg := eth.SystemConfig{
		BatcherAddr: common.Address{0x42},
		Overhead:    [32]byte{},
		Scalar:      [32]byte{},
		GasLimit:    baseGasLimit,
	}

	// holoceneExtraData is valid Holocene header extra data (non-zero denominator
	// and elasticity), required because Holocene is active.
	holoceneExtraData := eip1559.EncodeHoloceneExtraData(250, 6)

	// buildPayload assembles a minimal L2 payload whose first tx is the L1-info
	// deposit, with the given block timestamp and gas limit.
	buildPayload := func(t *testing.T, cfg *rollup.Config, blockTimestamp, gasLimit uint64) *eth.ExecutionPayload {
		rng := rand.New(rand.NewSource(1234))
		l1Info := testutils.RandomBlockInfo(rng)
		l1InfoTx, err := L1InfoDepositBytes(cfg, params.MergedTestChainConfig, sysCfg, 0, l1Info, blockTimestamp)
		require.NoError(t, err)
		return &eth.ExecutionPayload{
			BlockHash:    common.Hash{0x01},
			BlockNumber:  hexutil.Uint64(100),
			Timestamp:    hexutil.Uint64(blockTimestamp),
			GasLimit:     hexutil.Uint64(gasLimit),
			ExtraData:    holoceneExtraData,
			Transactions: []eth.Data{l1InfoTx},
		}
	}

	t.Run("strips upgrade gas at karst activation block", func(t *testing.T) {
		cfg := mkCfg()
		require.True(t, cfg.IsL2CMActivationBlock(karstTime), "karstTime must be the activation block")
		// The activation block's gas limit is base + the one-time upgrade gas.
		payload := buildPayload(t, cfg, karstTime, baseGasLimit+karstGas)
		got, err := PayloadToSystemConfig(cfg, payload)
		require.NoError(t, err)
		require.Equal(t, baseGasLimit, got.GasLimit,
			"upgrade gas must be stripped so the system config holds the steady-state gas limit")
	})

	t.Run("leaves gas limit untouched after activation block", func(t *testing.T) {
		cfg := mkCfg()
		nextTime := karstTime + blockTime
		require.False(t, cfg.IsL2CMActivationBlock(nextTime), "block after activation must not strip gas")
		payload := buildPayload(t, cfg, nextTime, baseGasLimit)
		got, err := PayloadToSystemConfig(cfg, payload)
		require.NoError(t, err)
		require.Equal(t, baseGasLimit, got.GasLimit)
	})

	t.Run("keeps inflated gas when KeepKarstUpgradeGas is set", func(t *testing.T) {
		cfg := mkCfg()
		cfg.KeepKarstUpgradeGas = true
		require.Zero(t, upgradeGasToStrip(cfg, karstTime), "opted out: activation block not stripped")
		payload := buildPayload(t, cfg, karstTime, baseGasLimit+karstGas)
		got, err := PayloadToSystemConfig(cfg, payload)
		require.NoError(t, err)
		require.Equal(t, baseGasLimit+karstGas, got.GasLimit,
			"upgrade gas must be kept when KeepKarstUpgradeGas is set")
	})

	// A setGasLimit landing in the L1 origin of the block right after the activation block
	// takes precedence over the upgrade-gas strip. PreparePayloadAttributes reconstructs the
	// parent (activation) config — the strip — and then, only on the first block of a new L1
	// origin, applies that origin's ConfigUpdate events via UpdateSystemConfigWithL1Receipts.
	// setGasLimit sets the gas limit absolutely, so applied after the strip it wins. This test
	// runs that exact sequence with the real functions.
	t.Run("setGasLimit in the post-activation block's L1 origin takes precedence over the strip", func(t *testing.T) {
		cfg := mkCfg()
		payload := buildPayload(t, cfg, karstTime, baseGasLimit+karstGas)
		reconstructed, err := PayloadToSystemConfig(cfg, payload)
		require.NoError(t, err)
		require.Equal(t, baseGasLimit, reconstructed.GasLimit, "strip recovers the base limit")

		const newGasLimit = uint64(45_000_000)
		require.NotEqual(t, baseGasLimit, newGasLimit)
		require.NotEqual(t, baseGasLimit+karstGas, newGasLimit)

		// Build the SystemConfig setGasLimit ConfigUpdate log for the new L1 origin.
		numberData, err := oneUint256.Pack(new(big.Int).SetUint64(newGasLimit))
		require.NoError(t, err)
		logData, err := bytesArgs.Pack(numberData)
		require.NoError(t, err)
		receipts := []*types.Receipt{{
			Status: types.ReceiptStatusSuccessful,
			Logs: []*types.Log{{
				Address: cfg.L1SystemConfigAddress,
				Topics: []common.Hash{
					ConfigUpdateEventABIHash, ConfigUpdateEventVersion0, SystemConfigUpdateGasLimit,
				},
				Data: logData,
			}},
		}}
		require.NoError(t, UpdateSystemConfigWithL1Receipts(&reconstructed, receipts, cfg, karstTime))
		require.Equal(t, newGasLimit, reconstructed.GasLimit,
			"setGasLimit in the post-activation block's L1 origin must override the upgrade-gas strip")
	})
}

// TestUpgradeGasToStrip covers the generalization to every NUT-bundle fork from Karst onward:
// each fork strips its own upgrade gas at its activation block, only Karst has the opt-out.
func TestUpgradeGasToStrip(t *testing.T) {
	karstGas, err := UpgradeGas(forks.Karst)
	require.NoError(t, err)
	lagoonGas, err := UpgradeGas(forks.Lagoon)
	require.NoError(t, err)

	const blockTime = uint64(2)
	karstTime := uint64(1000)
	lagoonTime := uint64(2000)
	cfg := &rollup.Config{BlockTime: blockTime}
	cfg.ActivateAtGenesis(forks.Jovian) // everything through Jovian active at genesis
	cfg.KarstTime = &karstTime
	cfg.LagoonTime = &lagoonTime

	// Each NUT-bundle fork strips its own upgrade gas at its activation block, and nowhere else.
	require.Equal(t, karstGas, upgradeGasToStrip(cfg, karstTime))
	require.Zero(t, upgradeGasToStrip(cfg, karstTime-blockTime))
	require.Zero(t, upgradeGasToStrip(cfg, karstTime+blockTime))
	require.Equal(t, lagoonGas, upgradeGasToStrip(cfg, lagoonTime))

	// KeepKarstUpgradeGas opts out of Karst only; later forks have no opt-out.
	cfg.KeepKarstUpgradeGas = true
	require.Zero(t, upgradeGasToStrip(cfg, karstTime), "Karst opted out")
	require.Equal(t, lagoonGas, upgradeGasToStrip(cfg, lagoonTime), "Lagoon still strips")
}
