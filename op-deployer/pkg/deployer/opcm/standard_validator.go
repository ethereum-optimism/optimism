package opcm

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"

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
	// BridgeHookConfigured expects the PORTAL-110 the validator reports for any hook.
	// The caller must verify the hook address itself.
	BridgeHookConfigured bool
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
	// A hook's PORTAL-110 is an error to the validator, so it reverts unless failure is allowed.
	args, err := method.Inputs.Pack(callInput, input.BridgeHookConfigured, overrides)
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
	if expected := expectedValidatorResult(overrides, input.BridgeHookConfigured); validationErr != expected {
		return fmt.Errorf("standard validator reported errors: %s (expected %q)", validationErr, expected)
	}
	return nil
}

// expectedValidatorResult mirrors the validator's result: override markers, then error codes.
func expectedValidatorResult(overrides validationOverrides, bridgeHookConfigured bool) string {
	var markers []string
	if overrides.L1PAOMultisig != (common.Address{}) {
		markers = append(markers, "OVERRIDES-L1PAOMULTISIG")
	}
	if overrides.Challenger != (common.Address{}) {
		markers = append(markers, "OVERRIDES-CHALLENGER")
	}
	if bridgeHookConfigured {
		markers = append(markers, "PORTAL-110")
	}
	return strings.Join(markers, ",")
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
