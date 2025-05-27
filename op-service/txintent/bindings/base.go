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

func CheckImpl(parent any) {
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

func InitImpl[T any](impl *T, factory *BaseCallFactory) {
	v := reflect.ValueOf(impl).Elem()
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic("expected struct")
	}
	for i := range v.NumField() {
		field := t.Field(i)
		fieldType := field.Type
		if fieldType.Kind() == reflect.Func {
			methodName := field.Tag.Get(MethodTagName)

			inputTypes := []reflect.Type{}
			for j := range fieldType.NumIn() {
				inputTypes = append(inputTypes, fieldType.In(j))
			}
			outputType := fieldType.Out(0)

			// outer: func(...args) -> <inner: (func() -> (bytes[], error))>
			// inner: func() -> (bytes[], error)
			funcInput := reflect.FuncOf([]reflect.Type{}, []reflect.Type{reflect.TypeOf([]byte{}), reflect.TypeOf((*error)(nil)).Elem()}, false)
			funcInputWrapper := reflect.FuncOf(inputTypes, []reflect.Type{funcInput}, false)

			encoderLambdaLambda := reflect.MakeFunc(funcInputWrapper, func(argsOuter []reflect.Value) []reflect.Value {
				encoderLambda := reflect.MakeFunc(funcInput, func(argsInner []reflect.Value) []reflect.Value {
					callArgs := make([]any, len(argsOuter))
					for i, a := range argsOuter {
						callArgs[i] = a.Interface()
					}
					v0, v1 := encoder(methodName, callArgs...)

					// guard
					var val0 reflect.Value
					if v0 == nil {
						val0 = reflect.Zero(reflect.TypeOf([]byte{}))
					} else {
						val0 = reflect.ValueOf(v0)
					}
					var val1 reflect.Value
					if v1 == nil {
						val1 = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())
					} else {
						val1 = reflect.ValueOf(v1)
					}

					return []reflect.Value{val0, val1}
				})
				inner := encoderLambda.Interface().(func() ([]byte, error))
				return []reflect.Value{reflect.ValueOf(inner)}
			})

			λ := reflect.MakeFunc(fieldType, func(args []reflect.Value) []reflect.Value {
				innerResults := encoderLambdaLambda.Call(args)
				if len(innerResults) != 1 {
					panic("expected one return value")
				}
				innerλ := innerResults[0].Interface().(func() ([]byte, error))
				wrap := reflect.New(outputType).Elem()
				wrap.FieldByName("MethodName").Set(reflect.ValueOf(methodName))
				wrap.FieldByName("EncodeInputLambda").Set(reflect.ValueOf(innerλ))
				wrap.FieldByName("BaseCallFactory").Set(reflect.ValueOf(factory))
				return []reflect.Value{wrap}
			})
			v.FieldByName(field.Name).Set(λ)
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
