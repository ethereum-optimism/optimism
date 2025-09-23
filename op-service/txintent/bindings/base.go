package bindings

import (
	"context"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// function fields(lambdas) corresponding to solidity functions must be tagged with sol
// tag value is used for initializing solidity function selector
const MethodTagName string = "sol"

// Bindings field is a user supplied struct which has lambdas as a field
const BindingsFieldName string = "Bindings"

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
	Require() *testreq.Assertions
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

// CheckImpl validates that the given struct satisfies the form BindingWrapper, which is initialized
// using binding struct that user provided, and the injected binding factory.
// User provided binding struct is checked that it has correctly defined function fields:
// Each function field must have a `sol` tag (MethodTagName).
func CheckImpl(v reflect.Value) (reflect.Value, reflect.Value) {
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic("expected struct")
	}
	baseCallFactory := findBaseCallFactory(v)
	if !baseCallFactory.IsValid() {
		panic("BaseCallFactory not found in embedded fields")
	}
	bindings := findBindings(v)
	if !bindings.IsValid() {
		panic("Bindings not found in embedded fields")
	}
	bindingType := bindings.Type()
	for i := range bindingType.NumField() {
		field := bindingType.Field(i)
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
	return baseCallFactory, bindings
}

// findBaseCallFactory recursively searches the struct for an embedded BaseCallFactory and returns its value
func findBaseCallFactory(v reflect.Value) reflect.Value {
	for i := range v.NumField() {
		field := v.Field(i)
		if !field.CanInterface() {
			continue
		}
		t := field.Type()
		if t == reflect.TypeOf(BaseCallFactory{}) {
			return field
		}
		if t.Kind() == reflect.Struct {
			if found := findBaseCallFactory(field); found.IsValid() {
				return found
			}
		}
	}
	return reflect.Value{}
}

func findBindings(v reflect.Value) reflect.Value {
	return v.FieldByName(BindingsFieldName)
}

// InitImpl initializes function fields (lambdas) in the given struct by assigning concrete implementations
// The input struct must be a pointer, and its fields are expected to follow a specific pattern for reflection-based setup
func InitImpl[T any](impl *T) {
	v := reflect.ValueOf(impl).Elem()
	baseCallFactory, bindings := CheckImpl(v)
	bindingsType := bindings.Type()
	for i := range bindingsType.NumField() {
		field := bindingsType.Field(i)
		fieldType := field.Type
		// Only care about function fields
		if fieldType.Kind() == reflect.Func {
			methodName := field.Tag.Get(MethodTagName)
			inputTypes := []reflect.Type{}
			for j := range fieldType.NumIn() {
				inputTypes = append(inputTypes, fieldType.In(j))
			}
			outputType := fieldType.Out(0)
			// inner: func() -> (bytes[], error)
			funcInputRet := []reflect.Type{reflect.TypeFor[[]byte](), reflect.TypeFor[error]()}
			funcInput := reflect.FuncOf([]reflect.Type{}, funcInputRet, false)
			// outer: func(args...) -> inner: (func() -> (bytes[], error))
			funcInputWrapper := reflect.FuncOf(inputTypes, []reflect.Type{funcInput}, false)

			// encoderOuter is a higher order function which returns encoderInner
			// encoderInner is a closure, binded with solidity method arguments and lazily evaluated
			encoderOuter := reflect.MakeFunc(funcInputWrapper, func(argsOuter []reflect.Value) []reflect.Value {
				encoderInner := reflect.MakeFunc(funcInput, func(argsInner []reflect.Value) []reflect.Value {
					callArgs := make([]any, len(argsOuter))
					for i, a := range argsOuter {
						callArgs[i] = a.Interface()
					}
					v0, v1 := ABIEncoder(methodName, callArgs...)
					ret := []reflect.Value{reflect.Zero(funcInputRet[0]), reflect.Zero(funcInputRet[1])}
					if v0 != nil { // bytes[]
						ret[0] = reflect.ValueOf(v0)
					}
					if v1 != nil { // error
						ret[1] = reflect.ValueOf(v1)
					}
					return ret
				})
				return []reflect.Value{encoderInner}
			})

			// Initialize actual binding function fields
			lambda := reflect.MakeFunc(fieldType, func(args []reflect.Value) []reflect.Value {
				innerResults := encoderOuter.Call(args)
				if len(innerResults) != 1 {
					panic("expected one return value")
				}
				encoderLambda := innerResults[0]
				typedCall := reflect.New(outputType).Elem()
				typedCall.FieldByName("MethodName").Set(reflect.ValueOf(methodName))
				typedCall.FieldByName("EncodeInputLambda").Set(encoderLambda)
				typedCall.FieldByName("BaseCallFactory").Set(baseCallFactory.Addr())
				return []reflect.Value{typedCall}
			})
			bindings.FieldByName(field.Name).Set(lambda)
		}
	}
}

// Call implements txintent Call interface
type Call struct {
	*BaseCallFactory

	MethodName        string
	EncodeInputLambda func() ([]byte, error)
}

func (c *Call) EncodeInput() ([]byte, error) {
	return c.EncodeInputLambda()
}

var _ txintent.Call = (*Call)(nil)

// TypedCall implements txintent CallView interface
type TypedCall[ReturnType any] struct {
	Call
}

var _ txintent.CallView[any] = (*TypedCall[any])(nil)

// CustomTypeToGoType converts custom type to go type
func CustomTypeToGoType(retTyp reflect.Type) reflect.Type {
	switch retTyp {
	case reflect.TypeOf(eth.ETH{}), reflect.TypeOf(eth.ChainID{}):
		return reflect.TypeOf(big.NewInt(0))
	case reflect.TypeOf(suptypes.Identifier{}):
		return reflect.TypeOf(ABIIdentifier{})
	default:
		return retTyp
	}
}

// ReplaceCustomInts recursively replaces all Uint128/Int128 (and pointer forms) with *big.Int
func ReplaceCustomInts(value any) (any, error) {
	return replaceCustomIntValue(reflect.ValueOf(value))
}

func unwrapCustomInt(val reflect.Value) (any, bool) {
	// Unwrap pointer chain
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			// Return zero value *big.Int only for known pointer types
			if customIntTypes[val.Type()] {
				return new(big.Int), true
			}
			return nil, false
		}
		val = val.Elem()
	}
	if !val.IsValid() {
		return nil, false
	}
	// Must match known custom int type
	if !customIntTypes[val.Type()] {
		return nil, false
	}
	// Confirm convertible to big.Int
	if !val.Type().ConvertibleTo(bigIntType) {
		return nil, false
	}
	converted := val.Convert(bigIntType).Interface().(big.Int)
	b := new(big.Int)
	b.Set(&converted)
	return b, true
}

func replaceCustomIntValue(val reflect.Value) (any, error) {
	typ := val.Type()
	// Skip native *big.Int
	if typ == reflect.TypeOf((*big.Int)(nil)) {
		return val.Interface(), nil
	}
	// custom ints to *big.Int
	if converted, ok := unwrapCustomInt(val); ok {
		return converted, nil
	}
	switch typ.Kind() {
	case reflect.Ptr:
		if val.IsNil() {
			return nil, nil
		}
		elemConverted, err := replaceCustomIntValue(val.Elem())
		if err != nil {
			return nil, err
		}
		ptr := reflect.New(reflect.TypeOf(elemConverted))
		ptr.Elem().Set(reflect.ValueOf(elemConverted))
		return ptr.Interface(), nil
	case reflect.Struct:
		return replaceStruct(val)
	case reflect.Array:
		return replaceArray(val)
	case reflect.Slice:
		return replaceSlice(val)
	default:
		return val.Interface(), nil
	}
}

func replaceStruct(val reflect.Value) (any, error) {
	typ := val.Type()
	var fields []reflect.StructField
	var values []reflect.Value

	for i := range typ.NumField() {
		field := typ.Field(i)
		fieldVal := val.Field(i)
		if !fieldVal.CanInterface() {
			return nil, fmt.Errorf("field %s must be exported", field.Name)
		}
		converted, err := replaceCustomIntValue(fieldVal)
		if err != nil {
			return nil, err
		}
		fields = append(fields, reflect.StructField{
			Name: field.Name,
			Type: reflect.TypeOf(converted),
			Tag:  field.Tag,
		})
		values = append(values, reflect.ValueOf(converted))
	}

	newType := reflect.StructOf(fields)
	newStruct := reflect.New(newType).Elem()
	for i := range values {
		newStruct.Field(i).Set(values[i])
	}
	return newStruct.Interface(), nil
}

func replaceSequence(val reflect.Value, makeContainer func(reflect.Type, int) reflect.Value) (any, error) {
	length := val.Len()
	if length == 0 {
		return val.Interface(), nil
	}
	var resultElemType reflect.Type
	elemValues := make([]reflect.Value, length)
	for i := range length {
		converted, err := replaceCustomIntValue(val.Index(i))
		if err != nil {
			return nil, err
		}
		elem := reflect.ValueOf(converted)
		elemValues[i] = elem
		resultElemType = elem.Type()
	}
	container := makeContainer(resultElemType, length)
	for i := range length {
		container.Index(i).Set(elemValues[i])
	}
	return container.Interface(), nil
}

func replaceSlice(val reflect.Value) (any, error) {
	return replaceSequence(val, func(elemType reflect.Type, length int) reflect.Value {
		return reflect.MakeSlice(reflect.SliceOf(elemType), length, length)
	})
}

func replaceArray(val reflect.Value) (any, error) {
	return replaceSequence(val, func(elemType reflect.Type, length int) reflect.Value {
		return reflect.New(reflect.ArrayOf(length, elemType)).Elem()
	})
}

// CustomValueToABIValue converts custom value to abi value
func CustomValueToABIValue(arg any) any {
	var value any
	switch v := arg.(type) {
	case eth.ETH:
		value = v.ToBig()
	case eth.ChainID:
		value = v.ToBig()
	case suptypes.Identifier:
		identifier := ABIIdentifier{
			Origin:      v.Origin,
			BlockNumber: big.NewInt(int64(v.BlockNumber)),
			LogIndex:    big.NewInt(int64(v.LogIndex)),
			Timestamp:   big.NewInt(int64(v.Timestamp)),
			ChainId:     v.ChainID.ToBig(),
		}
		value = identifier
	default:
		var err error
		value, err = ReplaceCustomInts(v)
		if err != nil {
			panic(fmt.Errorf("failed to replace custom int: %w", err))
		}
	}
	return value
}

// ABIValueToCustomValue converts abi value to custom value
func ABIValueToCustomValue[ReturnType any](retTyp reflect.Type, val any) ReturnType {
	var zero ReturnType
	switch retTyp {
	case reflect.TypeOf(eth.ETH{}):
		bigVal := abi.ConvertType(val, new(big.Int)).(*big.Int)
		var concrete eth.ETH
		if (*uint256.Int)(&concrete).SetFromBig(bigVal) {
			return zero
		}
		return any(concrete).(ReturnType)
	case reflect.TypeOf(eth.ChainID{}):
		bigVal := abi.ConvertType(val, new(big.Int)).(*big.Int)
		var concrete eth.ChainID
		if (*uint256.Int)(&concrete).SetFromBig(bigVal) {
			return zero
		}
		return any(concrete).(ReturnType)
	case reflect.TypeOf(Uint128{}):
		bigVal := abi.ConvertType(val, new(big.Int)).(*big.Int)
		return any(Uint128(*bigVal)).(ReturnType)
	case reflect.TypeOf(&Uint128{}):
		bigVal := abi.ConvertType(val, new(big.Int)).(*big.Int)
		return any((*Uint128)(bigVal)).(ReturnType)
	case reflect.TypeOf(Int128{}):
		bigVal := abi.ConvertType(val, new(big.Int)).(*big.Int)
		return any(Int128(*bigVal)).(ReturnType)
	case reflect.TypeOf(&Int128{}):
		bigVal := abi.ConvertType(val, new(big.Int)).(*big.Int)
		return any((*Int128)(bigVal)).(ReturnType)
	default:
		ptr := abi.ConvertType(val, new(ReturnType)).(*ReturnType)
		return *ptr
	}
}

// DecodeOutput unwraps ReturnType from TypedCall and abi decodes byte string
func (c *TypedCall[ReturnType]) DecodeOutput(data []byte) (ReturnType, error) {
	var zero ReturnType
	retTyp := reflect.TypeOf(zero)

	// nothing to decode since ReturnType was marked as any
	if retTyp == nil {
		return *new(ReturnType), nil
	}

	abiTargetType := CustomTypeToGoType(retTyp)
	abiType, components, err := goTypeToABIType(abiTargetType)
	if err != nil {
		return *new(ReturnType), fmt.Errorf("failed to convert go type to abi type: %w", err)
	}

	outputs := abi.Arguments{{Type: abiType}}
	// try to unpack assuming every field is static
	decoded, err := outputs.Unpack(data)
	if err != nil {
		// at lest one dynamic field is included so unpack by mimicing abi.UnpackIntoInterface method
		args := abi.Arguments{}
		for idx, component := range components {
			t, err := abi.NewType(component.Type, "", component.Components)
			if err != nil {
				return *new(ReturnType), fmt.Errorf("failed to create type: %w", err)
			}
			name := component.Name
			// make sure name is properly set and unique
			if name == "" || name == "_" {
				name = fmt.Sprintf("arg%d", idx)
			}
			args = append(args, abi.Argument{Type: t, Name: name})
		}
		decoded, err = args.Unpack(data)
		if err != nil {
			// we do not support custom value decoding when struct with dynamic fields.
			// using with eth.ETH or eth.ChainID will fail
			return *new(ReturnType), fmt.Errorf("failed to unpack: %w", err)
		}
		var val ReturnType
		err = args.Copy(&val, decoded)
		if err != nil {
			return *new(ReturnType), fmt.Errorf("failed to convert go format to provided struct: %w", err)
		}
		return val, nil
	}
	val := ABIValueToCustomValue[ReturnType](retTyp, decoded[0])
	return val, nil
}

// ABIEncoder abi encodes arguments with function name
func ABIEncoder(name string, args ...any) ([]byte, error) {
	inputs := make([]abi.Argument, len(args))
	argsTranslated := make([]any, len(args))
	for i, arg := range args {
		goType := CustomTypeToGoType(reflect.TypeOf(arg))
		abiValue := CustomValueToABIValue(arg)
		abiType, _, err := goTypeToABIType(goType)
		if err != nil {
			panic(err)
		}
		inputs[i] = abi.Argument{Type: abiType}
		argsTranslated[i] = abiValue
	}

	// Internally initialise sig and ID
	// Use dummy vars but calldata does not care
	method := abi.NewMethod(name, name, abi.Function, "payable", false, false, inputs, abi.Arguments{})
	arguments, err := method.Inputs.Pack(argsTranslated...)
	if err != nil {
		panic(err)
	}
	result := append(method.ID, arguments...)

	return result, err
}

type BindingsWrapper[T any] struct {
	BaseCallFactory
	Bindings T
}

// NewBindings is a helper function to inject base call factory and initialize the contract bindings implementation
func NewBindings[T any](opts ...CallFactoryOption) T {
	bindingsWrapper := BindingsWrapper[T]{
		BaseCallFactory: *NewBaseCallFactory(opts...),
		Bindings:        *new(T),
	}
	InitImpl(&bindingsWrapper)
	return bindingsWrapper.Bindings
}
