package opcm

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

type standardValidatorBackend struct {
	t             *testing.T
	wantMethod    []byte
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
	require.Equal(b.t, b.wantMethod, call.Data[:4])
	method := standardValidationMethod
	if string(b.wantMethod) == string(standardValidationDevMethod.ID) {
		method = standardValidationDevMethod
	}
	values, err := method.Inputs.Unpack(call.Data[4:])
	require.NoError(b.t, err)
	require.Len(b.t, values, 2)
	require.Equal(b.t, true, values[1])
	if b.err != nil {
		return nil, b.err
	}
	return standardValidationMethod.Outputs.Pack(b.result)
}

func TestValidateStandardDeployment(t *testing.T) {
	require.Equal(t, crypto.Keccak256([]byte("validate((address,bytes32,uint256,address),bool)"))[:4], standardValidationMethod.ID)
	require.Equal(t, crypto.Keccak256([]byte("validate((address,bytes32,bytes32,uint256,address),bool)"))[:4], standardValidationDevMethod.ID)

	validator := common.Address{0x01}
	var input StandardValidatorInput
	input.SystemConfig = common.Address{0x02}
	input.AbsolutePrestate = common.Hash{0x03}
	input.L2ChainID = big.NewInt(4)
	input.Proposer = common.Address{0x05}

	t.Run("empty result passes", func(t *testing.T) {
		backend := &standardValidatorBackend{
			t:             t,
			wantMethod:    standardValidationMethod.ID,
			wantValidator: validator,
		}
		require.NoError(t, ValidateStandardDeployment(t.Context(), backend, validator, input))
	})

	t.Run("non-empty result fails", func(t *testing.T) {
		backend := &standardValidatorBackend{
			t:             t,
			wantMethod:    standardValidationMethod.ID,
			wantValidator: validator,
			result:        "PDDG-FAULT",
		}
		err := ValidateStandardDeployment(t.Context(), backend, validator, input)
		require.ErrorContains(t, err, "PDDG-FAULT")
	})

	t.Run("call failure fails", func(t *testing.T) {
		backend := &standardValidatorBackend{
			t:             t,
			wantMethod:    standardValidationMethod.ID,
			wantValidator: validator,
			err:           fmt.Errorf("execution reverted"),
		}
		err := ValidateStandardDeployment(t.Context(), backend, validator, input)
		require.ErrorContains(t, err, "execution reverted")
	})

	t.Run("dev input uses overload", func(t *testing.T) {
		input.UseDevFeaturesInput = true
		input.CannonPrestate = common.Hash{0x06}
		input.CannonKonaPrestate = common.Hash{0x07}
		backend := &standardValidatorBackend{
			t:             t,
			wantMethod:    standardValidationDevMethod.ID,
			wantValidator: validator,
		}
		require.NoError(t, ValidateStandardDeployment(t.Context(), backend, validator, input))
	})
}
