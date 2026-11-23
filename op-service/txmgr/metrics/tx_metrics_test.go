package metrics

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

func gaugeValue(t *testing.T, g prometheus.Gauge) float64 {
	var m dto.Metric
	require.NoError(t, g.Write(&m))
	return m.GetGauge().GetValue()
}

func TestTxConfirmedFee(t *testing.T) {
	for name, tc := range map[string]struct {
		price   *big.Int
		gasUsed uint64
		fee     float64
	}{
		"fee in gwei":                   {price: big.NewInt(2 * params.GWei), gasUsed: 21_000, fee: 42_000},
		"wei product above u64":         {price: big.NewInt(1000 * params.GWei), gasUsed: 30_000_000, fee: 3e10},
		"effective gas price above u64": {price: new(big.Int).Lsh(big.NewInt(1), 64), gasUsed: 1, fee: 0x1p64 / params.GWei},
	} {
		t.Run(name, func(t *testing.T) {
			m := MakeTxMetrics("test", metrics.With(prometheus.NewRegistry()))
			m.TxConfirmed(&types.Receipt{EffectiveGasPrice: tc.price, GasUsed: tc.gasUsed})
			require.InEpsilon(t, tc.fee, gaugeValue(t, m.txL1GasFee), 1e-9)
		})
	}

	t.Run("no effective gas price", func(t *testing.T) {
		m := MakeTxMetrics("test", metrics.With(prometheus.NewRegistry()))
		m.TxConfirmed(&types.Receipt{GasUsed: 21_000, Type: types.BlobTxType})
		require.Zero(t, gaugeValue(t, m.txL1GasFee))
		require.Equal(t, float64(types.BlobTxType), gaugeValue(t, m.txType))
	})
}
