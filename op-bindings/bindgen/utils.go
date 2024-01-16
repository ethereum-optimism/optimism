package bindgen

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

type BindGenGeneratorBase struct {
	MetadataOut         string
	BindingsPackageName string
	MonorepoBasePath    string
	ContractsListPath   string
	Logger              log.Logger
}

type contractsList struct {
	Local  []string         `json:"local"`
	Remote []RemoteContract `json:"remote"`
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
func genContractBindings(logger log.Logger, monorepoRootPath, abiFilePath, bytecodeFilePath, goPackageName, contractName string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting cwd: %w", err)
	}

	outFilePath := path.Join(cwd, goPackageName, strings.ToLower(contractName)+".go")

	var existingOutput []byte
	if _, err := os.Stat(outFilePath); err == nil {
		existingOutput, err = os.ReadFile(outFilePath)
		if err != nil {
			return fmt.Errorf("error reading existing bindings output file, outFilePath: %s err: %w", outFilePath, err)
		}
	}

	if monorepoRootPath != "" {
		logger.Debug("Checking abigen version")

		// Fetch installed abigen version (format: abigen version X.Y.Z-<stable/nightly>-<commit_sha>)
		cmd := exec.Command("abigen", "--version")
		var versionBuf bytes.Buffer
		cmd.Stdout = bufio.NewWriter(&versionBuf)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error fetching abigen version: %w", err)
		}
		abigenVersion := bytes.Trim(versionBuf.Bytes(), "\n")

		// Fetch expected abigen version (format: vX.Y.Z)
		expectedAbigenVersion, err := readExpectedAbigenVersion(monorepoRootPath)
		if err != nil {
			return fmt.Errorf("error fetching the expected abigen version: %w", err)
		}

		if !bytes.Contains(abigenVersion, []byte(expectedAbigenVersion)) {
			return fmt.Errorf("abigen version mismatch, expected %s, got %s. Please run `pnpm install:abigen` in the monorepo root", expectedAbigenVersion, abigenVersion)
		}
	} else {
		logger.Debug("No monorepo root path provided, skipping abigen version check")
	}

	logger.Debug("Generating contract bindings", "contractName", contractName, "outFilePath", outFilePath)
	cmd := exec.Command("abigen", "--abi", abiFilePath, "--bin", bytecodeFilePath, "--pkg", goPackageName, "--type", contractName, "--out", outFilePath)
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running abigen for %s: %w", contractName, err)
	}

	if len(existingOutput) != 0 {
		newOutput, err := os.ReadFile(outFilePath)
		if err != nil {
			return fmt.Errorf("error reading new file: %w", err)
		}

		if bytes.Equal(existingOutput, newOutput) {
			logger.Debug("No changes detected in the contract bindings", "contractName", contractName)
		} else {
			logger.Warn("Changes detected in the contract bindings, old bindings have been overwritten", "contractName", contractName)
		}
	} else {
		logger.Debug("No existing contract bindings found, skipping comparison", "contractName", contractName)
	}

	return nil
}

// Versions is a struct for holding the versions of the tools used in the monorepo
type Versions struct {
	Abigen  string `json:"abigen"`
	Foundry string `json:"foundry"`
	Geth    string `json:"geth"`
	Nvm     string `json:"nvm"`
	Slither string `json:"slither"`
	Kontrol string `json:"kontrol"`
}

// readExpectedAbigenVersion reads the expected abigen version from the monorepo's
// versions.json file. This function will remove the 'v' prefix from the version
// string.
//
// Parameters:
// - monorepoRootPath: The path to the monorepo's root directory.
//
// Returns:
// - The expected abigen version.
// - An error if the versions.json file cannot be read or parsed, nil otherwise.
func readExpectedAbigenVersion(monorepoRootPath string) (string, error) {
	// Open the version control file
	jsonFile, err := os.Open(path.Join(monorepoRootPath, "versions.json"))
	if err != nil {
		return "", fmt.Errorf("error reading versions.json file: %w", err)
	}
	defer jsonFile.Close()

	// Parse the version control file
	byteValue, _ := io.ReadAll(jsonFile)
	var versions Versions
	if err := json.Unmarshal(byteValue, &versions); err != nil {
		return "", fmt.Errorf("error parsing versions.json file: %w", err)
	}

	// Trim the 'v' prefix from the version string
	return strings.Trim(versions.Abigen, "v"), nil
}
