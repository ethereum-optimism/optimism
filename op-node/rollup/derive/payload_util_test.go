package derive

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestPayloadToSystemConfigExtraDataGasLimit checks that PayloadToSystemConfig applies the
// Holocene extraData consensus rule with the block's own gas limit (op-geth's
// ValidateOptimismExtraData rejects an elasticity larger than the gas limit), so that op-node
// and op-geth agree on which L2 blocks are valid.
func TestPayloadToSystemConfigExtraDataGasLimit(t *testing.T) {
	zero := uint64(0)
	canyonDenominator := uint64(250)
	cfg := &rollup.Config{
		Genesis:      rollup.Genesis{L2: eth.BlockID{Hash: common.Hash{0xaa}, Number: 0}},
		RegolithTime: &zero,
		CanyonTime:   &zero,
		DeltaTime:    &zero,
		EcotoneTime:  &zero,
		FjordTime:    &zero,
		GraniteTime:  &zero,
		HoloceneTime: &zero,
		ChainOpConfig: &params.OptimismConfig{
			EIP1559Elasticity:        6,
			EIP1559Denominator:       50,
			EIP1559DenominatorCanyon: &canyonDenominator,
		},
	}

	rng := rand.New(rand.NewSource(7))
	l1Info := testutils.RandomBlockInfo(rng)
	sysCfg := randomL1Cfg(rng, l1Info)
	l1InfoTx, err := L1InfoDepositBytes(cfg, params.MergedTestChainConfig, sysCfg, 3, l1Info, 1000)
	require.NoError(t, err)

	for _, tc := range []struct {
		gasLimit uint64
		err      string
	}{
		{gasLimit: 30_000_000},
		{gasLimit: 6},
		{gasLimit: 5, err: "holocene extraData elasticity 6 exceeds gas limit 5"},
		{gasLimit: 0, err: "holocene extraData elasticity 6 exceeds gas limit 0"},
	} {
		payload := &eth.ExecutionPayload{
			BlockHash:    common.Hash{0xbb},
			BlockNumber:  100,
			Timestamp:    1000,
			GasLimit:     eth.Uint64Quantity(tc.gasLimit),
			ExtraData:    eth.BytesMax32(eip1559.EncodeHoloceneExtraData(canyonDenominator, 6)),
			Transactions: []hexutil.Bytes{l1InfoTx},
		}
		got, err := PayloadToSystemConfig(cfg, payload)
		if tc.err != "" {
			require.ErrorContains(t, err, tc.err, "gasLimit=%d", tc.gasLimit)
			continue
		}
		require.NoError(t, err, "gasLimit=%d", tc.gasLimit)
		require.Equal(t, tc.gasLimit, got.GasLimit)
		require.Equal(t, sysCfg.BatcherAddr, got.BatcherAddr)
	}
}
