package state

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func validBaseChainIntent() *ChainIntent {
	return &ChainIntent{
		ID: common.HexToHash("0x01"),
		Roles: ChainRoles{
			L1ProxyAdminOwner: common.HexToAddress("0x01"),
			L2ProxyAdminOwner: common.HexToAddress("0x02"),
			SystemConfigOwner: common.HexToAddress("0x03"),
			UnsafeBlockSigner: common.HexToAddress("0x04"),
			Batcher:           common.HexToAddress("0x05"),
			Proposer:          common.HexToAddress("0x06"),
			Challenger:        common.HexToAddress("0x07"),
		},
		Eip1559DenominatorCanyon:   5000,
		Eip1559Denominator:         5000,
		Eip1559Elasticity:          5000,
		GasLimit:                   30_000_000,
		BaseFeeVaultRecipient:      common.HexToAddress("0x08"),
		L1FeeVaultRecipient:        common.HexToAddress("0x09"),
		SequencerFeeVaultRecipient: common.HexToAddress("0x0A"),
		OperatorFeeVaultRecipient:  common.HexToAddress("0x0B"),
	}
}

func TestChainIntentCheck_ZKDisputeGame(t *testing.T) {
	verifier := common.HexToAddress("0xabc")
	prestate := common.HexToHash("0xdef")

	tests := []struct {
		name      string
		game      AdditionalDisputeGame
		expectErr error
	}{
		{
			name: "valid ZK game passes",
			game: AdditionalDisputeGame{
				VMType: VMTypeZK,
				ZKDisputeGame: &ZKDisputeGameParams{
					Verifier:             verifier,
					AbsolutePrestate:     prestate,
					MaxChallengeDuration: 3600,
					MaxProveDuration:     7200,
					ChallengerBond:       (*hexutil.Big)(big.NewInt(1e18)),
				},
			},
			expectErr: nil,
		},
		{
			name: "nil ZKDisputeGame params fails",
			game: AdditionalDisputeGame{
				VMType:        VMTypeZK,
				ZKDisputeGame: nil,
			},
			expectErr: ErrZKDisputeGameMissingParams,
		},
		{
			name: "zero Verifier address fails",
			game: AdditionalDisputeGame{
				VMType: VMTypeZK,
				ZKDisputeGame: &ZKDisputeGameParams{
					Verifier:         common.Address{},
					AbsolutePrestate: prestate,
				},
			},
			expectErr: ErrZKDisputeGameMissingParams,
		},
		{
			name: "zero AbsolutePrestate fails",
			game: AdditionalDisputeGame{
				VMType: VMTypeZK,
				ZKDisputeGame: &ZKDisputeGameParams{
					Verifier:         verifier,
					AbsolutePrestate: common.Hash{},
				},
			},
			expectErr: ErrZKDisputeGameMissingParams,
		},
		{
			name: "zero MaxChallengeDuration fails",
			game: AdditionalDisputeGame{
				VMType: VMTypeZK,
				ZKDisputeGame: &ZKDisputeGameParams{
					Verifier:             verifier,
					AbsolutePrestate:     prestate,
					MaxChallengeDuration: 0,
					MaxProveDuration:     7200,
					ChallengerBond:       (*hexutil.Big)(big.NewInt(1e18)),
				},
			},
			expectErr: ErrZKDisputeGameMissingParams,
		},
		{
			name: "zero MaxProveDuration fails",
			game: AdditionalDisputeGame{
				VMType: VMTypeZK,
				ZKDisputeGame: &ZKDisputeGameParams{
					Verifier:             verifier,
					AbsolutePrestate:     prestate,
					MaxChallengeDuration: 3600,
					MaxProveDuration:     0,
					ChallengerBond:       (*hexutil.Big)(big.NewInt(1e18)),
				},
			},
			expectErr: ErrZKDisputeGameMissingParams,
		},
		{
			name: "nil ChallengerBond fails",
			game: AdditionalDisputeGame{
				VMType: VMTypeZK,
				ZKDisputeGame: &ZKDisputeGameParams{
					Verifier:             verifier,
					AbsolutePrestate:     prestate,
					MaxChallengeDuration: 3600,
					MaxProveDuration:     7200,
					ChallengerBond:       nil,
				},
			},
			expectErr: ErrZKDisputeGameMissingParams,
		},
		{
			name: "zero ChallengerBond fails",
			game: AdditionalDisputeGame{
				VMType: VMTypeZK,
				ZKDisputeGame: &ZKDisputeGameParams{
					Verifier:             verifier,
					AbsolutePrestate:     prestate,
					MaxChallengeDuration: 3600,
					MaxProveDuration:     7200,
					ChallengerBond:       (*hexutil.Big)(big.NewInt(0)),
				},
			},
			expectErr: ErrZKDisputeGameMissingParams,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validBaseChainIntent()
			c.AdditionalDisputeGames = []AdditionalDisputeGame{tt.game}
			err := c.Check()
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

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
