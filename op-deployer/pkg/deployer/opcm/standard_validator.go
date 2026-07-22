package opcm

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type CallContractBackend interface {
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

type StandardValidatorInput struct {
	SystemConfig        common.Address
	AbsolutePrestate    common.Hash
	CannonPrestate      common.Hash
	CannonKonaPrestate  common.Hash
	L2ChainID           *big.Int
	Proposer            common.Address
	L1PAOMultisig       common.Address
	Challenger          common.Address
	UseDevFeaturesInput bool
}

type validationInput struct {
	SystemConfig     common.Address `abi:"sysCfg"`
	AbsolutePrestate common.Hash    `abi:"absolutePrestate"`
	L2ChainID        *big.Int       `abi:"l2ChainID"`
	Proposer         common.Address `abi:"proposer"`
}

type validationInputDev struct {
	SystemConfig       common.Address `abi:"sysCfg"`
	CannonPrestate     common.Hash    `abi:"cannonPrestate"`
	CannonKonaPrestate common.Hash    `abi:"cannonKonaPrestate"`
	L2ChainID          *big.Int       `abi:"l2ChainID"`
	Proposer           common.Address `abi:"proposer"`
}

type validationOverrides struct {
	L1PAOMultisig common.Address `abi:"l1PAOMultisig"`
	Challenger    common.Address `abi:"challenger"`
}

var (
	standardValidationMethod    = newStandardValidationMethod(false)
	standardValidationDevMethod = newStandardValidationMethod(true)
)

func newStandardValidationMethod(dev bool) abi.Method {
	components := []abi.ArgumentMarshaling{
		{Name: "sysCfg", Type: "address"},
	}
	if dev {
		components = append(components,
			abi.ArgumentMarshaling{Name: "cannonPrestate", Type: "bytes32"},
			abi.ArgumentMarshaling{Name: "cannonKonaPrestate", Type: "bytes32"},
		)
	} else {
		components = append(components, abi.ArgumentMarshaling{Name: "absolutePrestate", Type: "bytes32"})
	}
	components = append(components,
		abi.ArgumentMarshaling{Name: "l2ChainID", Type: "uint256"},
		abi.ArgumentMarshaling{Name: "proposer", Type: "address"},
	)

	tuple, err := abi.NewType("tuple", "", components)
	if err != nil {
		panic(err)
	}
	overrides, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "l1PAOMultisig", Type: "address"},
		{Name: "challenger", Type: "address"},
	})
	if err != nil {
		panic(err)
	}
	return abi.NewMethod(
		"validateWithOverrides",
		"validateWithOverrides",
		abi.Function,
		"view",
		true,
		false,
		abi.Arguments{
			{Name: "_input", Type: tuple},
			{Name: "_allowFailure", Type: MustType("bool")},
			{Name: "_overrides", Type: overrides},
		},
		abi.Arguments{{Name: "", Type: MustType("string")}},
	)
}

func ValidateStandardDeployment(
	ctx context.Context,
	backend CallContractBackend,
	validator common.Address,
	input StandardValidatorInput,
) error {
	if input.L2ChainID == nil {
		return fmt.Errorf("standard validator input has nil L2 chain ID")
	}

	method := standardValidationMethod
	var callInput any = validationInput{
		SystemConfig:     input.SystemConfig,
		AbsolutePrestate: input.AbsolutePrestate,
		L2ChainID:        input.L2ChainID,
		Proposer:         input.Proposer,
	}
	if input.UseDevFeaturesInput {
		method = standardValidationDevMethod
		callInput = validationInputDev{
			SystemConfig:       input.SystemConfig,
			CannonPrestate:     input.CannonPrestate,
			CannonKonaPrestate: input.CannonKonaPrestate,
			L2ChainID:          input.L2ChainID,
			Proposer:           input.Proposer,
		}
	}

	overrides := validationOverrides{
		L1PAOMultisig: input.L1PAOMultisig,
		Challenger:    input.Challenger,
	}
	args, err := method.Inputs.Pack(callInput, false, overrides)
	if err != nil {
		return fmt.Errorf("failed to encode standard validator input: %w", err)
	}
	calldata := append(bytes.Clone(method.ID), args...)
	result, err := backend.CallContract(ctx, ethereum.CallMsg{To: &validator, Data: calldata}, nil)
	if err != nil {
		return fmt.Errorf("standard validator call failed: %w", err)
	}
	values, err := method.Outputs.Unpack(result)
	if err != nil {
		return fmt.Errorf("failed to decode standard validator result: %w", err)
	}
	if len(values) != 1 {
		return fmt.Errorf("standard validator returned %d values", len(values))
	}
	validationErr, ok := values[0].(string)
	if !ok {
		return fmt.Errorf("standard validator returned unexpected type %T", values[0])
	}
	if expected := expectedOverrideMarker(overrides); validationErr != expected {
		return fmt.Errorf("standard validator reported errors: %s", validationErr)
	}
	return nil
}

func expectedOverrideMarker(overrides validationOverrides) string {
	switch {
	case overrides.L1PAOMultisig != (common.Address{}) && overrides.Challenger != (common.Address{}):
		return "OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER"
	case overrides.L1PAOMultisig != (common.Address{}):
		return "OVERRIDES-L1PAOMULTISIG"
	case overrides.Challenger != (common.Address{}):
		return "OVERRIDES-CHALLENGER"
	default:
		return ""
	}
}

type ScriptHostCallBackend struct {
	host *script.Host
}

func NewScriptHostCallBackend(host *script.Host) *ScriptHostCallBackend {
	return &ScriptHostCallBackend{host: host}
}

func (b *ScriptHostCallBackend) CallContract(
	ctx context.Context,
	call ethereum.CallMsg,
	blockNumber *big.Int,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if blockNumber != nil {
		return nil, fmt.Errorf("script host calls do not support a block number")
	}
	if call.To == nil {
		return nil, fmt.Errorf("script host call has no recipient")
	}

	value := uint256.NewInt(0)
	if call.Value != nil {
		var overflow bool
		value, overflow = uint256.FromBig(call.Value)
		if overflow {
			return nil, fmt.Errorf("script host call value exceeds uint256")
		}
	}
	gas := call.Gas
	if gas == 0 {
		gas = script.DefaultFoundryGasLimit
	}
	result, _, err := b.host.Call(call.From, *call.To, call.Data, gas, value)
	return result, err
}
