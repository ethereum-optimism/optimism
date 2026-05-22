package opcm

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"gopkg.in/yaml.v3"
)

const LegacyInputMappingKind = "legacy-input-mapping"

type StaticInputMapping struct {
	Version int                        `yaml:"version"`
	Kind    string                     `yaml:"kind"`
	Script  StaticInputMappingScript   `yaml:"script"`
	Input   map[string]StaticInputExpr `yaml:"input"`
}

type StaticInputMappingScript struct {
	Artifact string `yaml:"artifact"`
	Contract string `yaml:"contract"`
	Function string `yaml:"function"`
}

type StaticInputExpr struct {
	From      string            `yaml:"from,omitempty"`
	Value     any               `yaml:"value,omitempty"`
	Coalesce  []StaticInputExpr `yaml:"coalesce,omitempty"`
	Transform string            `yaml:"transform,omitempty"`
}

func LoadStaticInputMappingYAMLFile(path string) (*StaticInputMapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read static input mapping %s: %w", path, err)
	}
	mapping, err := LoadStaticInputMappingYAML(data)
	if err != nil {
		return nil, fmt.Errorf("load static input mapping %s: %w", path, err)
	}
	return mapping, nil
}

func LoadStaticInputMappingYAML(data []byte) (*StaticInputMapping, error) {
	var mapping StaticInputMapping
	if err := yaml.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("parse static input mapping: %w", err)
	}
	if mapping.Version == 0 {
		return nil, fmt.Errorf("static input mapping version is required")
	}
	if mapping.Kind != LegacyInputMappingKind {
		return nil, fmt.Errorf("unsupported static input mapping kind %q", mapping.Kind)
	}
	if mapping.Script.Artifact == "" {
		return nil, fmt.Errorf("static input mapping script.artifact is required")
	}
	if mapping.Script.Contract == "" {
		return nil, fmt.Errorf("static input mapping script.contract is required")
	}
	if mapping.Script.Function == "" {
		mapping.Script.Function = "run"
	}
	if len(mapping.Input) == 0 {
		return nil, fmt.Errorf("static input mapping input must not be empty")
	}
	for field, expr := range mapping.Input {
		if err := validateStaticInputExpr(expr); err != nil {
			return nil, fmt.Errorf("invalid mapping for %q: %w", field, err)
		}
	}
	return &mapping, nil
}

func ValidateStaticInputMapping(contractABI abi.ABI, mapping StaticInputMapping) error {
	methodName := mapping.Script.Function
	if methodName == "" {
		methodName = "run"
	}
	method, ok := contractABI.Methods[methodName]
	if !ok {
		return fmt.Errorf("method %q not found in ABI", methodName)
	}
	if len(method.Inputs) != 1 || method.Inputs[0].Type.T != abi.TupleTy {
		return fmt.Errorf("method %q must accept exactly one tuple input", methodName)
	}
	return ValidateStaticInputMappingForType(method.Inputs[0].Type, mapping)
}

func ValidateStaticInputMappingForType(inputType abi.Type, mapping StaticInputMapping) error {
	if inputType.T != abi.TupleTy {
		return fmt.Errorf("input type must be a tuple, got %s", inputType.String())
	}

	abiFields := make(map[string]struct{}, len(inputType.TupleRawNames))
	for _, name := range inputType.TupleRawNames {
		abiFields[name] = struct{}{}
		if _, ok := mapping.Input[name]; !ok {
			return fmt.Errorf("static input mapping is missing ABI input %q", name)
		}
	}
	for name := range mapping.Input {
		if _, ok := abiFields[name]; !ok {
			return fmt.Errorf("static input mapping targets unknown ABI input %q", name)
		}
	}
	return nil
}

func validateStaticInputExpr(expr StaticInputExpr) error {
	sources := 0
	if expr.From != "" {
		sources++
	}
	if expr.Value != nil {
		sources++
	}
	if len(expr.Coalesce) > 0 {
		sources++
	}
	if sources != 1 {
		return fmt.Errorf("exactly one of from, value, or coalesce is required")
	}
	for i, child := range expr.Coalesce {
		if err := validateStaticInputExpr(child); err != nil {
			return fmt.Errorf("invalid coalesce[%d]: %w", i, err)
		}
	}
	return nil
}
