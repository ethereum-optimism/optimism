package main

// Imports
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Types
type TypeObject struct {
	Name         string       `json:"name"`
	Type         string       `json:"type"`
	InternalType string       `json:"internalType"`
	Components   []TypeObject `json:"components"`
}

type ABI struct {
	Type            string       `json:"type"`
	StateMutability string       `json:"stateMutability"`
	Name            string       `json:"name"`
	Inputs          []TypeObject `json:"inputs"`
	Outputs         []TypeObject `json:"outputs"`
}

type Foreign struct {
	Name string `json:"name"`
}

type SymbolAliases struct {
	Foreign Foreign `json:"foreign"`
	Local   string  `json:"local"`
}

type Nodes struct {
	Name          string          `json:"name"`
	NodeType      string          `json:"nodeType"`
	CanonicalName string          `json:"canonicalName"`
	AbsolutePath  string          `json:"absolutePath"`
	SymbolAliases []SymbolAliases `json:"symbolAliases"`
	Nodes         []Nodes         `json:"nodes"`
	Members       []Members       `json:"members"`
	BaseContracts []BaseContracts `json:"baseContracts"`
}

type BaseContracts struct {
	BaseName BaseName `json:"baseName"`
}

type BaseName struct {
	Name string `json:"name"`
}

type Members struct {
	Name     string     `json:"name"`
	TypeName ObjectType `json:"typeName"`
}

type ObjectType struct {
	Name string `json:"name"`
}

type AST struct {
	Nodes []Nodes `json:"nodes"`
}

type JsonOutput struct {
	Abi []ABI `json:"abi"`
	Ast AST   `json:"ast"`
}

type TypeOwner string

const (
	Imported     TypeOwner = "Imported"
	ThisContract TypeOwner = "ThisContract"
	ThisFile     TypeOwner = "ThisFile"
)

// Global Constants and Variables
const (
	ARTIFACTS_DIR string = "forge-artifacts/"
)

var (
	builtInTypes = map[string]bool{
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
	detectedStructs = make(map[string]bool)
	detectedEnums   = make(map[string]bool)
	detectedImports = make(map[string]bool)
	explicitImports = make(map[string][]string)
	detectedTypes   = make(map[string]bool)

	constructorFound bool = false
	jsonOutput       JsonOutput
	contractName     string
	FUNCTIONS        string
	EVENTS           string
	RECEIVE          string
	FALLBACK         string
	CONSTRUCTOR      string
	ERRORS           string
	STRUCTS          string
	FILE_STRUCTS     string
	ENUMS            string
	FILE_ENUMS       string
	IMPORTS          string
	TYPES            string
	FILE_TYPES       string
)

// Functions

// Resets all global variables to the default value of their type
func resetVars() {
	detectedStructs = make(map[string]bool)
	detectedEnums = make(map[string]bool)
	detectedImports = make(map[string]bool)
	explicitImports = make(map[string][]string)
	detectedTypes = make(map[string]bool)
	constructorFound = false
	jsonOutput = JsonOutput{}
	contractName = ""
	FUNCTIONS = ""
	EVENTS = ""
	RECEIVE = ""
	FALLBACK = ""
	CONSTRUCTOR = ""
	ERRORS = ""
	STRUCTS = ""
	FILE_STRUCTS = ""
	ENUMS = ""
	FILE_ENUMS = ""
	IMPORTS = ""
	TYPES = ""
	FILE_TYPES = ""
}

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
		detectedImports[contractName] = true
		detectedImports["I"+contractName] = true

		fmt.Println("Generating interfaces for contract: " + contractFile)
		_interface, err := generateInterfaces()
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
}

func generateInterfaces() (string, error) {
	// Initialize the interface with the license, pragma, and interface definition
	var NEW_INTERFACE string = "// SPDX-License-Identifier: MIT\npragma solidity ^0.8.0;\n\n"
	var INTERFACE_DEFINITION string = "interface I" + contractName + " {\n"
	var err error

	// Get the JSON output for the contract
	jsonOutput, err = getJsonOutput(contractName)
	if err != nil {
		return "", err
	}

	// Check for loose imports, if any are found, return an error.
	unusedImports := checkForLooseImports(jsonOutput)
	if len(unusedImports) > 0 {
		err = fmt.Errorf("loose imports: %s", strings.Join(unusedImports, ", "))
		return "", err
	}

	// process the ABI
	for _, abiObj := range jsonOutput.Abi {
		var abiObjType string = abiObj.Type

		switch abiObjType {
		case "function":
			err = processFunction(abiObj)
		case "event":
			err = processEvent(abiObj)
		case "receive":
			processReceive()
		case "fallback":
			err = processFallback(abiObj)
		case "constructor":
			{
				if constructorFound {
					err = fmt.Errorf("multiple constructors found")
					return "", err
				}
				constructorFound = true
				err = processConstructor(abiObj)
			}
		case "error":
			err = processError(abiObj)
		default:
			{
				err = fmt.Errorf("invalid abiObjType, error: %s", abiObjType)
				return "", err
			}
		}
		if err != nil {
			err = fmt.Errorf("error processing abiObjType: %s", err.Error())
			return "", err
		}
	}

	if !constructorFound {
		CONSTRUCTOR = " function __constructor__() external;\n"
	}

	// Combine all sections into the final interface, giving line breaks where needed.
	writeImports()
	combineSections()
	NEW_INTERFACE += IMPORTS + FILE_TYPES + FILE_ENUMS + FILE_STRUCTS + INTERFACE_DEFINITION + TYPES + ENUMS + STRUCTS + ERRORS + EVENTS + CONSTRUCTOR + FUNCTIONS + FALLBACK + RECEIVE + "}"
	return NEW_INTERFACE, nil
}

func processFunction(abiObj ABI) error {
	var err error

	var name string = abiObj.Name
	var stateMutability string = handleStateMutability(abiObj.StateMutability)

	inputs, err := processParameters(abiObj.Inputs, false, true)
	if err != nil {
		return err
	}
	outputs, err := processParameters(abiObj.Outputs, true, true)
	if err != nil {
		return err
	}

	FUNCTIONS += "  function " + name + "(" + inputs + ") external" + stateMutability + outputs + ";\n"

	return err
}

func processEvent(abiObj ABI) error {
	var name string = abiObj.Name
	inputs, err := processParameters(abiObj.Inputs, false, false)
	if err != nil {
		return err
	}
	EVENTS += " event " + name + "(" + inputs + ");\n"

	return err
}

func processReceive() {
	RECEIVE += "    receive" + "() external payable;\n"
}

func processFallback(abiObj ABI) error {
	var stateMutability string = handleStateMutability(abiObj.StateMutability)
	inputs, err := processParameters(abiObj.Inputs, false, true)
	if err != nil {
		return err
	}
	FALLBACK += "   fallback" + "(" + inputs + ") external" + stateMutability + ";\n"

	return err
}

func processConstructor(abiObj ABI) error {
	var stateMutability string = handleStateMutability(abiObj.StateMutability)
	inputs, err := processParameters(abiObj.Inputs, false, true)
	if err != nil {
		return err
	}
	CONSTRUCTOR += "    function __constructor__" + "(" + inputs + ") external" + stateMutability + ";\n"

	return err
}

func processParameters(parameters []TypeObject, isOutput bool, isLocalVar bool) (string, error) {
	var err error = nil

	var stringParameters string = ""
	for i, parameter := range parameters {
		p, err := process_parameter_type(parameter)
		if err != nil {
			return "", err
		}

		// if its a local variable, add the memory annotation if needed
		if isLocalVar {
			if shouldAddMemoryAnnotation(parameter.Type) {
				p += " memory"
			}
		}
		stringParameters += p
		if parameter.Name != "" {
			stringParameters += " " + parameter.Name
		}
		if i != (len(parameters) - 1) {
			stringParameters += ", "
		}
	}

	// if its an output parameter, add the returns keyword
	if isOutput && stringParameters != "" {
		stringParameters = " returns(" + stringParameters + ")"
	}

	return stringParameters, err
}

func process_parameter_type(parameters TypeObject) (string, error) {
	var err error

	internalType := parameters.InternalType
	_type := parameters.Type

	// Regex pattern to match struct, enum and contract types
	var pattern = regexp.MustCompile(`^(struct|enum|contract)\s+(.+)$`)

	var p string

	// Check for struct, enum, and contract types, if so, process them.
	var match = pattern.FindStringSubmatch(internalType)
	if match != nil {
		var v = strings.Split(internalType, " ")
		switch v[0] {
		case "enum":
			{
				err = processEnum(v[1])
				if err != nil {
					return "", err
				}
			}
		case "struct":
			{
				structs, structTypeOwner, err := processStruct(parameters, v[1])
				if err != nil {
					return "", err
				}
				if structTypeOwner == ThisContract {
					STRUCTS += structs
				} else if structTypeOwner == ThisFile {
					FILE_STRUCTS += structs
				}
			}
		case "contract":
			{
				if v[1] != contractName {
					err = processImport(v[1])
					if err != nil {
						return "", err
					}
				}
			}
		}
		p = v[1]
	} else {
		// If no match, then it's a built-in type or a user defined type
		p = internalType
	}

	// Check if (its defined within a contract, library or interface block) or (if its defined at the file level or is a built in type)
	if strings.Contains(p, ".") {
		var v = strings.Split(p, ".")
		if v[0] == contractName {
			if internalType == p {
				err = processType(v[1], _type, ThisContract)
				if err != nil {
					return "", err
				}
			}
			return v[1], err
		} else {
			err = processImport(v[0])
			if err != nil {
				return "", err
			}
			return p, err
		}
	} else {
		if match != nil {
			return p, err
		} else {
			// Check if p is a file-level defined user-defined type
			if isUserDefinedType(p) {
				err = processType(p, _type, ThisFile)
				if err != nil {
					return "", err
				}
			}
			// If not, it's a built-in type
			return p, err
		}
	}
}

func processError(abiObj ABI) error {
	var err error = processImport(abiObj.Name)
	if err == nil {
		return err
	}

	var name string = abiObj.Name
	inputs, err := processParameters(abiObj.Inputs, false, false)
	if err != nil {
		return err
	}
	ERRORS += " error " + name + "(" + inputs + ");\n"

	return err
}

func processStruct(parameter TypeObject, name string) (string, TypeOwner, error) {
	var err error
	var structure string = ""

	var typeOwner TypeOwner
	typeOwner, err = isImported(name)
	if err != nil {
		return "", typeOwner, err
	}
	if typeOwner == Imported {
		err = processImport(strings.Split(name, ".")[0])
		if err != nil {
			return "", typeOwner, err
		}
		return structure, typeOwner, err
	}
	var s = strings.Split(name, ".")
	name = s[len(s)-1]
	name = strings.Split(name, "[")[0]

	if detectedStructs[name] {
		return structure, typeOwner, err
	}

	// check imports for struct definition
	if typeOwner == ThisFile {
		var err = processImport(name)
		if err == nil {
			return "", typeOwner, err
		}
	}

	structure += "  struct " + name + " {\n"
	for _, component := range parameter.Components {
		p, err := process_parameter_type(component)
		if err != nil {
			return "", typeOwner, err
		}
		structure += "      " + p + " " + component.Name + ";\n"
	}
	structure += "  }\n"

	detectedStructs[name] = true

	return structure, typeOwner, err
}

func processEnum(name string) error {
	var err error

	var typeOwner TypeOwner
	typeOwner, err = isImported(name)
	if err != nil {
		return err
	}
	if typeOwner == Imported {
		err = processImport(strings.Split(name, ".")[0])
		if err != nil {
			return err
		}
		return err
	}

	var s = strings.Split(name, ".")
	name = s[len(s)-1]
	name = strings.Split(name, "[")[0]

	if detectedEnums[name] {
		return err
	}

	// Search the contract for the enum definition
	var node Nodes
	for _, _node := range jsonOutput.Ast.Nodes {
		if _node.NodeType == "ContractDefinition" && _node.CanonicalName == contractName {
			node = _node
			break
		}
	}
	var found bool
	for _, _node := range node.Nodes {
		if _node.NodeType == "EnumDefinition" && _node.Name == name {
			found = true
			err = writeEnum(name, _node, typeOwner)
			if err != nil {
				return err
			}
			break
		}
	}

	if !found {
		// if not found, try to see if its imported from another file
		err = processImport(name)
		if err != nil {
			// if it is not then it must be a file-level enum
			for _, _node := range jsonOutput.Ast.Nodes {
				if _node.NodeType == "EnumDefinition" && _node.Name == name {
					found = true
					err = writeEnum(name, _node, typeOwner)
					if err != nil {
						return err
					}
					break
				}
			}
			if !found {
				err = fmt.Errorf("enum not found: %s", name)
				return err
			}
		}
	}

	detectedEnums[name] = true

	return err
}

func writeEnum(name string, _node Nodes, typeOwner TypeOwner) error {
	var err error

	var enums string = ""
	enums += "  enum " + name + " {\n"
	for i, variant := range _node.Members {
		enums += "      " + variant.Name
		if i != (len(_node.Members) - 1) {
			enums += ",\n"
		} else {
			enums += "\n"
		}
	}
	enums += "  }\n"

	if typeOwner == ThisContract {
		ENUMS += enums
	} else if typeOwner == ThisFile {
		FILE_ENUMS += enums
	} else {
		err = fmt.Errorf("unreachable:invalid type owner: %v", typeOwner)
	}

	return err
}

func processImport(name string) error {
	name = strings.Split(name, "[")[0]
	return processImportWith(name, jsonOutput, contractName)
}

func processImportWith(name string, newJsonOutput JsonOutput, __contractName string) error {
	var err error = nil

	if detectedImports[name] {
		return err
	}

	var from string
	for _, node := range newJsonOutput.Ast.Nodes {
		if node.NodeType == "ImportDirective" {
			for _, symbolAlias := range node.SymbolAliases {
				if symbolAlias.Foreign.Name == name {
					from = node.AbsolutePath
					break
				}
			}
		}
	}

	if from == "" {
		err = checkInheritedContractsForImports(name, newJsonOutput, __contractName)
		if err != nil {
			return fmt.Errorf("import not found: %s", name)
		} else {
			return nil
		}
	}

	explicitImports[from] = append(explicitImports[from], name)
	detectedImports[name] = true

	return err
}

func writeImports() {
	for file, imports := range explicitImports {
		var lastIndex = len(imports) - 1
		IMPORTS += "import { "
		for i, _import := range imports {
			IMPORTS += _import
			if i != lastIndex {
				IMPORTS += ", "
			}
		}
		IMPORTS += " } from \"" + file + "\";\n"
	}
}

func checkInheritedContractsForImports(name string, oldJsonOutput JsonOutput, __contractName string) error {
	for _, _node := range oldJsonOutput.Ast.Nodes {
		if _node.NodeType == "ContractDefinition" && _node.Name == __contractName {
			for _, baseContract := range _node.BaseContracts {
				newJsonOutput, err := getJsonOutput(baseContract.BaseName.Name)
				if err != nil {
					return err
				}
				if processImportWith(name, newJsonOutput, baseContract.BaseName.Name) == nil {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("import not found")
}

func processType(internalType string, _type string, typeOwner TypeOwner) error {
	var err error

	if detectedTypes[internalType] {
		return err
	}

	internalType = strings.Split(internalType, "[")[0]
	_type = strings.Split(_type, "[")[0]

	if typeOwner == ThisContract {
		TYPES += "  type " + internalType + " is " + _type + ";\n"
	} else if typeOwner == ThisFile {
		FILE_TYPES += " type " + internalType + " is " + _type + ";\n"
	} else {
		err = fmt.Errorf("unreachable:invalid type owner: %v", typeOwner)
		return err
	}

	detectedTypes[internalType] = true

	return err
}

func getJsonOutput(__contractName string) (JsonOutput, error) {
	var newJsonOutput JsonOutput
	contractsBase, _ := os.Getwd()
	var jsonOutputPath = filepath.Join(contractsBase, ARTIFACTS_DIR, __contractName+".sol/", __contractName+".json")
	data, err := os.ReadFile(jsonOutputPath)
	if err != nil {
		return JsonOutput{}, err
	}
	err = json.Unmarshal(data, &newJsonOutput)
	if err != nil {
		return JsonOutput{}, err
	}
	return newJsonOutput, nil
}

func handleStateMutability(stateMutability string) string {
	if stateMutability == "nonpayable" {
		stateMutability = ""
	} else {
		stateMutability = " " + stateMutability
	}
	return stateMutability
}

func shouldAddMemoryAnnotation(parameterType string) bool {
	if isArrayType(parameterType) {
		return true
	}

	switch parameterType {
	case "string", "bytes", "tuple":
		return true
	default:
		return false
	}
}

func isArrayType(parameterType string) bool {
	return strings.HasSuffix(parameterType, "[]")
}

// checks for user-defined types
func isUserDefinedType(typeName string) bool {
	// If the type is not in the list of built-in types, it's considered a user-defined type
	return !builtInTypes[typeName] && !isArrayType(typeName)
}

func combineSections() {
	if IMPORTS != "" {
		IMPORTS += "\n"
	}
	if FILE_TYPES != "" {
		FILE_TYPES += "\n"
	}
	if FILE_ENUMS != "" {
		FILE_ENUMS += "\n"
	}
	if FILE_STRUCTS != "" {
		FILE_STRUCTS += "\n"
	}
	if TYPES != "" {
		TYPES += "\n"
	}
	if ENUMS != "" {
		ENUMS += "\n"
	}
	if STRUCTS != "" {
		STRUCTS += "\n"
	}
	if ERRORS != "" {
		ERRORS += "\n"
	}
	if EVENTS != "" {
		EVENTS += "\n"
	}
	if CONSTRUCTOR != "" {
		CONSTRUCTOR += "\n"
	}
	if FUNCTIONS != "" {
		FUNCTIONS += "\n"
	}
	if FALLBACK != "" {
		FALLBACK += "\n"
	}
	// Note: We don't add a newline after RECEIVE as it's the last section
}

func isImported(name string) (TypeOwner, error) {
	var err error

	if strings.Contains(name, ".") {
		var v = strings.Split(name, ".")
		if !(len(v) > 1) {
			return ThisFile, fmt.Errorf("unreachable:invalid import: %s", name)
		}
		if (v[0] != contractName) && (v[0] != "I"+contractName) {
			return Imported, err
		} else {
			return ThisContract, err
		}
	}
	return ThisFile, err
}

func createDirs(baseDir string, dirs ...string) error {
	for _, dir := range dirs {
		path := filepath.Join(baseDir, dir)
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating directory %s: %w", path, err)
		}
	}
	return nil
}

func findContractFiles(dir string, excludeDirs ...string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			for _, excludeDir := range excludeDirs {
				if info.Name() == excludeDir {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) == ".sol" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return []string{}, fmt.Errorf("error walking the path %v: %w", dir, err)
	}
	return files, err
}

func checkForLooseImports(newJsonOutput JsonOutput) []string {
	var unusedImports []string
	for _, node := range newJsonOutput.Ast.Nodes {
		if node.NodeType == "ImportDirective" && len(node.SymbolAliases) == 0 {
			unusedImports = append(unusedImports, node.AbsolutePath)
		}
	}
	return unusedImports
}
