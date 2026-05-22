package opcm

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const runWithBytesSignature = "runWithBytes(bytes)"

type ScriptSpec struct {
	ScriptFile      string
	ContractName    string
	ForgeScriptPath string
}

type ScriptInput map[string]any
type ScriptOutput map[string]any

type ScriptWithOutput interface {
	script.ForgeScript
	Run(input ScriptInput) (ScriptOutput, error)
}

type ScriptWithoutOutput interface {
	script.ForgeScript
	Run(input ScriptInput) error
}

type dynamicScriptWithoutOutput struct {
	script script.ForgeScript
	method abi.Method
}

type dynamicScriptWithOutput struct {
	dynamicScriptWithoutOutput
}

func NewScriptWithoutOutputFromFile(host *script.Host, spec ScriptSpec) (ScriptWithoutOutput, error) {
	forgeScript, err := script.NewForgeScriptFromFile(host, spec.ScriptFile, spec.ContractName)
	if err != nil {
		return nil, err
	}
	return NewScriptWithoutOutput(forgeScript, "run")
}

func NewScriptWithOutputFromFile(host *script.Host, spec ScriptSpec) (ScriptWithOutput, error) {
	forgeScript, err := script.NewForgeScriptFromFile(host, spec.ScriptFile, spec.ContractName)
	if err != nil {
		return nil, err
	}
	return NewScriptWithOutput(forgeScript, "run")
}

func NewScriptWithoutOutput(forgeScript script.ForgeScript, methodName string) (ScriptWithoutOutput, error) {
	method, err := validateDynamicMethod(forgeScript, methodName, false)
	if err != nil {
		return nil, err
	}
	return &dynamicScriptWithoutOutput{script: forgeScript, method: method}, nil
}

func NewScriptWithOutput(forgeScript script.ForgeScript, methodName string) (ScriptWithOutput, error) {
	method, err := validateDynamicMethod(forgeScript, methodName, true)
	if err != nil {
		return nil, err
	}
	return &dynamicScriptWithOutput{
		dynamicScriptWithoutOutput: dynamicScriptWithoutOutput{
			script: forgeScript,
			method: method,
		},
	}, nil
}

func validateDynamicMethod(forgeScript script.ForgeScript, methodName string, requireOutput bool) (abi.Method, error) {
	method, ok := forgeScript.ABI().Methods[methodName]
	if !ok {
		return abi.Method{}, fmt.Errorf("script %s does not have a method called %s", forgeScript.Name(), methodName)
	}
	if len(method.Inputs) != 1 || method.Inputs[0].Type.T != abi.TupleTy {
		return abi.Method{}, fmt.Errorf("script %s method %s must accept exactly one tuple input", forgeScript.Name(), methodName)
	}
	if requireOutput {
		if len(method.Outputs) != 1 || method.Outputs[0].Type.T != abi.TupleTy {
			return abi.Method{}, fmt.Errorf("script %s method %s must return exactly one tuple output", forgeScript.Name(), methodName)
		}
	} else if len(method.Outputs) != 0 {
		return abi.Method{}, fmt.Errorf("script %s method %s must not return output", forgeScript.Name(), methodName)
	}
	return method, nil
}

func (d *dynamicScriptWithoutOutput) ABI() abi.ABI {
	return d.script.ABI()
}

func (d *dynamicScriptWithoutOutput) Name() string {
	return d.script.Name()
}

func (d *dynamicScriptWithoutOutput) Call(input []byte) ([]byte, error) {
	return d.script.Call(input)
}

func (d *dynamicScriptWithoutOutput) Run(input ScriptInput) error {
	_, err := d.run(input)
	return err
}

func (d *dynamicScriptWithoutOutput) run(input ScriptInput) ([]byte, error) {
	packedInput, err := packMethodInput(d.ABI(), d.method, input)
	if err != nil {
		return nil, fmt.Errorf("failed to encode input for %s method of script %s: %w", d.method.RawName, d.Name(), err)
	}
	result, err := d.Call(packedInput)
	if err != nil {
		return nil, fmt.Errorf("failed to run %s method of script %s using:\n\n%v\n\n: %w", d.method.RawName, d.Name(), input, err)
	}
	return result, nil
}

func (d *dynamicScriptWithOutput) Run(input ScriptInput) (ScriptOutput, error) {
	result, err := d.dynamicScriptWithoutOutput.run(input)
	if err != nil {
		return nil, err
	}
	unpacked, err := d.ABI().Unpack(d.method.RawName, result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode output for %s method of script %s using data 0x%s: %w", d.method.RawName, d.Name(), common.Bytes2Hex(result), err)
	}
	if len(unpacked) != 1 {
		return nil, fmt.Errorf("expected 1 output value from %s, got %d", d.method.RawName, len(unpacked))
	}
	return tupleValueToMap(d.method.Outputs[0].Type, unpacked[0])
}

type dynamicBytesEncoder struct {
	inputType abi.Type
}

func (e dynamicBytesEncoder) Encode(input ScriptInput) ([]byte, error) {
	tupleValue, err := mapToTupleValue(e.inputType, input)
	if err != nil {
		return nil, err
	}
	return abi.Arguments{{Type: e.inputType}}.Pack(tupleValue.Interface())
}

type dynamicBytesDecoder struct {
	outputType abi.Type
}

func (d dynamicBytesDecoder) Decode(raw []byte) (ScriptOutput, error) {
	unpacked, err := abi.Arguments{{Type: d.outputType}}.Unpack(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack output: %w", err)
	}
	if len(unpacked) != 1 {
		return nil, fmt.Errorf("expected 1 unpacked value, got %d", len(unpacked))
	}
	return tupleValueToMap(d.outputType, unpacked[0])
}

func NewScriptForgeCaller(client *forge.Client, spec ScriptSpec) forge.ScriptCaller[ScriptInput, ScriptOutput] {
	return func(ctx context.Context, input ScriptInput, opts ...string) (ScriptOutput, bool, error) {
		var out ScriptOutput
		artifact, err := readForgeArtifact(client, spec)
		if err != nil {
			return out, false, err
		}
		method, ok := artifact.ABI.Methods["run"]
		if !ok {
			return out, false, fmt.Errorf("script %s does not have run method", spec.ContractName)
		}
		if len(method.Inputs) != 1 || method.Inputs[0].Type.T != abi.TupleTy {
			return out, false, fmt.Errorf("script %s run method must accept exactly one tuple input", spec.ContractName)
		}
		if len(method.Outputs) != 1 || method.Outputs[0].Type.T != abi.TupleTy {
			return out, false, fmt.Errorf("script %s run method must return exactly one tuple output", spec.ContractName)
		}

		caller := forge.NewScriptCaller(
			client,
			spec.ForgeScriptPath,
			runWithBytesSignature,
			dynamicBytesEncoder{inputType: method.Inputs[0].Type},
			dynamicBytesDecoder{outputType: method.Outputs[0].Type},
		)
		return caller(ctx, input, opts...)
	}
}

func readForgeArtifact(client *forge.Client, spec ScriptSpec) (*foundry.Artifact, error) {
	artifactsDir, err := findForgeArtifactsDir(client.Wd, spec.ScriptFile)
	if err != nil {
		return nil, err
	}
	artifact, err := foundry.OpenArtifactsDir(artifactsDir).ReadArtifact(spec.ScriptFile, spec.ContractName)
	if err != nil {
		return nil, fmt.Errorf("failed to load artifact %s/%s from %s: %w", spec.ScriptFile, spec.ContractName, artifactsDir, err)
	}
	return artifact, nil
}

func findForgeArtifactsDir(wd string, scriptFile string) (string, error) {
	candidates := []string{
		filepath.Join(wd, "forge-artifacts"),
		filepath.Join(wd, "out"),
		wd,
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, scriptFile)); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not find forge artifacts directory from forge working directory %s", wd)
}

func packMethodInput(contractABI abi.ABI, method abi.Method, input ScriptInput) ([]byte, error) {
	tupleValue, err := mapToTupleValue(method.Inputs[0].Type, input)
	if err != nil {
		return nil, err
	}
	return contractABI.Pack(method.RawName, tupleValue.Interface())
}

func mapToTupleValue(tuple abi.Type, input ScriptInput) (reflect.Value, error) {
	if tuple.T != abi.TupleTy {
		return reflect.Value{}, fmt.Errorf("expected tuple ABI type, got %s", tuple.String())
	}
	structType := abiReflectType(tuple)
	value := reflect.New(structType).Elem()

	known := make(map[string]struct{}, len(tuple.TupleRawNames))
	for i, name := range tuple.TupleRawNames {
		known[name] = struct{}{}
		raw, ok := input[name]
		if !ok {
			return reflect.Value{}, fmt.Errorf("missing required ABI input %q", name)
		}
		fieldValue, err := valueForABIType(*tuple.TupleElems[i], raw)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid ABI input %q: %w", name, err)
		}
		value.Field(i).Set(fieldValue)
	}
	for name := range input {
		if _, ok := known[name]; !ok {
			return reflect.Value{}, fmt.Errorf("unknown ABI input %q", name)
		}
	}
	return value, nil
}

func abiReflectType(t abi.Type) reflect.Type {
	switch t.T {
	case abi.TupleTy:
		fields := make([]reflect.StructField, len(t.TupleRawNames))
		used := make(map[string]bool, len(t.TupleRawNames))
		for i, name := range t.TupleRawNames {
			fieldName := abi.ToCamelCase(name)
			fieldName = abi.ResolveNameConflict(fieldName, func(s string) bool { return used[s] })
			used[fieldName] = true
			fields[i] = reflect.StructField{
				Name: fieldName,
				Type: abiReflectType(*t.TupleElems[i]),
				Tag:  reflect.StructTag(fmt.Sprintf(`abi:"%s" json:"%s" toml:"%s"`, name, name, name)),
			}
		}
		return reflect.StructOf(fields)
	case abi.SliceTy:
		return reflect.SliceOf(abiReflectType(*t.Elem))
	case abi.ArrayTy:
		return reflect.ArrayOf(t.Size, abiReflectType(*t.Elem))
	default:
		return t.GetType()
	}
}

func valueForABIType(t abi.Type, raw any) (reflect.Value, error) {
	target := abiReflectType(t)
	if raw == nil {
		return reflect.Value{}, fmt.Errorf("nil is not a valid %s", t.String())
	}

	if t.T == abi.TupleTy {
		input, ok := raw.(ScriptInput)
		if !ok {
			if m, ok := raw.(map[string]any); ok {
				input = ScriptInput(m)
			} else {
				return reflect.Value{}, fmt.Errorf("expected tuple map, got %T", raw)
			}
		}
		return mapToTupleValue(t, input)
	}

	if t.T == abi.SliceTy || t.T == abi.ArrayTy {
		return collectionValueForABIType(t, raw)
	}

	if t.T == abi.AddressTy {
		return scalarAddressValue(raw)
	}

	if t.T == abi.FixedBytesTy || t.T == abi.HashTy {
		return fixedBytesValue(target, raw)
	}

	if t.T == abi.BytesTy {
		return bytesValue(raw)
	}

	if t.T == abi.UintTy || t.T == abi.IntTy {
		return integerValue(target, raw)
	}

	value := reflect.ValueOf(raw)
	if value.Type().AssignableTo(target) {
		return value, nil
	}
	if value.Type().ConvertibleTo(target) {
		return value.Convert(target), nil
	}
	return reflect.Value{}, fmt.Errorf("cannot use %T as %s", raw, target)
}

func collectionValueForABIType(t abi.Type, raw any) (reflect.Value, error) {
	target := abiReflectType(t)
	rawValue := reflect.ValueOf(raw)
	if rawValue.Kind() != reflect.Slice && rawValue.Kind() != reflect.Array {
		return reflect.Value{}, fmt.Errorf("expected collection, got %T", raw)
	}
	if t.T == abi.ArrayTy && rawValue.Len() != t.Size {
		return reflect.Value{}, fmt.Errorf("expected array length %d, got %d", t.Size, rawValue.Len())
	}

	var out reflect.Value
	if t.T == abi.ArrayTy {
		out = reflect.New(target).Elem()
	} else {
		out = reflect.MakeSlice(target, rawValue.Len(), rawValue.Len())
	}
	for i := 0; i < rawValue.Len(); i++ {
		elem, err := valueForABIType(*t.Elem, rawValue.Index(i).Interface())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid element %d: %w", i, err)
		}
		out.Index(i).Set(elem)
	}
	return out, nil
}

func scalarAddressValue(raw any) (reflect.Value, error) {
	switch v := raw.(type) {
	case common.Address:
		return reflect.ValueOf(v), nil
	case string:
		return reflect.ValueOf(common.HexToAddress(v)), nil
	default:
		value := reflect.ValueOf(raw)
		target := reflect.TypeOf(common.Address{})
		if value.Type().ConvertibleTo(target) {
			return value.Convert(target), nil
		}
		return reflect.Value{}, fmt.Errorf("cannot use %T as address", raw)
	}
}

func fixedBytesValue(target reflect.Type, raw any) (reflect.Value, error) {
	value := reflect.ValueOf(raw)
	if value.IsValid() {
		if value.Type().AssignableTo(target) {
			return value, nil
		}
		if value.Type().ConvertibleTo(target) {
			return value.Convert(target), nil
		}
	}

	var bytes []byte
	switch v := raw.(type) {
	case []byte:
		bytes = v
	case string:
		trimmed := strings.TrimPrefix(v, "0x")
		decoded, err := hex.DecodeString(trimmed)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid hex bytes: %w", err)
		}
		bytes = decoded
	default:
		return reflect.Value{}, fmt.Errorf("cannot use %T as fixed bytes", raw)
	}
	if len(bytes) != target.Len() {
		return reflect.Value{}, fmt.Errorf("expected %d bytes, got %d", target.Len(), len(bytes))
	}
	out := reflect.New(target).Elem()
	for i, b := range bytes {
		out.Index(i).SetUint(uint64(b))
	}
	return out, nil
}

func bytesValue(raw any) (reflect.Value, error) {
	switch v := raw.(type) {
	case []byte:
		return reflect.ValueOf(v), nil
	case string:
		trimmed := strings.TrimPrefix(v, "0x")
		decoded, err := hex.DecodeString(trimmed)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid hex bytes: %w", err)
		}
		return reflect.ValueOf(decoded), nil
	default:
		return reflect.Value{}, fmt.Errorf("cannot use %T as bytes", raw)
	}
}

func integerValue(target reflect.Type, raw any) (reflect.Value, error) {
	value := reflect.ValueOf(raw)
	if value.IsValid() {
		if value.Type().AssignableTo(target) {
			return value, nil
		}
		if value.Type().ConvertibleTo(target) && valueCanConvertInteger(value, target) {
			return value.Convert(target), nil
		}
	}

	if s, ok := raw.(string); ok {
		return integerStringValue(target, s)
	}

	if target == reflect.TypeOf((*big.Int)(nil)) {
		switch v := raw.(type) {
		case big.Int:
			return reflect.ValueOf(new(big.Int).Set(&v)), nil
		case uint64:
			return reflect.ValueOf(new(big.Int).SetUint64(v)), nil
		case uint32:
			return reflect.ValueOf(new(big.Int).SetUint64(uint64(v))), nil
		case int:
			if v < 0 {
				return reflect.Value{}, fmt.Errorf("negative integer %d cannot be converted to uint256", v)
			}
			return reflect.ValueOf(big.NewInt(int64(v))), nil
		}
	}

	return reflect.Value{}, fmt.Errorf("cannot use %T as %s", raw, target)
}

func integerStringValue(target reflect.Type, raw string) (reflect.Value, error) {
	if target == reflect.TypeOf((*big.Int)(nil)) {
		n, ok := new(big.Int).SetString(raw, 0)
		if !ok {
			return reflect.Value{}, fmt.Errorf("invalid integer string %q", raw)
		}
		return reflect.ValueOf(n), nil
	}
	switch target.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 0, target.Bits())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid unsigned integer string %q: %w", raw, err)
		}
		out := reflect.New(target).Elem()
		out.SetUint(n)
		return out, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 0, target.Bits())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid signed integer string %q: %w", raw, err)
		}
		out := reflect.New(target).Elem()
		out.SetInt(n)
		return out, nil
	default:
		return reflect.Value{}, fmt.Errorf("cannot use integer string as %s", target)
	}
}

func valueCanConvertInteger(value reflect.Value, target reflect.Type) bool {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := value.Int()
		switch target.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return n >= 0 && uint64(n) <= maxUintForKind(target.Kind())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return n >= minIntForKind(target.Kind()) && n <= maxIntForKind(target.Kind())
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n := value.Uint()
		switch target.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return n <= maxUintForKind(target.Kind())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return n <= uint64(maxIntForKind(target.Kind()))
		}
	}
	return false
}

func maxUintForKind(kind reflect.Kind) uint64 {
	switch kind {
	case reflect.Uint8:
		return math.MaxUint8
	case reflect.Uint16:
		return math.MaxUint16
	case reflect.Uint32:
		return math.MaxUint32
	case reflect.Uint, reflect.Uint64:
		return math.MaxUint64
	default:
		return 0
	}
}

func minIntForKind(kind reflect.Kind) int64 {
	switch kind {
	case reflect.Int8:
		return math.MinInt8
	case reflect.Int16:
		return math.MinInt16
	case reflect.Int32:
		return math.MinInt32
	case reflect.Int, reflect.Int64:
		return math.MinInt64
	default:
		return 0
	}
}

func maxIntForKind(kind reflect.Kind) int64 {
	switch kind {
	case reflect.Int8:
		return math.MaxInt8
	case reflect.Int16:
		return math.MaxInt16
	case reflect.Int32:
		return math.MaxInt32
	case reflect.Int, reflect.Int64:
		return math.MaxInt64
	default:
		return 0
	}
}

func tupleValueToMap(tuple abi.Type, raw any) (ScriptOutput, error) {
	if tuple.T != abi.TupleTy {
		return nil, fmt.Errorf("expected tuple ABI type, got %s", tuple.String())
	}
	value := reflect.ValueOf(raw)
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected tuple struct, got %T", raw)
	}

	out := make(ScriptOutput, len(tuple.TupleRawNames))
	for i, name := range tuple.TupleRawNames {
		field := value.Field(i)
		normalized, err := normalizeOutputValue(*tuple.TupleElems[i], field)
		if err != nil {
			return nil, fmt.Errorf("invalid ABI output %q: %w", name, err)
		}
		out[name] = normalized
	}
	return out, nil
}

func normalizeOutputValue(t abi.Type, value reflect.Value) (any, error) {
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return nil, nil
	}
	switch t.T {
	case abi.TupleTy:
		return tupleValueToMap(t, value.Interface())
	case abi.SliceTy, abi.ArrayTy:
		out := make([]any, value.Len())
		for i := 0; i < value.Len(); i++ {
			normalized, err := normalizeOutputValue(*t.Elem, value.Index(i))
			if err != nil {
				return nil, err
			}
			out[i] = normalized
		}
		return out, nil
	case abi.FixedBytesTy, abi.HashTy:
		bytes, err := fixedBytesFromValue(value)
		if err != nil {
			return nil, err
		}
		if len(bytes) == common.HashLength {
			return common.BytesToHash(bytes), nil
		}
		return bytes, nil
	default:
		return value.Interface(), nil
	}
}

func fixedBytesFromValue(value reflect.Value) ([]byte, error) {
	if value.Kind() != reflect.Array {
		return nil, fmt.Errorf("expected fixed bytes array, got %s", value.Kind())
	}
	bytes := make([]byte, value.Len())
	for i := 0; i < value.Len(); i++ {
		elem := value.Index(i)
		if elem.Kind() != reflect.Uint8 {
			return nil, fmt.Errorf("expected fixed bytes element uint8, got %s", elem.Kind())
		}
		bytes[i] = byte(elem.Uint())
	}
	return bytes, nil
}

func (o ScriptOutput) Address(name string) common.Address {
	return addressFromMap("output", o, name)
}

func (i ScriptInput) Address(name string) common.Address {
	return addressFromMap("input", i, name)
}

func (o ScriptOutput) Hash(name string) common.Hash {
	value, ok := o[name]
	if !ok {
		panic(fmt.Sprintf("missing ABI output %q", name))
	}
	switch v := value.(type) {
	case common.Hash:
		return v
	case [32]byte:
		return common.Hash(v)
	case string:
		return common.HexToHash(v)
	default:
		panic(fmt.Sprintf("ABI output %q is %T, not bytes32", name, value))
	}
}

func addressFromMap(kind string, values map[string]any, name string) common.Address {
	value, ok := values[name]
	if !ok {
		panic(fmt.Sprintf("missing ABI %s %q", kind, name))
	}
	switch v := value.(type) {
	case common.Address:
		return v
	case string:
		return common.HexToAddress(v)
	default:
		panic(fmt.Sprintf("ABI %s %q is %T, not address", kind, name, value))
	}
}
