package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ... Add more structs as needed for other node types
var builtInTypes = map[string]bool{
	"address": true, "address payable": true, "bool": true, "string": true, "bytes": true,
	"int": true, "int8": true, "int16": true, "int24": true, "int32": true,
	"int40": true, "int48": true, "int56": true, "int64": true, "int72": true,
	"int80": true, "int88": true, "int96": true, "int104": true, "int112": true,
	"int120": true, "int128": true, "int136": true, "int144": true, "int152": true,
	"int160": true, "int168": true, "int176": true, "int184": true, "int192": true,
	"int200": true, "int208": true, "int216": true, "int224": true, "int232": true,
	"int240": true, "int248": true, "int256": true,
	"uint": true, "uint8": true, "uint16": true, "uint24": true, "uint32": true,
	"uint40": true, "uint48": true, "uint56": true, "uint64": true, "uint72": true,
	"uint80": true, "uint88": true, "uint96": true, "uint104": true, "uint112": true,
	"uint120": true, "uint128": true, "uint136": true, "uint144": true, "uint152": true,
	"uint160": true, "uint168": true, "uint176": true, "uint184": true, "uint192": true,
	"uint200": true, "uint208": true, "uint216": true, "uint224": true, "uint232": true,
	"uint240": true, "uint248": true, "uint256": true,
	"bytes1": true, "bytes2": true, "bytes3": true, "bytes4": true, "bytes5": true,
	"bytes6": true, "bytes7": true, "bytes8": true, "bytes9": true, "bytes10": true,
	"bytes11": true, "bytes12": true, "bytes13": true, "bytes14": true, "bytes15": true,
	"bytes16": true, "bytes17": true, "bytes18": true, "bytes19": true, "bytes20": true,
	"bytes21": true, "bytes22": true, "bytes23": true, "bytes24": true, "bytes25": true,
	"bytes26": true, "bytes27": true, "bytes28": true, "bytes29": true, "bytes30": true,
	"bytes31": true, "bytes32": true,
}
var excludedFiles = map[string]bool{
	// Interface for this not needed
	"PreimageKeyLib": true,
	// Interface for this not needed
	"WETH": true,
	// Interface for this not needed
	"OPContractsManager": true,
	// Interface for this not needed
	"OPContractsManagerInterop": true,
	// returns StandardBridge but interface expects IStandardBridge
	"L2StandardBridge": true,
	// returns StandardBridge but interface expects IStandardBridge
	"L2StandardBridgeInterop": true,
}
var mainName = map[string]string{}
var importedNames = map[string]string{}
var overriddenFunctions = map[string]bool{}
var eventSelectors = map[string]bool{}
var JSON_OUTPUT JsonOutput
var contractName string
var constructorFound bool = false
var level int = 0

func main() {
	contractsBase, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting working directory:", err)
		os.Exit(1)
	}

	// Find contract files
	l1ContractFiles, err := findContractFiles(filepath.Join(contractsBase, "src", "L1"), "interfaces")
	if err != nil {
		fmt.Println("Error finding L1 contract files:", err)
	}
	l2ContractFiles, err := findContractFiles(filepath.Join(contractsBase, "src", "L2"), "interfaces")
	if err != nil {
		fmt.Println("Error finding L2 contract files:", err)
	}
	cannonContractFiles, err := findContractFiles(filepath.Join(contractsBase, "src", "cannon"), "interfaces", "libraries")
	if err != nil {
		fmt.Println("Error finding cannon contract files:", err)
	}
	disputeContractFiles, err := findContractFiles(filepath.Join(contractsBase, "src", "dispute"), "interfaces", "lib")
	if err != nil {
		fmt.Println("Error finding dispute contract files:", err)
	}
	governanceContractFiles, err := findContractFiles(filepath.Join(contractsBase, "src", "governance"), "interfaces")
	if err != nil {
		fmt.Println("Error finding governance contract files:", err)
	}

	// Create interface directories
	interfaceDir := filepath.Join(contractsBase, "src", "interfaces")
	err = createDirs(interfaceDir, "L1", "L2", "cannon", "dispute", "governance")
	if err != nil {
		fmt.Println("Error creating directories:", err)
		os.Exit(1)
	}

	// Combine all contract files into a single list
	all := make([]string, 0, len(l1ContractFiles)+len(l2ContractFiles)+len(cannonContractFiles)+len(disputeContractFiles)+len(governanceContractFiles))
	all = append(all, l1ContractFiles...)
	all = append(all, l2ContractFiles...)
	all = append(all, cannonContractFiles...)
	all = append(all, disputeContractFiles...)
	all = append(all, governanceContractFiles...)

	// Generate interfaces for each contract file
	for _, contractFile := range all {
		contractName = strings.Split(filepath.Base(contractFile), ".")[0]
		if excludedFiles[contractName] {
			continue
		}

		fmt.Println("Generating interfaces for contract: " + contractFile)
		_interface, err := run()
		if err != nil {
			fmt.Println("Error generating interfaces for", contractName, err)
			resetVars()
			continue
		}
		parentDir := filepath.Base(filepath.Dir(contractFile))
		if err := os.WriteFile(filepath.Join(contractsBase, "src", "interfaces", parentDir, "I"+contractName+".sol"), []byte(_interface), 0o644); err != nil {
			fmt.Println("Error writing file:", err)
		}
		resetVars()
	}

	// contractName = "OptimismSuperchainERC20Factory"
	// _interface, err := run()
	// if err != nil {
	//  panic(err)
	// }
	// fmt.Println(_interface)
}

func run() (string, error) {
	var INTERFACE string
	var err error

	JSON_OUTPUT, err = getJsonOutput(contractName)
	if err != nil {
		return "", err
	}

	ast := JSON_OUTPUT.AST
	INTERFACE = "// SPDX-License-Identifier: " + ast.License + "\n"

	for _, node := range ast.Nodes {
		var temp string
		temp, err = parseNode(node)
		if err != nil {
			return "", err
		}
		INTERFACE += temp
	}

	if level != 0 {
		return "", fmt.Errorf("invariant: level is not 0")
	}

	return INTERFACE, nil
}

func parseNode(node Node) (string, error) {
	var result string
	var err error

	switch node.NodeType {
	case "PragmaDirective":
		return parsePragmaDirective(node)
	case "ImportDirective":
		return parseImportDirective(node)
	case "ContractDefinition":
		return parseContractDefinition(node)
	case "EnumDefinition":
		return parseEnumDefinition(node)
	case "StructDefinition":
		return parseStructDefinition(node)
	case "FunctionDefinition":
		return parseFunctionDefinition(node)
	case "VariableDeclaration":
		return parseVariableDeclaration(node)
	case "EventDefinition":
		return parseEventDefinition(node)
	case "ErrorDefinition":
		return parseErrorDefinition(node)
	case "UserDefinedValueTypeDefinition":
		return parseUserDefinedValueTypeDefinition(node)
	}

	return result, err
}

func parseUserDefinedValueTypeDefinition(node Node) (string, error) {
	var result string = "type " + node.Name + " is " + node.UnderlyingType.Name + ";\n"
	return result, nil
}

func parseContractDefinition(node Node) (string, error) {
	var result string = "interface " + "I" + node.Name
	var err error

	result += " {\n"

	imports, temp, err := processInheritance(node)
	if err != nil {
		return "", err
	}
	result = imports + result + temp

	for _, _node := range node.Nodes {
		var temp string
		temp, err = parseNode(_node)
		if err != nil {
			return "", err
		}
		result += temp
	}

	if !constructorFound && level == 0 {
		result += "function __constructor__() external;\n"
	}

	result += "}\n"

	return result, err
}

func processInheritance(node Node) (string, string, error) {
	level++
	defer func() { level-- }()

	var imports string
	var result string
	var err error

	if node.NodeType == "ContractDefinition" && len(node.BaseContracts) > 0 {
		for _, baseContract := range node.BaseContracts {
			var name string = mainName[baseContract.BaseName.Name]
			if name == "" {
				name = baseContract.BaseName.Name
			}
			if name == "ISemver" || name == "IWETH" {
				continue
			}

			jsonOutput, err := getJsonOutput(name)
			if err != nil {
				return "", "", err
			}

			for _, _node := range jsonOutput.AST.Nodes {
				if _node.NodeType == "ContractDefinition" && _node.Name == name {
					temp1, temp2, err := processInheritance(_node)
					if err != nil {
						return "", "", err
					}
					imports += temp1
					result += temp2

					for _, __node := range _node.Nodes {
						temp2, err = parseNode(__node)
						if err != nil {
							return "", "", err
						}
						result += temp2
					}
				} else if _node.NodeType == "ImportDirective" {
					temp, err := parseImportDirective(_node)
					if err != nil {
						return "", "", err
					}
					imports += temp
				}
			}
		}
	}

	return imports, result, err
}

func parseImportDirective(node Node) (string, error) {
	var result string = "import "
	var err error

	if strings.HasPrefix(node.AbsolutePath, "lib") && node.AbsolutePath != "lib/lib-keccak/contracts/lib/LibKeccak.sol" {
		return "", nil
	}

	if len(node.SymbolAliases) > 0 {
		result += "{ "
		for i, symbolAlias := range node.SymbolAliases {
			if symbolAlias.Foreign.Name == "I"+contractName || symbolAlias.Local == "I"+contractName {
				if len(node.SymbolAliases) > 1 {
					continue
				} else {
					result = strings.TrimSuffix(result, "import { ")
					break
				}
			}
			result += symbolAlias.Foreign.Name
			if symbolAlias.Local != "" {
				result += " as " + symbolAlias.Local
				mainName[symbolAlias.Local] = symbolAlias.Foreign.Name
				importedNames[symbolAlias.Local] = node.AbsolutePath
			} else {
				mainName[symbolAlias.Foreign.Name] = symbolAlias.Foreign.Name
				importedNames[symbolAlias.Foreign.Name] = node.AbsolutePath
			}

			if i != len(node.SymbolAliases)-1 {
				result += ", "
			} else {
				result += " } from \"" + node.AbsolutePath + "\";\n"
			}
		}
	} else {
		result += "\"" + node.AbsolutePath + "\";\n"
	}

	return result, err
}

func parsePragmaDirective(_ Node) (string, error) {
	var result string = "pragma solidity ^0.8.0;\n"
	return result, nil
}

func parseEnumDefinition(node Node) (string, error) {
	var result string = "enum " + node.Name + " {\n"
	var err error

	for i, member := range node.Members {
		if i == len(node.Members)-1 {
			result += member.Name + "\n"
		} else {
			result += member.Name + ",\n"
		}
	}

	result += "}\n"

	return result, err
}

func parseStructDefinition(node Node) (string, error) {
	var result string = "struct " + node.Name + " {\n"
	var err error

	for _, member := range node.Members {
		temp, err := processType(member.TypeDescriptions.TypeString)
		if err != nil {
			return "", err
		}
		result += temp + " " + member.Name + ";\n"
	}

	result += "}\n"

	return result, err
}

func parseEventDefinition(node Node) (string, error) {
	var result string = "event " + node.Name + "("
	var err error

	if eventSelectors[node.EventSelector] {
		return "", nil
	}

	for i, parameter := range node.Parameters.Parameters {
		temp, err := processType(parameter.TypeDescriptions.TypeString)
		if err != nil {
			return "", err
		}
		var name string = ifNotNullPrefixWhitespace(parameter.Name)
		if parameter.Indexed {
			name = " indexed" + name
		}
		if i == len(node.Parameters.Parameters)-1 {
			result += temp + name
		} else {
			result += temp + name + ", "
		}
	}

	result += ");\n"
	eventSelectors[node.EventSelector] = true

	return result, err
}

func parseErrorDefinition(node Node) (string, error) {
	var result string = "error " + node.Name + "("
	var err error

	for i, parameter := range node.Parameters.Parameters {
		temp, err := processType(parameter.TypeDescriptions.TypeString)
		if err != nil {
			return "", err
		}
		if i == len(node.Parameters.Parameters)-1 {
			result += temp + " " + parameter.Name
		} else {
			result += temp + " " + parameter.Name + ", "
		}
	}

	result += ");\n"

	return result, err
}

func parseFunctionDefinition(node Node) (string, error) {
	switch node.Visibility {
	case "internal", "private":
		return "", nil
	}

	if overriddenFunctions[node.FunctionSelector] {
		return "", nil
	}

	var result string
	switch node.Kind {
	case "constructor":
		{
			if level != 0 {
				return "", nil
			}

			result = "function __constructor__("
			constructorFound = true
		}
	case "function":
		result = "function " + node.Name + "("
	case "receive":
		{
			if overriddenFunctions["receive"] {
				return "", nil
			}
			overriddenFunctions["receive"] = true
			result = "receive("
		}
	case "fallback":
		{
			if overriddenFunctions["fallback"] {
				return "", nil
			}
			overriddenFunctions["fallback"] = true
			result = "fallback("
		}
	default:
		return "", fmt.Errorf("unknown function kind: %s", node.Kind)
	}
	var err error

	for i, parameter := range node.Parameters.Parameters {
		temp, err := processType(parameter.TypeDescriptions.TypeString)
		if err != nil {
			return "", err
		}

		storageLocation, err := getStorageLocationFromFunctionParameterNode(parameter)
		if err != nil {
			return "", err
		}

		var name string = ifNotNullPrefixWhitespace(parameter.Name)
		if i == len(node.Parameters.Parameters)-1 {
			result += temp + storageLocation + name
		} else {
			result += temp + storageLocation + name + ", "
		}
	}
	result += ") external"

	switch node.StateMutability {
	case "payable":
		result += " payable"
	case "view":
		result += " view"
	case "pure":
		result += " pure"
	}

	if len(node.ReturnParameters.Parameters) > 0 {
		result += " returns ("
		for i, parameter := range node.ReturnParameters.Parameters {
			temp, err := processType(parameter.TypeDescriptions.TypeString)
			if err != nil {
				return "", err
			}

			storageLocation, err := getStorageLocationFromFunctionParameterNode(parameter)
			if err != nil {
				return "", err
			}

			var name string = ifNotNullPrefixWhitespace(parameter.Name)
			if i == len(node.ReturnParameters.Parameters)-1 {
				result += temp + storageLocation + name
			} else {
				result += temp + storageLocation + name + ", "
			}
		}
		result += ")"
	}

	result += ";\n"
	if node.FunctionSelector != "" {
		overriddenFunctions[node.FunctionSelector] = true
	}

	return result, err
}

func parseVariableDeclaration(node Node) (string, error) {
	if node.Visibility != "public" {
		return "", nil
	}

	var result string
	var err error

	if overriddenFunctions[node.FunctionSelector] {
		return "", nil
	}

	switch node.TypeName.NodeType {
	case "UserDefinedTypeName":
		{
			temp, err := processValueTypeReturnParameters(node)
			if err != nil {
				return "", err
			}
			result = "function " + node.Name + "() external view returns (" + temp + ");\n"
		}
	case "ElementaryTypeName":
		{
			temp, err := processValueTypeReturnParameters(node)
			if err != nil {
				return "", err
			}
			result = "function " + node.Name + "() external view returns (" + temp + ");\n"
		}
	case "ArrayTypeName":
		{
			var nestedAmount int = strings.Count(node.TypeDescriptions.TypeString, "[")
			result = "function " + node.Name + "("
			for i := 0; i < nestedAmount; i++ {
				result += "uint256"
				if i != nestedAmount-1 {
					result += ", "
				}
			}

			temp, err := processArrayTypeReturnParameters(node)
			if err != nil {
				return "", err
			}

			result += ") external view returns (" + temp + ");\n"
		}
	case "Mapping":
		{
			result = "function " + node.Name + "("

			var inputTypes []string
			var valueTypeNode = node.TypeName
			for {
				keyType, err := processType(valueTypeNode.KeyType.TypeDescriptions.TypeString)
				if err != nil {
					return "", err
				}

				if valueTypeNode.ValueType.NodeType == "Mapping" {
					var name string = ifNotNullPrefixWhitespace(valueTypeNode.KeyName)
					var storageLocation string = getStorageLocationFromVariableDeclarationType(valueTypeNode.KeyType.TypeDescriptions.TypeString)
					result += keyType + storageLocation + name + ", "
					inputTypes = append(inputTypes, keyType)

					valueTypeNode = valueTypeNode.ValueType
				} else if valueTypeNode.ValueType.NodeType == "ArrayTypeName" {
					var name string = ifNotNullPrefixWhitespace(valueTypeNode.KeyName)
					var storageLocation string = getStorageLocationFromVariableDeclarationType(valueTypeNode.KeyType.TypeDescriptions.TypeString)
					result += keyType + storageLocation + name + ", "
					inputTypes = append(inputTypes, keyType)

					var nestedAmount int = 1
					var current = valueTypeNode.ValueType.BaseType
					for {
						if current.NodeType == "ArrayTypeName" {
							nestedAmount++
							current = current.BaseType
						} else {
							break
						}
					}
					for i := 0; i < nestedAmount; i++ {
						result += "uint256"
						inputTypes = append(inputTypes, "uint256")

						if i != nestedAmount-1 {
							result += ", "
						}
					}
					break
				} else {
					var name string = ifNotNullPrefixWhitespace(valueTypeNode.KeyName)
					var storageLocation string = getStorageLocationFromVariableDeclarationType(valueTypeNode.KeyType.TypeDescriptions.TypeString)
					result += keyType + storageLocation + name
					inputTypes = append(inputTypes, keyType)
					break
				}
			}
			var returnParameters string
			returnParameters, err = processMappingTypeReturnParameters(node, inputTypes)
			if err != nil {
				return "", err
			}

			result += ") external view returns (" + returnParameters + ");\n"
		}
	default:
		return "", fmt.Errorf("unknown node type: %s", node.TypeName.NodeType)
	}

	if node.FunctionSelector != "" {
		overriddenFunctions[node.FunctionSelector] = true
	}

	return result, err
}
