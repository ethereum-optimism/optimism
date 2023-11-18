package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

type contractsList struct {
	Local  []string         `json:"local"`
	Remote []remoteContract `json:"remote"`
}

// readContractList reads a JSON file from the given `filePath` and unmarshals
// its content into the provided result interface. It logs the path of the file
// being read.
//
// Parameters:
// - logger: An instance of go-ethereum/log
// - filePath: The path to the JSON file to be read.
// - result: A pointer to the structure where the JSON data will be unmarshaled.
//
// Returns:
// - An error if reading the file or unmarshaling fails, nil otherwise.
func readContractList(logger log.Logger, filePath string) (contractsList, error) {
	logger.Debug("Reading contract list", "filePath", filePath)

	var contracts contractsList
	contractData, err := os.ReadFile(filePath)
	if err != nil {
		return contracts, err
	}

	return contracts, json.Unmarshal(contractData, &contracts)
}

// mkTempArtifactsDir creates a temporary directory with a "op-bindings" prefix
// for holding contract artifacts. The path to the created directory is logged.
//
// Parameters:
// - logger: An instance of go-ethereum/log
//
// Returns:
// - The path to the created temporary directory.
// - An error if the directory creation fails, nil otherwise.
func mkTempArtifactsDir(logger log.Logger) (string, error) {
	dir, err := os.MkdirTemp("", "op-bindings")
	if err != nil {
		return "", err
	}

	logger.Debug("Created temporary artifacts directory", "dir", dir)
	return dir, nil
}

// writeContractArtifacts writes the provided ABI and bytecode data to respective
// files in the specified temporary directory. The naming convention for these
// files is based on the provided contract name. The ABI data is written to a file
// with a ".abi" extension, and the bytecode data is written to a file with a ".bin"
// extension.
//
// Parameters:
// - logger: An instance of go-ethereum/log
// - tempDirPath: The directory path where the ABI and bytecode files will be written.
// - contractName: The name of the contract, used to create the filenames.
// - abi: The ABI data of the contract.
// - bytecode: The bytecode of the contract.
//
// Returns:
// - The full path to the written ABI file.
// - The full path to the written bytecode file.
// - An error if writing either file fails, nil otherwise.
func writeContractArtifacts(logger log.Logger, tempDirPath, contractName string, abi, bytecode []byte) (string, string, error) {
	logger.Debug("Writing ABI and bytecode to temporary artifacts directory", "contractName", contractName, "tempDirPath", tempDirPath)

	abiFilePath := path.Join(tempDirPath, contractName+".abi")
	if err := os.WriteFile(abiFilePath, abi, 0o600); err != nil {
		return "", "", fmt.Errorf("error writing %s's ABI file: %w", contractName, err)
	}

	bytecodeFilePath := path.Join(tempDirPath, contractName+".bin")
	if err := os.WriteFile(bytecodeFilePath, bytecode, 0o600); err != nil {
		return "", "", fmt.Errorf("error writing %s's bytecode file: %w", contractName, err)
	}

	return abiFilePath, bytecodeFilePath, nil
}

// genContractBindings generates Go bindings for an Ethereum contract using
// the provided ABI and bytecode files. The bindings are generated using the
// `abigen` tool and are written to the specified Go package directory. The
// generated file's name is based on the provided contract name and will have
// a ".go" extension. The generated bindings will be part of the provided Go
// package.
//
// Parameters:
// - logger: An instance of go-ethereum/log
// - abiFilePath: The path to the ABI file for the contract.
// - bytecodeFilePath: The path to the bytecode file for the contract.
// - goPackageName: The name of the Go package where the bindings will be written.
// - contractName: The name of the contract, used for naming the output file and
// defining the type in the generated bindings.
//
// Returns:
// - An error if there's an issue during any step of the binding generation process,
// nil otherwise.
//
// Note: This function relies on the external `abigen` tool, which should be
// installed and available in the system's PATH.
func genContractBindings(logger log.Logger, abiFilePath, bytecodeFilePath, goPackageName, contractName string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting cwd: %w", err)
	}

	outFilePath := path.Join(cwd, goPackageName, strings.ToLower(contractName)+".go")
	logger.Debug("Generating contract bindings", "contractName", contractName, "outFilePath", outFilePath)

	cmd := exec.Command("abigen", "--abi", abiFilePath, "--bin", bytecodeFilePath, "--pkg", goPackageName, "--type", contractName, "--out", outFilePath)
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running abigen for %s: %w", contractName, err)
	}

	return nil
}
