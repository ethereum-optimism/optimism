package state

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestGetInitialLiquidity(t *testing.T) {
	tests := []struct {
		name     string
		cgt      CustomGasToken
		expected *big.Int
	}{
		{
			name: "returns type(uint248).max when CustomGasToken is enabled and InitialLiquidity is not set",
			cgt: CustomGasToken{
				Name:             "Custom Gas Token",
				Symbol:           "CGT",
				InitialLiquidity: nil,
			},
			expected: func() *big.Int {
				maxUint248 := new(big.Int)
				maxUint248.SetString("00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
				return maxUint248
			}(),
		},
		{
			name: "returns custom value when InitialLiquidity is explicitly set",
			cgt: CustomGasToken{
				Name:             "Custom Gas Token",
				Symbol:           "CGT",
				InitialLiquidity: (*hexutil.Big)(big.NewInt(1000)),
			},
			expected: big.NewInt(1000),
		},
		{
			name: "returns zero when CustomGasToken is not enabled",
			cgt: CustomGasToken{
				Name:             "",
				Symbol:           "",
				InitialLiquidity: nil,
			},
			expected: big.NewInt(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainIntent := &ChainIntent{
				CustomGasToken: tt.cgt,
			}
			result := chainIntent.GetInitialLiquidity()
			require.Equal(t, tt.expected, result, "GetInitialLiquidity() should return the expected value")
		})
	}
}

func TestGetLiquidityControllerOwner(t *testing.T) {
	defaultOwner := common.HexToAddress("0x1234")
	customOwner := common.HexToAddress("0x5678")

	tests := []struct {
		name     string
		cgt      CustomGasToken
		roles    ChainRoles
		expected common.Address
	}{
		{
			name: "returns L2ProxyAdminOwner when CustomGasToken.LiquidityControllerOwner is not set",
			cgt: CustomGasToken{
				Name:   "Custom Gas Token",
				Symbol: "CGT",
			},
			roles: ChainRoles{
				L2ProxyAdminOwner: defaultOwner,
			},
			expected: defaultOwner,
		},
		{
			name: "returns custom owner when CustomGasToken.LiquidityControllerOwner is set",
			cgt: CustomGasToken{
				Name:                     "Custom Gas Token",
				Symbol:                   "CGT",
				LiquidityControllerOwner: customOwner,
			},
			roles: ChainRoles{
				L2ProxyAdminOwner: defaultOwner,
			},
			expected: customOwner,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainIntent := &ChainIntent{
				CustomGasToken: tt.cgt,
				Roles:          tt.roles,
			}
			result := chainIntent.GetLiquidityControllerOwner()
			require.Equal(t, tt.expected, result, "GetLiquidityControllerOwner() should return the expected address")
		})
	}
}

func TestIsCustomGasTokenEnabled(t *testing.T) {
	tests := []struct {
		name     string
		cgt      CustomGasToken
		expected bool
	}{
		{
			name: "returns true when both Name and Symbol are set",
			cgt: CustomGasToken{
				Name:   "Custom Gas Token",
				Symbol: "CGT",
			},
			expected: true,
		},
		{
			name: "returns false when Name is empty",
			cgt: CustomGasToken{
				Name:   "",
				Symbol: "CGT",
			},
			expected: false,
		},
		{
			name: "returns false when Symbol is empty",
			cgt: CustomGasToken{
				Name:   "Custom Gas Token",
				Symbol: "",
			},
			expected: false,
		},
		{
			name: "returns false when both Name and Symbol are empty",
			cgt: CustomGasToken{
				Name:   "",
				Symbol: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainIntent := &ChainIntent{
				CustomGasToken: tt.cgt,
			}
			result := chainIntent.IsCustomGasTokenEnabled()
			require.Equal(t, tt.expected, result, "IsCustomGasTokenEnabled() should return the expected value")
		})
	}
}
