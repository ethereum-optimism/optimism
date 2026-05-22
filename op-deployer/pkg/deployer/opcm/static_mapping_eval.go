package opcm

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type StaticInputSources map[string]any

func EvaluateStaticInputMapping(mapping StaticInputMapping, sources StaticInputSources) (ScriptInput, error) {
	input := make(ScriptInput, len(mapping.Input))
	for name, expr := range mapping.Input {
		value, ok, err := evalStaticInputExpr(expr, sources)
		if err != nil {
			return nil, fmt.Errorf("evaluate static input %q: %w", name, err)
		}
		if !ok {
			return nil, fmt.Errorf("evaluate static input %q: no value resolved", name)
		}
		input[name] = value
	}
	return input, nil
}

func EvaluateStaticInputMappingForABI(contractABI abi.ABI, mapping StaticInputMapping, sources StaticInputSources) (ScriptInput, error) {
	methodName := mapping.Script.Function
	if methodName == "" {
		methodName = "run"
	}
	method, ok := contractABI.Methods[methodName]
	if !ok {
		return nil, fmt.Errorf("method %q not found in ABI", methodName)
	}
	if err := ValidateStaticInputMapping(contractABI, mapping); err != nil {
		return nil, err
	}
	input, err := EvaluateStaticInputMapping(mapping, sources)
	if err != nil {
		return nil, err
	}
	if _, err := mapToTupleValue(method.Inputs[0].Type, input); err != nil {
		return nil, fmt.Errorf("validate evaluated static input: %w", err)
	}
	return input, nil
}

func evalStaticInputExpr(expr StaticInputExpr, sources StaticInputSources) (any, bool, error) {
	var value any
	var ok bool
	switch {
	case expr.From != "":
		var err error
		value, ok, err = resolveStaticInputPath(sources, expr.From)
		if err != nil {
			return nil, false, err
		}
	case len(expr.Coalesce) > 0:
		for i, child := range expr.Coalesce {
			childValue, childOK, err := evalStaticInputExpr(child, sources)
			if err != nil {
				return nil, false, fmt.Errorf("coalesce[%d]: %w", i, err)
			}
			if childOK {
				value = childValue
				ok = true
				break
			}
		}
	case expr.Value != nil:
		value = expr.Value
		ok = true
	}
	if !ok || isNilStaticInputValue(value) {
		return nil, false, nil
	}
	value, err := applyStaticInputTransform(value, expr.Transform)
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func resolveStaticInputPath(sources StaticInputSources, path string) (any, bool, error) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 || parts[0] == "" {
		return nil, false, fmt.Errorf("invalid empty path")
	}
	value, ok := sources[parts[0]]
	if !ok {
		return nil, false, nil
	}
	for _, part := range parts[1:] {
		if part == "" {
			return nil, false, fmt.Errorf("invalid path %q: empty segment", path)
		}
		next, found, err := resolveStaticInputSegment(value, part)
		if err != nil {
			return nil, false, fmt.Errorf("resolve %q in path %q: %w", part, path, err)
		}
		if !found {
			return nil, false, nil
		}
		value = next
	}
	if isNilStaticInputValue(value) {
		return nil, false, nil
	}
	return value, true, nil
}

func resolveStaticInputSegment(source any, segment string) (any, bool, error) {
	value := reflect.ValueOf(source)
	value = derefStaticInputValue(value)
	if !value.IsValid() {
		return nil, false, nil
	}

	switch value.Kind() {
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return nil, false, fmt.Errorf("map key type must be string, got %s", value.Type().Key())
		}
		item := value.MapIndex(reflect.ValueOf(segment).Convert(value.Type().Key()))
		if !item.IsValid() {
			return nil, false, nil
		}
		return item.Interface(), true, nil
	case reflect.Struct:
		field, ok := staticInputStructField(value, segment)
		if !ok {
			return nil, false, nil
		}
		return field.Interface(), true, nil
	default:
		return nil, false, nil
	}
}

func derefStaticInputValue(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func staticInputStructField(value reflect.Value, segment string) (reflect.Value, bool) {
	valueType := value.Type()
	if field, ok := valueType.FieldByName(segment); ok && field.IsExported() {
		fieldValue := value.FieldByIndex(field.Index)
		if fieldValue.CanInterface() {
			return fieldValue, true
		}
	}

	for i := 0; i < valueType.NumField(); i++ {
		field := valueType.Field(i)
		if !field.IsExported() {
			continue
		}
		if staticInputFieldMatches(field, segment) {
			fieldValue := value.Field(i)
			if fieldValue.CanInterface() {
				return fieldValue, true
			}
		}
	}
	return reflect.Value{}, false
}

func staticInputFieldMatches(field reflect.StructField, segment string) bool {
	if lowerFirst(field.Name) == segment {
		return true
	}
	for _, tag := range []string{"json", "toml", "yaml"} {
		if tagName(field.Tag.Get(tag)) == segment {
			return true
		}
	}
	return false
}

func tagName(tag string) string {
	name, _, _ := strings.Cut(tag, ",")
	if name == "-" {
		return ""
	}
	return name
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func isNilStaticInputValue(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func applyStaticInputTransform(value any, transform string) (any, error) {
	switch transform {
	case "":
		return value, nil
	case "bigint":
		return staticInputBigInt(value)
	case "string":
		return staticInputString(value), nil
	case "isCustomGasTokenEnabled":
		return staticInputCustomGasTokenEnabled(value)
	default:
		return nil, fmt.Errorf("unknown transform %q", transform)
	}
}

func staticInputBigInt(value any) (*big.Int, error) {
	switch v := value.(type) {
	case *big.Int:
		if v == nil {
			return nil, fmt.Errorf("nil bigint")
		}
		return new(big.Int).Set(v), nil
	case big.Int:
		return new(big.Int).Set(&v), nil
	case common.Hash:
		return v.Big(), nil
	case []byte:
		return new(big.Int).SetBytes(v), nil
	case json.Number:
		return parseStaticInputBigIntString(v.String())
	case string:
		return parseStaticInputBigIntString(v)
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := rv.Int()
		if n < 0 {
			return nil, fmt.Errorf("negative integer %d cannot be converted to bigint", n)
		}
		return new(big.Int).SetUint64(uint64(n)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return new(big.Int).SetUint64(rv.Uint()), nil
	default:
		return nil, fmt.Errorf("cannot convert %T to bigint", value)
	}
}

func parseStaticInputBigIntString(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty integer string")
	}
	n, ok := new(big.Int).SetString(s, 0)
	if !ok {
		return nil, fmt.Errorf("invalid integer string %q", s)
	}
	if n.Sign() < 0 {
		return nil, fmt.Errorf("negative integer %s cannot be converted to bigint", s)
	}
	return n, nil
}

func staticInputString(value any) string {
	if stringer, ok := value.(fmt.Stringer); ok {
		return stringer.String()
	}
	return fmt.Sprint(value)
}

func staticInputCustomGasTokenEnabled(value any) (bool, error) {
	if enabled, ok, err := callStaticInputBoolMethod(value, "IsCustomGasTokenEnabled"); ok || err != nil {
		return enabled, err
	}

	customGasToken, ok, err := resolveStaticInputSegment(value, "customGasToken")
	if err != nil {
		return false, err
	}
	if !ok {
		customGasToken, ok, err = resolveStaticInputSegment(value, "CustomGasToken")
		if err != nil {
			return false, err
		}
	}
	if !ok {
		return false, nil
	}

	name, _ := staticInputStringField(customGasToken, "name", "Name")
	symbol, _ := staticInputStringField(customGasToken, "symbol", "Symbol")
	return name != "" && symbol != "", nil
}

func callStaticInputBoolMethod(value any, methodName string) (bool, bool, error) {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return false, false, nil
	}
	method := rv.MethodByName(methodName)
	if !method.IsValid() && rv.Kind() != reflect.Pointer && rv.CanAddr() {
		method = rv.Addr().MethodByName(methodName)
	}
	if !method.IsValid() {
		return false, false, nil
	}
	methodType := method.Type()
	if methodType.NumIn() != 0 || methodType.NumOut() != 1 || methodType.Out(0).Kind() != reflect.Bool {
		return false, true, fmt.Errorf("method %s must have signature func() bool", methodName)
	}
	out := method.Call(nil)
	return out[0].Bool(), true, nil
}

func staticInputStringField(value any, names ...string) (string, bool) {
	for _, name := range names {
		field, ok, err := resolveStaticInputSegment(value, name)
		if err != nil || !ok || isNilStaticInputValue(field) {
			continue
		}
		return staticInputString(field), true
	}
	return "", false
}
