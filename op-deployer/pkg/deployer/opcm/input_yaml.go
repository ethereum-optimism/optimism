package opcm

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"gopkg.in/yaml.v3"
)

const (
	scriptInputEnvelopeInputField   = "input"
	scriptInputEnvelopeVersionField = "version"
)

// IsNativeScriptInputYAML reports whether a YAML document uses the v2 native
// input envelope. Presence of the top-level input field is the signal.
func IsNativeScriptInputYAML(data []byte) (bool, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return false, fmt.Errorf("parse YAML input: %w", err)
	}
	root := yamlDocumentRoot(&doc)
	if root.Kind != yaml.MappingNode {
		return false, nil
	}
	return yamlMappingHasKey(root, scriptInputEnvelopeInputField), nil
}

// LoadScriptInputYAMLFileFromArtifact loads ABI-shaped YAML input using a Foundry artifact.
func LoadScriptInputYAMLFileFromArtifact(artifact *foundry.Artifact, methodName string, path string) (ScriptInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read input file %s: %w", path, err)
	}
	input, err := LoadScriptInputYAMLFromArtifact(artifact, methodName, data)
	if err != nil {
		return nil, fmt.Errorf("load input file %s: %w", path, err)
	}
	return input, nil
}

// LoadScriptInputYAMLFromArtifact loads ABI-shaped YAML input using a Foundry artifact.
func LoadScriptInputYAMLFromArtifact(artifact *foundry.Artifact, methodName string, data []byte) (ScriptInput, error) {
	if artifact == nil {
		return nil, fmt.Errorf("artifact is nil")
	}
	return LoadScriptInputYAML(artifact.ABI, methodName, data)
}

// LoadScriptInputYAMLFile loads ABI-shaped YAML input for a script method.
func LoadScriptInputYAMLFile(contractABI abi.ABI, methodName string, path string) (ScriptInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read input file %s: %w", path, err)
	}
	input, err := LoadScriptInputYAML(contractABI, methodName, data)
	if err != nil {
		return nil, fmt.Errorf("load input file %s: %w", path, err)
	}
	return input, nil
}

// LoadScriptInputYAML loads ABI-shaped YAML input for a script method.
func LoadScriptInputYAML(contractABI abi.ABI, methodName string, data []byte) (ScriptInput, error) {
	method, ok := contractABI.Methods[methodName]
	if !ok {
		return nil, fmt.Errorf("method %q not found in ABI", methodName)
	}
	if len(method.Inputs) != 1 || method.Inputs[0].Type.T != abi.TupleTy {
		return nil, fmt.Errorf("method %q must accept exactly one tuple input", methodName)
	}
	return LoadScriptInputYAMLForType(method.Inputs[0].Type, data)
}

// LoadScriptInputYAMLForType loads ABI-shaped YAML input for a tuple ABI type.
func LoadScriptInputYAMLForType(inputType abi.Type, data []byte) (ScriptInput, error) {
	if inputType.T != abi.TupleTy {
		return nil, fmt.Errorf("input type must be a tuple, got %s", inputType.String())
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse YAML input: %w", err)
	}
	raw, err := yamlNodeToValue(&doc)
	if err != nil {
		return nil, err
	}
	input, ok := raw.(ScriptInput)
	if !ok {
		return nil, fmt.Errorf("YAML input must be a mapping")
	}
	input, err = unwrapInputEnvelope(input)
	if err != nil {
		return nil, err
	}

	if _, err := mapToTupleValue(inputType, input); err != nil {
		return nil, err
	}
	return input, nil
}

func unwrapInputEnvelope(input ScriptInput) (ScriptInput, error) {
	raw, ok := input[scriptInputEnvelopeInputField]
	if !ok {
		return input, nil
	}

	for key := range input {
		if key != scriptInputEnvelopeInputField && key != scriptInputEnvelopeVersionField {
			return nil, fmt.Errorf("v2 input envelope supports only %q and optional %q fields, got %q", scriptInputEnvelopeInputField, scriptInputEnvelopeVersionField, key)
		}
	}

	nested, ok := raw.(ScriptInput)
	if !ok {
		return nil, fmt.Errorf("v2 input envelope field %q must be a mapping", scriptInputEnvelopeInputField)
	}
	return nested, nil
}

func yamlNodeToValue(node *yaml.Node) (any, error) {
	if node == nil {
		return nil, nil
	}
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return ScriptInput{}, nil
		}
		return yamlNodeToValue(node.Content[0])
	case yaml.MappingNode:
		out := make(ScriptInput, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			if keyNode.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("YAML mapping key at line %d must be a scalar", keyNode.Line)
			}
			key := keyNode.Value
			if _, ok := out[key]; ok {
				return nil, fmt.Errorf("duplicate YAML input field %q at line %d", key, keyNode.Line)
			}
			value, err := yamlNodeToValue(node.Content[i+1])
			if err != nil {
				return nil, err
			}
			out[key] = value
		}
		return out, nil
	case yaml.SequenceNode:
		out := make([]any, len(node.Content))
		for i, item := range node.Content {
			value, err := yamlNodeToValue(item)
			if err != nil {
				return nil, err
			}
			out[i] = value
		}
		return out, nil
	case yaml.ScalarNode:
		return yamlScalarToValue(node)
	case yaml.AliasNode:
		return nil, fmt.Errorf("YAML aliases are not supported at line %d", node.Line)
	default:
		return nil, fmt.Errorf("unsupported YAML node kind %d at line %d", node.Kind, node.Line)
	}
}

func yamlDocumentRoot(node *yaml.Node) *yaml.Node {
	if node.Kind != yaml.DocumentNode {
		return node
	}
	if len(node.Content) == 0 {
		return &yaml.Node{Kind: yaml.MappingNode}
	}
	return node.Content[0]
}

func yamlMappingHasKey(node *yaml.Node, key string) bool {
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Kind == yaml.ScalarNode && node.Content[i].Value == key {
			return true
		}
	}
	return false
}

func yamlScalarToValue(node *yaml.Node) (any, error) {
	switch node.Tag {
	case "!!null":
		return nil, nil
	case "!!bool":
		v, err := strconv.ParseBool(node.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid YAML bool %q at line %d: %w", node.Value, node.Line, err)
		}
		return v, nil
	case "!!int", "!!float", "!!str", "":
		return node.Value, nil
	default:
		return node.Value, nil
	}
}
