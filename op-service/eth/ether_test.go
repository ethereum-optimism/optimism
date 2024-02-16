package eth

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
)

func TestGweiToWei(t *testing.T) {
	maxUint256p1, _ := new(big.Int).Add(abi.MaxUint256, big.NewInt(1)).Float64()
	for _, tt := range []struct {
		desc string
		gwei float64
		wei  *big.Int
		err  bool
	}{
		{
			desc: "zero",
			gwei: 0,
			wei:  new(big.Int),
		},
		{
			desc: "one-wei",
			gwei: 0.000000001,
			wei:  big.NewInt(1),
		},
		{
			desc: "one-gwei",
			gwei: 1.0,
			wei:  big.NewInt(1e9),
		},
		{
			desc: "one-ether",
			gwei: 1e9,
			wei:  big.NewInt(1e18),
		},
		{
			desc: "err-pos-inf",
			gwei: math.Inf(1),
			err:  true,
		},
		{
			desc: "err-neg-inf",
			gwei: math.Inf(-1),
			err:  true,
		},
		{
			desc: "err-nan",
			gwei: math.NaN(),
			err:  true,
		},
		{
			desc: "err-too-large",
			gwei: maxUint256p1,
			err:  true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			wei, err := GweiToWei(tt.gwei)
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wei, wei)
			}
		})
	}
}
