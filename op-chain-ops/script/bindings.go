package script

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"
)

// CallBackendFn is the function that encoded binding calls get made with.
// The function may return vm.ErrExecutionReverted to revert (revert is ABI decoded from data).
// Or any other error, where the return-data is then ignored.
type CallBackendFn func(data []byte) ([]byte, error)

// MakeBindings turns a struct type with function-typed fields into EVM call bindings
// that are hooked up to the backend function.
// fields annotated with `evm:"-"` are ignored.
func MakeBindings[E any](backendFn CallBackendFn,
	checkABI func(abiDef string) bool,
) (*E, error) {
	v := new(E)
	val := reflect.ValueOf(v)
	err := hydrateBindingsStruct(val, backendFn, checkABI)
	return v, err
}

// hydrateBindingsStruct initializes a struct with function fields into
// a struct of ABI functions hooked up to the backend.
func hydrateBindingsStruct(
	val reflect.Value,
	backendFn CallBackendFn,
	checkABI func(abiDef string) bool,
) error {
	typ := val.Type()
	if typ.Kind() == reflect.Pointer {
		if val.IsNil() {
			return errors.New("cannot hydrate nil pointer value")
		}
		val = val.Elem()
		typ = val.Type()
	}
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("object is not a struct: %s", typ)
	}
	// Hydrate each of the fields
	fieldCount := val.NumField()
	for i := 0; i < fieldCount; i++ {
		fieldDef := typ.Field(i)
		if !fieldDef.IsExported() { // ignore unexposed fields
			continue
		}
		// ignore fields / embedded structs that are annotated with `evm:"-"`
		if v, ok := fieldDef.Tag.Lookup("evm"); ok && v == "-" {
			continue
		}
		if fieldDef.Anonymous { // fields of embedded structs will be hydrated too
			if err := hydrateBindingsStruct(val.Field(i), backendFn, checkABI); err != nil {
				return fmt.Errorf("failed to hydrate bindings of embedded field %q: %w", fieldDef.Name, err)
			}
			continue
		}
		if fieldDef.Type.Kind() != reflect.Func { // We can only hydrate fields with a function type
			continue
		}
		fVal := val.Field(i)
		if !fVal.IsNil() {
			return fmt.Errorf("cannot hydrate bindings func, field %q is already set", fieldDef.Name)
		}
		if err := hydrateBindingsField(fVal, fieldDef, backendFn, checkABI); err != nil {
			return fmt.Errorf("cannot hydrate bindings field %q: %w", fieldDef.Name, err)
		}
	}
	return nil
}

var ErrABICheck = errors.New("failed ABI check")

// hydrateBindingsField initializes a struct field value to a function that calls the implied ABI function.
func hydrateBindingsField(
	fVal reflect.Value,
	fieldDef reflect.StructField,
	backendFn CallBackendFn,
	checkABI func(abiDef string) bool,
) error {
	// derive the ABI function name from the field name
	abiFunctionName := fieldDef.Name
	if len(abiFunctionName) == 0 {
		return errors.New("ABI method name must not be empty")
	}
	if lo := strings.ToLower(abiFunctionName[:1]); lo != abiFunctionName[:1] {
		abiFunctionName = lo + abiFunctionName[1:]
	}

	// derive the ABI function arguments from the function type
	inArgs, err := makeArgs(fieldDef.Type.NumIn(), fieldDef.Type.In)
	if err != nil {
		return fmt.Errorf("failed to determine ABI types of input args: %w", err)
	}
	inArgTypes := makeArgTypes(inArgs)
	methodSig := fmt.Sprintf("%v(%v)", abiFunctionName, strings.Join(inArgTypes, ","))

	// check the ABI, if we can
	if checkABI != nil {
		if !checkABI(methodSig) {
			return fmt.Errorf("function %s with signature %q is invalid: %w", fieldDef.Name, methodSig, ErrABICheck)
		}
	}
	byte4Sig := bytes4(methodSig)

	// Define how we encode Go arguments as function calldata, including the function selector
	inArgsEncodeFn := func(args []reflect.Value) ([]byte, error) {
		vals := make([]interface{}, len(args))
		for i := range args {
			vals[i] = args[i].Interface()
		}
		out, err := inArgs.PackValues(vals)
		if err != nil {
			return nil, fmt.Errorf("failed to encode call data: %w", err)
		}
		calldata := make([]byte, 0, len(out)+4)
		calldata = append(calldata, byte4Sig[:]...)
		calldata = append(calldata, out...)
		return calldata, nil
	}
	// Determine how many arguments we decode from ABI, and if we have an error return case.
	outArgCount := fieldDef.Type.NumOut()
	errReturn := hasTrailingError(outArgCount, fieldDef.Type.Out)
	var nilErrValue reflect.Value
	if errReturn {
		outArgCount -= 1
		nilErrValue = reflect.New(fieldDef.Type.Out(outArgCount)).Elem()
	}
	outArgs, err := makeArgs(outArgCount, fieldDef.Type.Out)
	if err != nil {
		return fmt.Errorf("failed to determine ABI types of output args: %w", err)
	}
	outAllocators := makeArgAllocators(outArgCount, fieldDef.Type.Out)
	// Helper func to return an error with, where we try to fit it in the returned error value, if there is any.
	returnErr := func(err error) []reflect.Value {
		if !errReturn {
			panic(fmt.Errorf("error, but cannot return as arg: %w", err))
		}
		out := make([]reflect.Value, outArgCount+1)
		for i := 0; i < outArgCount; i++ {
			out[i] = reflect.New(fieldDef.Type.Out(i)).Elem()
		}
		out[outArgCount] = reflect.ValueOf(err)
		return out
	}
	// Decodes the result of the backend into values to return as function, including error/revert handling.
	outDecodeFn := func(result []byte, resultErr error) []reflect.Value {
		if resultErr != nil {
			// Empty return-data might happen on a regular description-less revert. No need to unpack in that case.
			if len(result) > 0 && errors.Is(resultErr, vm.ErrExecutionReverted) {
				msg, err := abi.UnpackRevert(result)
				if err != nil {
					return returnErr(fmt.Errorf("failed to unpack result args: %w", err))
				}
				return returnErr(fmt.Errorf("revert: %s", msg))
			}
			return returnErr(resultErr)
		}
		out := make([]reflect.Value, outArgCount, outArgCount+1)
		err := abiToValues(outArgs, outAllocators, out, result)
		if err != nil {
			return returnErr(fmt.Errorf("failed to convert output to return values: %w", err))
		}
		if errReturn { // don't forget the nil error value, to match the expected output arg count
			out = append(out, nilErrValue)
		}
		return out
	}
	// Construct the actual Go function: it encodes the Go args, turns it into an ABI call, and decodes the results.
	f := reflect.MakeFunc(fieldDef.Type, func(args []reflect.Value) (results []reflect.Value) {
		input, err := inArgsEncodeFn(args) // ABI encode args
		if err != nil {
			return returnErr(fmt.Errorf("failed to encode input args: %w", err))
		}
		result, err := backendFn(input) // call backend func
		return outDecodeFn(result, err) // ABI decode result
	})
	// Now hydrate the field definition with our new Go function
	fVal.Set(f)
	return nil
}
