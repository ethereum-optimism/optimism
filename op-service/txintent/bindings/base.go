package bindings

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

const MethodTagName string = "sol"

// BaseCall contains fields to populate fields of txplan
type BaseCall struct {
	target     common.Address
	accessList types.AccessList
}

func (c *BaseCall) To() (*common.Address, error) {
	return &c.target, nil
}

func (c *BaseCall) AccessList() (types.AccessList, error) {
	return c.accessList, nil
}

// BaseCall contains client for reading the blockchain
type BaseCallView struct {
	client apis.EthClient
}

func (c *BaseCallView) Client() apis.EthClient {
	return c.client
}

// BaseCall represents minimal testing interface
type BaseTest interface {
	Require() *require.Assertions
	Ctx() context.Context
}

// BaseCallTest contains tester to embed for the CallFactory
type BaseCallTest struct {
	t BaseTest
}

func (c *BaseCallTest) Test() BaseTest {
	return c.t
}

// BaseCallFactory composes building blocks for initializing contract factory.
// Intended to be embedded while adding contract binding factory.
type BaseCallFactory struct {
	BaseCall
	BaseCallView
	BaseCallTest
}

// Options to populate the factory
type CallFactoryOption func(*BaseCallFactory)

func WithTo(target common.Address) CallFactoryOption {
	return func(f *BaseCallFactory) {
		f.target = target
	}
}

func WithClient(client apis.EthClient) CallFactoryOption {
	return func(f *BaseCallFactory) {
		f.client = client
	}
}

func WithTest(t BaseTest) CallFactoryOption {
	return func(f *BaseCallFactory) {
		f.t = t
	}
}

func NewBaseCallFactory(opts ...CallFactoryOption) *BaseCallFactory {
	b := &BaseCallFactory{}
	b.ApplyFactoryOptions(opts...)
	return b
}

func (b *BaseCallFactory) ApplyFactoryOptions(opts ...CallFactoryOption) {
	for _, opt := range opts {
		opt(b)
	}
}

func CheckFactoryImpl(parent any) {
	t := reflect.TypeOf(parent)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := range t.NumField() {
		field := t.Field(i)
		fieldType := field.Type
		// check only function fields, which will be automatically inferred for codec
		if fieldType.Kind() != reflect.Func {
			continue
		}
		if len(field.Tag.Get(MethodTagName)) == 0 {
			panic(fmt.Sprintf("all methods must have `%s` tags for calldata", MethodTagName))
		}
		if fieldType.NumOut() != 1 {
			panic("all methods must have single return type")
		}
	}
}

type Call struct {
	*BaseCallFactory

	MethodName        string
	EncodeInputLambda func() ([]byte, error)
}

func (c *Call) EncodeInput() ([]byte, error) {
	return c.EncodeInputLambda()
}

var _ txintent.Call = (*Call)(nil)

type TypedCall[ReturnType any] struct {
	Call
}

func (c *TypedCall[ReturnType]) DecodeOutput(data []byte) (ReturnType, error) {
	var zero ReturnType
	retTyp := reflect.TypeOf(zero)

	// Special handling for eth.ETH
	var abiTargetType reflect.Type
	if retTyp == reflect.TypeOf(eth.ETH{}) {
		abiTargetType = reflect.TypeOf(big.NewInt(0))
	} else {
		abiTargetType = retTyp
	}

	abiType, err := script.GoTypeToABIType(abiTargetType)
	if err != nil {
		return zero, fmt.Errorf("failed to convert Go type to ABI type: %w", err)
	}

	outputs := abi.Arguments{{Type: abiType}}
	decoded, err := outputs.Unpack(data)
	if err != nil {
		return zero, fmt.Errorf("ABI unpack error: %w", err)
	}

	// TODO: handle multiple returns
	val := decoded[0]

	// Special handling for eth.ETH
	switch retTyp {
	case reflect.TypeOf(eth.ETH{}):
		bigVal := abi.ConvertType(val, new(big.Int)).(*big.Int)
		var concrete eth.ETH
		if (*uint256.Int)(&concrete).SetFromBig(bigVal) {
			return zero, errors.New("result conversion failure: does not fit in uint256")
		}
		return any(concrete).(ReturnType), nil
	default:
		ptr := abi.ConvertType(val, new(ReturnType)).(*ReturnType)
		return *ptr, nil
	}
}

var _ txintent.CallView[any] = (*TypedCall[any])(nil)

func encoder(name string, args ...any) ([]byte, error) {
	inputs := []abi.Argument{}
	args_translated := []any{}
	for i, arg := range args {
		var typ reflect.Type
		// handle op service types
		switch v := arg.(type) {
		case eth.ETH:
			argsTyped := v.ToBig()
			typ = reflect.TypeOf(argsTyped)
			args_translated = append(args_translated, argsTyped)
		default:
			typ = reflect.TypeOf(arg)
			args_translated = append(args_translated, arg)
		}
		abiTyp, err := script.GoTypeToABIType(typ)
		if err != nil {
			panic("go type to abi type")
		}
		input := abi.Argument{
			Name: fmt.Sprintf("arg_%d", i),
			Type: abiTyp,
		}
		inputs = append(inputs, input)
	}

	// Internally initialise sig and ID
	// Use dummy vars but calldata does not care
	method := abi.NewMethod(name, name, abi.Function, "payable", false, false, inputs, abi.Arguments{})
	arguments, err := method.Inputs.Pack(args_translated...)
	if err != nil {
		panic(err)
	}
	result := append(method.ID, arguments...)

	return result, err
}
