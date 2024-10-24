package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func processValueTypeReturnParameters(node Node) (string, error) {
	var result string
	var err error

	// search abi to find the function that matches the node.Name, has 0 inputs and has outputs > 0
	for _, entry := range JSON_OUTPUT.ABI {
		if entry.Name == node.Name && entry.Type == "function" && len(entry.Inputs) == 0 && len(entry.Outputs) > 0 {
			for i, output := range entry.Outputs {
				processedType, err := processType(output.InternalType)
				if err != nil {
					return "", err
				}
				var name string = ifNotNullPrefixWhitespace(output.Name)
				var storageLocation string = getStorageLocationFromVariableDeclarationType(output.InternalType)
				result += processedType + storageLocation + name
				if i < len(entry.Outputs)-1 {
					result += ", "
				}
			}
		}
	}

	if result == "" {
		return "", fmt.Errorf("no function found for node %s", node.Name)
	}

	return result, err
}

func processArrayTypeReturnParameters(node Node) (string, error) {
	var result string
	var err error

	var expectedInputLength int = strings.Count(node.TypeDescriptions.TypeString, "[")

	// search abi to find the function that matches the node.Name, has 0 inputs and has outputs > 0
	for _, entry := range JSON_OUTPUT.ABI {
		if entry.Name == node.Name && entry.Type == "function" && len(entry.Inputs) == expectedInputLength && len(entry.Outputs) > 0 {
			// verify all inputs are of type uint256
			allInputsUint256 := true
			for _, input := range entry.Inputs {
				if input.InternalType != "uint256" {
					allInputsUint256 = false
					break
				}
			}
			if !allInputsUint256 {
				continue
			}

			for i, output := range entry.Outputs {
				processedType, err := processType(output.InternalType)
				if err != nil {
					return "", err
				}
				var name string = ifNotNullPrefixWhitespace(output.Name)
				var storageLocation string = getStorageLocationFromVariableDeclarationType(output.InternalType)
				result += processedType + storageLocation + name
				if i < len(entry.Outputs)-1 {
					result += ", "
				}
			}
		}
	}

	if result == "" {
		return "", fmt.Errorf("no function found for node %s", node.Name)
	}

	return result, err
}

func processMappingTypeReturnParameters(node Node, inputTypes []string) (string, error) {
	var result string
	var err error

	// search abi to find the function that matches the node.Name, has 0 inputs and has outputs > 0
	for _, entry := range JSON_OUTPUT.ABI {
		if entry.Name == node.Name && entry.Type == "function" && len(entry.Inputs) == len(inputTypes) && len(entry.Outputs) > 0 {
			// verify all inputs are of type uint256
			allInputsAreCorrect := true
			for i, input := range entry.Inputs {
				processedType, err := processType(input.InternalType)
				if err != nil {
					return "", err
				}
				if processedType != inputTypes[i] {
					allInputsAreCorrect = false
					break
				}
			}
			if !allInputsAreCorrect {
				continue
			}

			for i, output := range entry.Outputs {
				processedType, err := processType(output.InternalType)
				if err != nil {
					return "", err
				}
				var name string = ifNotNullPrefixWhitespace(output.Name)
				var storageLocation string = getStorageLocationFromVariableDeclarationType(output.InternalType)
				result += processedType + storageLocation + name
				if i < len(entry.Outputs)-1 {
					result += ", "
				}
			}
		}
	}

	if result == "" {
		return "", fmt.Errorf("no function found for node %s", node.Name)
	}

	return result, err
}

// Resets all global variables to the default value of their type
func resetVars() {
	contractName = ""
	mainName = make(map[string]string)
	importedNames = make(map[string]string)
	eventSelectors = make(map[string]bool)
	overriddenFunctions = make(map[string]bool)
	errorSelectors = make(map[string]bool)
	constructorFound = false
	level = 0 // redundant?
}

func ifNotNullPrefixWhitespace(value string) string {
	if value == "" {
		return ""
	}
	return " " + value
}

func getStorageLocationFromFunctionParameterNode(node Node) (string, error) {
	var result string = ifNotNullPrefixWhitespace(node.StorageLocation)
	var err error

	if result == " default" {
		if strings.Contains(node.TypeDescriptions.TypeString, "[") {
			result = " memory"
		} else {
			result = ""
		}
	}

	return result, err
}

func getStorageLocationFromVariableDeclarationType(typeName string) string {
	var storageLocation string

	if strings.Contains(typeName, "struct ") || typeName == "string" || typeName == "bytes" {
		storageLocation = " memory"
	}

	return storageLocation
}

func processType(name string) (string, error) {
	var result string = name
	var err error

	if builtInTypes[result] {
		return result, nil
	}

	if strings.Contains(result, " ") {
		result = strings.Split(result, " ")[1]
	}

	if strings.Contains(result, ".") {
		arr := strings.Split(result, ".")
		owner, typeName := arr[0], arr[1]
		if owner == contractName || owner == "I"+contractName {
			result = typeName
		}
	}

	return result, err
}

func getJsonOutput(__contractName string) (JsonOutput, error) {
	var newJsonOutput JsonOutput
	var err error

	contractsBase, _ := os.Getwd()
	var jsonOutputPath = filepath.Join(contractsBase, "forge-artifacts/", __contractName+".sol/", __contractName+".json")
	data, err := os.ReadFile(jsonOutputPath)
	if err != nil {
		jsonOutputPath = filepath.Join(contractsBase, "forge-artifacts/", __contractName+".sol/", __contractName+".0.8.25.json")
		data, err = os.ReadFile(jsonOutputPath)
		if err != nil {
			switch __contractName {
			// For governance contract that imports ERC20Votes which imports ERC20Permit from OZ. Handle its transient dependencies
			case "ERC20Permit", "IERC20Permit", "EIP712":
				{
					jsonOutputPath = filepath.Join(contractsBase, "forge-artifacts/", "draft-"+__contractName+".sol/", __contractName+".0.8.25.json")
					data, err = os.ReadFile(jsonOutputPath)
					if err != nil {
						return JsonOutput{}, err
					}
				}
			default:
				return JsonOutput{}, err
			}
		}

	}
	err = json.Unmarshal(data, &newJsonOutput)
	if err != nil {
		return JsonOutput{}, err
	}
	return newJsonOutput, nil
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

// func getInterfacePathOf(name string) (string, error) {
//  var result string
//  var err error

//  // Split the path into directory and file components
//  dir, file := filepath.Split(name)

//  // Remove the file extension
//  baseName := strings.TrimSuffix(file, ".sol")

//  // Add "I" prefix to the file name
//  interfaceName := "I" + baseName

//  // Construct the new path
//  result = filepath.Join(dir, "interfaces", interfaceName+".sol")

//  // // If the path doesn't start with "src", add it
//  // if !strings.HasPrefix(result, "src") {
//  //  result = filepath.Join("src", result)
//  // }

//  return result, err
// }
