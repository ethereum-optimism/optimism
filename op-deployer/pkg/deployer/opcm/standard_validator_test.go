package opcm

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

type standardValidatorBackend struct {
	t                *testing.T
	wantMethod       abi.Method
	wantInput        any
	wantAllowFailure bool
	wantOverrides    validationOverrides
	wantValidator    common.Address
	result           string
	err              error
	// wantWords pins calldata words by index, independently of wantMethod's own ABI definition.
	wantWords map[int]common.Hash
}

func (b *standardValidatorBackend) CallContract(
	_ context.Context,
	call ethereum.CallMsg,
	_ *big.Int,
) ([]byte, error) {
	b.t.Helper()
	require.NotNil(b.t, call.To)
	require.Equal(b.t, b.wantValidator, *call.To)
	require.Equal(b.t, b.wantMethod.ID, call.Data[:4])
	wantArgs, err := b.wantMethod.Inputs.Pack(b.wantInput, b.wantAllowFailure, b.wantOverrides)
	require.NoError(b.t, err)
	require.Equal(b.t, wantArgs, call.Data[4:])
	for i, want := range b.wantWords {
		require.Equal(b.t, want, common.BytesToHash(call.Data[4+32*i:4+32*(i+1)]), "calldata word %d", i)
	}
	if b.err != nil {
		return nil, b.err
	}
	return b.wantMethod.Outputs.Pack(b.result)
}

func TestValidateStandardDeployment(t *testing.T) {
	require.Equal(
		t,
		crypto.Keccak256([]byte("validateWithOverrides((address,bytes32,uint256,address),bool,(address,address))"))[:4],
		standardValidationMethod.ID,
	)
	require.Equal(
		t,
		crypto.Keccak256([]byte("validateWithOverrides((address,bytes32,bytes32,bytes32,uint256,address),bool,(address,address))"))[:4],
		standardValidationDevMethod.ID,
	)

	validator := common.Address{0x01}
	var input StandardValidatorInput
	input.SystemConfig = common.Address{0x02}
	input.AbsolutePrestate = common.Hash{0x03}
	input.L2ChainID = big.NewInt(4)
	input.Proposer = common.Address{0x05}
	input.L1PAOMultisig = common.Address{0x06}
	input.Challenger = common.Address{0x07}
	wantOverrides := validationOverrides{
		L1PAOMultisig: input.L1PAOMultisig,
		Challenger:    input.Challenger,
	}
	wantResult := "OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER"
	wantInput := validationInput{
		SystemConfig:     input.SystemConfig,
		AbsolutePrestate: input.AbsolutePrestate,
		L2ChainID:        input.L2ChainID,
		Proposer:         input.Proposer,
	}

	t.Run("expected override marker passes", func(t *testing.T) {
		backend := &standardValidatorBackend{
			t:             t,
			wantMethod:    standardValidationMethod,
			wantInput:     wantInput,
			wantOverrides: wantOverrides,
			wantValidator: validator,
			result:        wantResult,
		}
		require.NoError(t, ValidateStandardDeployment(t.Context(), backend, validator, input))
	})

	t.Run("unexpected result fails", func(t *testing.T) {
		backend := &standardValidatorBackend{
			t:             t,
			wantMethod:    standardValidationMethod,
			wantInput:     wantInput,
			wantOverrides: wantOverrides,
			wantValidator: validator,
			result:        "PDDG-FAULT",
		}
		err := ValidateStandardDeployment(t.Context(), backend, validator, input)
		require.ErrorContains(t, err, "PDDG-FAULT")
	})

	t.Run("call failure fails", func(t *testing.T) {
		backend := &standardValidatorBackend{
			t:             t,
			wantMethod:    standardValidationMethod,
			wantInput:     wantInput,
			wantOverrides: wantOverrides,
			wantValidator: validator,
			err:           fmt.Errorf("execution reverted"),
		}
		err := ValidateStandardDeployment(t.Context(), backend, validator, input)
		require.ErrorContains(t, err, "execution reverted")
	})

	t.Run("configured bridge hook expects PORTAL-110", func(t *testing.T) {
		hookInput := input
		hookInput.BridgeHookConfigured = true
		backend := &standardValidatorBackend{
			t:                t,
			wantMethod:       standardValidationMethod,
			wantInput:        wantInput,
			wantAllowFailure: true,
			wantOverrides:    wantOverrides,
			wantValidator:    validator,
			result:           wantResult + ",PORTAL-110",
		}
		require.NoError(t, ValidateStandardDeployment(t.Context(), backend, validator, hookInput))

		backend.result = wantResult
		err := ValidateStandardDeployment(t.Context(), backend, validator, hookInput)
		require.ErrorContains(t, err, "standard validator reported errors")
	})

	t.Run("PORTAL-110 without a configured bridge hook fails", func(t *testing.T) {
		backend := &standardValidatorBackend{
			t:             t,
			wantMethod:    standardValidationMethod,
			wantInput:     wantInput,
			wantOverrides: wantOverrides,
			wantValidator: validator,
			result:        wantResult + ",PORTAL-110",
		}
		err := ValidateStandardDeployment(t.Context(), backend, validator, input)
		require.ErrorContains(t, err, "PORTAL-110")
	})

	t.Run("dev input uses overload", func(t *testing.T) {
		input.UseDevFeaturesInput = true
		input.CannonPrestate = common.Hash{0x08}
		input.CannonKonaPrestate = common.Hash{0x09}
		input.ZKPrestate = common.Hash{0x0a}
		backend := &standardValidatorBackend{
			t:          t,
			wantMethod: standardValidationDevMethod,
			wantInput: validationInputDev{
				SystemConfig:       input.SystemConfig,
				CannonPrestate:     input.CannonPrestate,
				CannonKonaPrestate: input.CannonKonaPrestate,
				ZKPrestate:         input.ZKPrestate,
				L2ChainID:          input.L2ChainID,
				Proposer:           input.Proposer,
			},
			wantOverrides: wantOverrides,
			wantValidator: validator,
			result:        wantResult,
			// ValidationInputDev is a static tuple encoded inline, in Solidity's field order:
			// sysCfg, cannonPrestate, cannonKonaPrestate, zkPrestate, l2ChainID, proposer.
			wantWords: map[int]common.Hash{1: input.CannonPrestate, 2: input.CannonKonaPrestate, 3: input.ZKPrestate},
		}
		require.NoError(t, ValidateStandardDeployment(t.Context(), backend, validator, input))
	})
}

func TestExpectedValidatorResult(t *testing.T) {
	set := common.Address{0x01}
	tests := []struct {
		overrides validationOverrides
		hook      bool
		want      string
	}{
		{overrides: validationOverrides{}, want: ""},
		{overrides: validationOverrides{}, hook: true, want: "PORTAL-110"},
		{overrides: validationOverrides{Challenger: set}, hook: true, want: "OVERRIDES-CHALLENGER,PORTAL-110"},
		{overrides: validationOverrides{L1PAOMultisig: set, Challenger: set}, want: "OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, expectedValidatorResult(tt.overrides, tt.hook))
	}
}
