package derive

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
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
		// Activate everything through Jovian at genesis — Jovian must be active before Karst can
		// activate, so this is the realistic configuration — and schedule Karst at karstTime.
		cfg.ActivateAtGenesis(forks.Jovian)
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

	// buildPayload assembles a minimal L2 payload whose first tx is the L1-info
	// deposit, with the given block timestamp and gas limit.
	buildPayload := func(t *testing.T, cfg *rollup.Config, blockTimestamp, gasLimit uint64) *eth.ExecutionPayload {
		rng := rand.New(rand.NewSource(1234))
		l1Info := testutils.RandomBlockInfo(rng)
		l1InfoTx, err := L1InfoDepositBytes(cfg, params.MergedTestChainConfig, sysCfg, 0, l1Info, blockTimestamp)
		require.NoError(t, err)
		return &eth.ExecutionPayload{
			BlockHash:   common.Hash{0x01},
			BlockNumber: hexutil.Uint64(100),
			Timestamp:   hexutil.Uint64(blockTimestamp),
			GasLimit:    hexutil.Uint64(gasLimit),
			// Jovian is active, so extra data carries the EIP-1559 params plus a min base fee
			// (value irrelevant; the test only asserts the gas limit).
			ExtraData:    eip1559.EncodeOptimismExtraData(cfg, blockTimestamp, 250, 6, ptr.New(uint64(1))),
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
		require.False(t, cfg.IsL2CMActivationBlock(nextTime), "next block must not be an activation block")
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

	// A setGasLimit landing in the L1 origin of the block right after the activation block takes
	// precedence over the upgrade-gas strip. Driving it through PreparePayloadAttributes exercises
	// the real ordering — the parent (activation) config is reconstructed and stripped, then, only
	// on the first block of a new L1 origin, that origin's ConfigUpdate events are applied via
	// UpdateSystemConfigWithL1Receipts. setGasLimit sets the gas limit absolutely, so applied after
	// the strip it wins.
	t.Run("setGasLimit in the post-activation block's L1 origin takes precedence over the strip", func(t *testing.T) {
		cfg := mkCfg()

		// The reconstructed parent (activation-block) config has the upgrade gas stripped; this is
		// what SystemConfigByL2Hash returns for the activation block.
		activationPayload := buildPayload(t, cfg, karstTime, baseGasLimit+karstGas)
		parentCfg, err := PayloadToSystemConfig(cfg, activationPayload)
		require.NoError(t, err)
		require.Equal(t, baseGasLimit, parentCfg.GasLimit, "strip recovers the base limit")

		const newGasLimit = uint64(45_000_000)
		require.NotEqual(t, baseGasLimit, newGasLimit)
		require.NotEqual(t, baseGasLimit+karstGas, newGasLimit)

		rng := rand.New(rand.NewSource(1234))
		l2Parent := testutils.RandomL2BlockRef(rng)
		l2Parent.Time = karstTime // parent is the Karst activation block

		l1Fetcher := &testutils.MockL1Source{}
		defer l1Fetcher.AssertExpectations(t)
		l1CfgFetcher := &testutils.MockL2Client{}
		l1CfgFetcher.ExpectSystemConfigByL2Hash(l2Parent.Hash, parentCfg, nil)
		defer l1CfgFetcher.AssertExpectations(t)

		// The block being built is the first of a new L1 origin, whose receipts carry the setGasLimit.
		l1Info := testutils.RandomBlockInfo(rng)
		l1Info.InfoParentHash = l2Parent.L1Origin.Hash
		l1Info.InfoNum = l2Parent.L1Origin.Number + 1
		l1Info.InfoTime = karstTime // <= nextL2Time, satisfies the L1-origin time invariant
		epoch := l1Info.ID()

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
		l1Fetcher.ExpectFetchReceipts(epoch.Hash, l1Info, optypes.FromGethReceipts(receipts), nil)

		attrBuilder := NewFetchingAttributesBuilder(testlog.Logger(t, log.LevelInfo), cfg, params.MergedTestChainConfig, nil, l1Fetcher, l1CfgFetcher)
		attrs, err := attrBuilder.PreparePayloadAttributes(context.Background(), l2Parent, epoch)
		require.NoError(t, err)
		require.NotNil(t, attrs.GasLimit)
		require.Equal(t, newGasLimit, uint64(*attrs.GasLimit),
			"setGasLimit in the post-activation block's L1 origin must override the upgrade-gas strip")
	})
}

// TestUpgradeGasToStrip covers the generalization to every NUT-bundle fork from Karst onward: each
// fork strips its own upgrade gas at its activation block, only Karst has the opt-out. It loops over
// forks.From(Karst), so forks not yet named are covered automatically.
func TestUpgradeGasToStrip(t *testing.T) {
	const blockTime = uint64(2)
	cfg := &rollup.Config{BlockTime: blockTime}
	cfg.ActivateAtGenesis(forks.Jovian) // everything through Jovian active at genesis

	fromKarst := forks.From(forks.Karst)
	// Schedule every NUT-bundle fork from Karst onward at distinct, increasing times.
	activation := func(i int) uint64 { return uint64(1000 * (i + 1)) }
	for i, fork := range fromKarst {
		ts := activation(i)
		cfg.SetActivationTime(fork, &ts)
	}

	// A fork's upgrade gas, or 0 for a fork without a NUT bundle (UpgradeGas errors).
	upgradeGas := func(fork forks.Name) uint64 {
		if gas, err := UpgradeGas(fork); err == nil {
			return gas
		}
		return 0
	}

	for i, fork := range fromKarst {
		ts := activation(i)
		require.Equalf(t, upgradeGas(fork), upgradeGasToStrip(cfg, ts),
			"%s strips its own upgrade gas at its activation block", fork)
		require.Zerof(t, upgradeGasToStrip(cfg, ts-blockTime), "%s: block before activation strips nothing", fork)
		require.Zerof(t, upgradeGasToStrip(cfg, ts+blockTime), "%s: block after activation strips nothing", fork)
	}

	// KeepKarstUpgradeGas opts out of Karst only; later forks have no opt-out.
	cfg.KeepKarstUpgradeGas = true
	for i, fork := range fromKarst {
		ts := activation(i)
		want := upgradeGas(fork)
		if fork == forks.Karst {
			want = 0
		}
		require.Equalf(t, want, upgradeGasToStrip(cfg, ts),
			"with KeepKarstUpgradeGas set, only Karst opts out (checking %s)", fork)
	}
}

// TestPayloadToBlockRef covers the L1-info extraction from a payload's opaque
// first transaction. Payload transactions carry raw encodings, so the L1-info
// deposit must decode without go-ethereum's typed-transaction support.
func TestPayloadToBlockRef(t *testing.T) {
	cfg := &rollup.Config{
		BlockTime:              2,
		L1ChainID:              big.NewInt(101),
		L2ChainID:              big.NewInt(102),
		DepositContractAddress: common.Address{0xbb},
		L1SystemConfigAddress:  common.Address{0xcc},
		Genesis: rollup.Genesis{
			L1: eth.BlockID{Hash: common.Hash{0xaa}, Number: 9},
			L2: eth.BlockID{Hash: common.Hash{0xbb}, Number: 90},
		},
	}
	cfg.ActivateAtGenesis(forks.Jovian)
	const blockTimestamp = uint64(1000)
	const seqNumber = uint64(7)

	rng := rand.New(rand.NewSource(42))
	l1Info := testutils.RandomBlockInfo(rng)
	sysCfg := eth.SystemConfig{BatcherAddr: common.Address{0x42}}
	l1InfoTx, err := L1InfoDepositBytes(cfg, params.MergedTestChainConfig, sysCfg, seqNumber, l1Info, blockTimestamp)
	require.NoError(t, err)

	payload := &eth.ExecutionPayload{
		BlockHash:    common.Hash{0x01},
		ParentHash:   common.Hash{0x02},
		BlockNumber:  hexutil.Uint64(91),
		Timestamp:    hexutil.Uint64(blockTimestamp),
		Transactions: []eth.Data{l1InfoTx},
	}

	t.Run("deposit payload", func(t *testing.T) {
		ref, err := PayloadToBlockRef(cfg, payload)
		require.NoError(t, err)
		require.Equal(t, eth.L2BlockRef{
			Hash:           payload.BlockHash,
			Number:         uint64(payload.BlockNumber),
			ParentHash:     payload.ParentHash,
			Time:           blockTimestamp,
			L1Origin:       eth.BlockID{Hash: l1Info.Hash(), Number: l1Info.NumberU64()},
			SequenceNumber: seqNumber,
		}, ref)
	})

	t.Run("genesis payload", func(t *testing.T) {
		genesisPayload := &eth.ExecutionPayload{
			BlockHash:   cfg.Genesis.L2.Hash,
			BlockNumber: hexutil.Uint64(cfg.Genesis.L2.Number),
		}
		ref, err := PayloadToBlockRef(cfg, genesisPayload)
		require.NoError(t, err)
		require.Equal(t, cfg.Genesis.L1, ref.L1Origin)
		require.Zero(t, ref.SequenceNumber)

		genesisPayload.BlockHash = common.Hash{0xde, 0xad}
		_, err = PayloadToBlockRef(cfg, genesisPayload)
		require.ErrorContains(t, err, "expected L2 genesis hash")
	})

	t.Run("errors", func(t *testing.T) {
		mut := func(mut func(p *eth.ExecutionPayload)) *eth.ExecutionPayload {
			p := *payload
			mut(&p)
			return &p
		}
		for _, tc := range []struct {
			name    string
			payload *eth.ExecutionPayload
			errStr  string
		}{
			{"no txs", mut(func(p *eth.ExecutionPayload) { p.Transactions = nil }), "missing L1 info deposit"},
			{"empty first tx", mut(func(p *eth.ExecutionPayload) { p.Transactions = []eth.Data{{}} }), "failed to decode L1 info deposit"},
			{"non-deposit first tx", mut(func(p *eth.ExecutionPayload) { p.Transactions = []eth.Data{{0x02, 0xc0}} }), "failed to decode L1 info deposit"},
			{"truncated deposit", mut(func(p *eth.ExecutionPayload) { p.Transactions = []eth.Data{{optypes.DepositTxType}} }), "failed to decode L1 info deposit"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := PayloadToBlockRef(cfg, tc.payload)
				require.ErrorContains(t, err, tc.errStr)
			})
		}
	})
}
