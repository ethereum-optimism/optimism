package derive

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// buildPayloadAndDeposit produces an L2 payload whose first transaction is a
// valid L1 info deposit, plus the decoded *types.Transaction for the deposit.
func buildPayloadAndDeposit(t *testing.T, rng *rand.Rand, rollupCfg *rollup.Config, sysCfg eth.SystemConfig, seqNr, l2Num, l2Time uint64) (*eth.ExecutionPayload, *types.Transaction) {
	t.Helper()
	l1Info := testutils.MakeBlockInfo(nil)(rng)
	depTx, err := L1InfoDeposit(rollupCfg, params.MergedTestChainConfig, sysCfg, seqNr, l1Info, l2Time)
	require.NoError(t, err)
	depBin, err := types.NewTx(depTx).MarshalBinary()
	require.NoError(t, err)

	payload := &eth.ExecutionPayload{
		ParentHash:    testutils.RandomHash(rng),
		FeeRecipient:  testutils.RandomAddress(rng),
		StateRoot:     eth.Bytes32(testutils.RandomHash(rng)),
		ReceiptsRoot:  eth.Bytes32(testutils.RandomHash(rng)),
		LogsBloom:     eth.Bytes256{},
		PrevRandao:    eth.Bytes32(testutils.RandomHash(rng)),
		BlockNumber:   eth.Uint64Quantity(l2Num),
		GasLimit:      eth.Uint64Quantity(sysCfg.GasLimit),
		GasUsed:       21000,
		Timestamp:     eth.Uint64Quantity(l2Time),
		ExtraData:     eth.BytesMax32(make([]byte, 0)),
		BaseFeePerGas: eth.Uint256Quantity(*uint256.NewInt(1)),
		BlockHash:     testutils.RandomHash(rng),
		Transactions:  []eth.Data{depBin},
	}
	return payload, types.NewTx(depTx)
}

// TestPayloadToBlockRef_ParityWithInnerHelper verifies that the legacy
// PayloadToBlockRef wrapper and the new BlockRefFromHeaderAndDeposit inner
// helper produce identical L2BlockRefs across multiple block configurations.
func TestPayloadToBlockRef_ParityWithInnerHelper(t *testing.T) {
	rollupCfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2: eth.BlockID{Number: 0, Hash: common.Hash{0xab}},
			L1: eth.BlockID{Number: 1000, Hash: common.Hash{0xcd}},
		},
	}
	sysCfg := eth.SystemConfig{
		BatcherAddr: testutils.RandomAddress(rand.New(rand.NewSource(1))),
		GasLimit:    30_000_000,
	}

	cases := []struct {
		name string
		num  uint64
	}{
		{"post-genesis", 100},
		{"deeper", 1_000_000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(int64(c.num)))
			payload, deposit := buildPayloadAndDeposit(t, rng, rollupCfg, sysCfg, 7, c.num, 1_700_000_000+c.num)

			refViaPayload, err := PayloadToBlockRef(rollupCfg, payload)
			require.NoError(t, err)

			refViaInner, err := BlockRefFromHeaderAndDeposit(rollupCfg, eth.PayloadToInfo(payload), deposit)
			require.NoError(t, err)

			require.Equal(t, refViaPayload, refViaInner)
		})
	}
}

// TestPayloadToSystemConfig_ParityWithInnerHelper verifies that
// PayloadToSystemConfig and SystemConfigFromHeaderAndDeposit produce
// identical SystemConfigs for the same payload across multiple forks.
func TestPayloadToSystemConfig_ParityWithInnerHelper(t *testing.T) {
	cases := []struct {
		name      string
		rollupCfg *rollup.Config
		l2Time    uint64
	}{
		{
			name: "bedrock",
			rollupCfg: &rollup.Config{
				Genesis: rollup.Genesis{
					L2:           eth.BlockID{Number: 0, Hash: common.Hash{0xab}},
					L1:           eth.BlockID{Number: 1000, Hash: common.Hash{0xcd}},
					SystemConfig: eth.SystemConfig{GasLimit: 30_000_000},
				},
			},
			l2Time: 1_700_000_000,
		},
	}
	sysCfg := eth.SystemConfig{
		BatcherAddr: testutils.RandomAddress(rand.New(rand.NewSource(2))),
		GasLimit:    30_000_000,
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(int64(c.l2Time)))
			payload, deposit := buildPayloadAndDeposit(t, rng, c.rollupCfg, sysCfg, 3, 100, c.l2Time)

			cfgViaPayload, err := PayloadToSystemConfig(c.rollupCfg, payload)
			require.NoError(t, err)

			cfgViaInner, err := SystemConfigFromHeaderAndDeposit(c.rollupCfg, eth.PayloadToInfo(payload), deposit)
			require.NoError(t, err)

			require.Equal(t, cfgViaPayload, cfgViaInner)
		})
	}
}

// TestBlockRefFromHeaderAndDeposit_Genesis verifies that at genesis, the
// L1Origin comes from the rollup config and the deposit is not consulted
// (passing nil deposit is allowed).
func TestBlockRefFromHeaderAndDeposit_Genesis(t *testing.T) {
	genesisHash := common.Hash{0xab}
	rollupCfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2: eth.BlockID{Number: 0, Hash: genesisHash},
			L1: eth.BlockID{Number: 1000, Hash: common.Hash{0xcd}},
		},
	}
	hdr := &types.Header{
		ParentHash: common.Hash{},
		Number:     big.NewInt(0),
		Time:       100,
	}
	info := eth.HeaderBlockInfoTrusted(genesisHash, hdr)

	ref, err := BlockRefFromHeaderAndDeposit(rollupCfg, info, nil)
	require.NoError(t, err)
	require.Equal(t, rollupCfg.Genesis.L1, ref.L1Origin)
	require.Equal(t, uint64(0), ref.SequenceNumber)
	require.Equal(t, genesisHash, ref.Hash)
}

// TestBlockRefFromHeaderAndDeposit_GenesisHashMismatch verifies that a
// genesis-numbered block with the wrong hash is rejected.
func TestBlockRefFromHeaderAndDeposit_GenesisHashMismatch(t *testing.T) {
	rollupCfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2: eth.BlockID{Number: 0, Hash: common.Hash{0xab}},
		},
	}
	hdr := &types.Header{Number: big.NewInt(0)}
	info := eth.HeaderBlockInfoTrusted(common.Hash{0x99}, hdr)

	_, err := BlockRefFromHeaderAndDeposit(rollupCfg, info, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected L2 genesis hash")
}

// TestBlockRefFromHeaderAndDeposit_FirstTxNotDeposit rejects a non-deposit
// first transaction.
func TestBlockRefFromHeaderAndDeposit_FirstTxNotDeposit(t *testing.T) {
	rollupCfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2: eth.BlockID{Number: 0, Hash: common.Hash{0xab}},
		},
	}
	hdr := &types.Header{Number: big.NewInt(100)}
	info := eth.HeaderBlockInfoTrusted(common.Hash{0x77}, hdr)
	nonDeposit := types.NewTx(&types.LegacyTx{Nonce: 0, Gas: 21000, GasPrice: big.NewInt(1)})

	_, err := BlockRefFromHeaderAndDeposit(rollupCfg, info, nonDeposit)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected tx type")
}

// TestBlockRefFromHeaderAndDeposit_MissingDeposit rejects a non-genesis
// block with a nil deposit.
func TestBlockRefFromHeaderAndDeposit_MissingDeposit(t *testing.T) {
	rollupCfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2: eth.BlockID{Number: 0, Hash: common.Hash{0xab}},
		},
	}
	hdr := &types.Header{Number: big.NewInt(100)}
	info := eth.HeaderBlockInfoTrusted(common.Hash{0x77}, hdr)

	_, err := BlockRefFromHeaderAndDeposit(rollupCfg, info, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing L1 info deposit")
}
