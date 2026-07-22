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
	t             *testing.T
	wantMethod    abi.Method
	wantInput     any
	wantOverrides validationOverrides
	wantValidator common.Address
	result        string
	err           error
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
	wantArgs, err := b.wantMethod.Inputs.Pack(b.wantInput, false, b.wantOverrides)
	require.NoError(b.t, err)
	require.Equal(b.t, wantArgs, call.Data[4:])
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
		crypto.Keccak256([]byte("validateWithOverrides((address,bytes32,bytes32,uint256,address),bool,(address,address))"))[:4],
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

	t.Run("dev input uses overload", func(t *testing.T) {
		input.UseDevFeaturesInput = true
		input.CannonPrestate = common.Hash{0x08}
		input.CannonKonaPrestate = common.Hash{0x09}
		backend := &standardValidatorBackend{
			t:          t,
			wantMethod: standardValidationDevMethod,
			wantInput: validationInputDev{
				SystemConfig:       input.SystemConfig,
				CannonPrestate:     input.CannonPrestate,
				CannonKonaPrestate: input.CannonKonaPrestate,
				L2ChainID:          input.L2ChainID,
				Proposer:           input.Proposer,
			},
			wantOverrides: wantOverrides,
			wantValidator: validator,
			result:        wantResult,
		}
		require.NoError(t, ValidateStandardDeployment(t.Context(), backend, validator, input))
	})
}
