// Package main implements a CLI tool to verify deployed Ethereum contract bytecode
// against local build artifacts. It supports verifying single contracts, blueprints
// (ERC-5202), and the contracts managed by an OPContractsManager instance.
package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	ccom "github.com/ethereum-optimism/optimism/packages/contracts-bedrock/scripts/checks/common"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/scripts/checks/verify-bytecode/bindings"
)

// VerificationType indicates the kind of verification performed.
type VerificationType string

// VerificationType constants.
const (
	DeployedContract      VerificationType = "deployed contract"
	Blueprint             VerificationType = "blueprint"
	SplitBlueprintPart1   VerificationType = "split blueprint part 1"
	SplitBlueprintPart2   VerificationType = "split blueprint part 2"
	OPContractsManager    VerificationType = "OPContractsManager"
	Implementation        VerificationType = "implementation"
	UnknownImplementation VerificationType = "unknown implementation"
	UnknownBlueprint      VerificationType = "unknown blueprint"
)

// BytecodeDifference represents a contiguous block of differing bytes found during comparison.
type BytecodeDifference struct {
	Start         int    // Byte offset where the difference begins.
	Length        int    // Length of the differing block in bytes.
	Expected      string // Expected bytes (hex encoded).
	Actual        string // Actual bytes found onchain (hex encoded).
	InImmutable   bool   // True if this difference falls within a known immutable reference range.
	ImmutableName string // Name of the immutable variable if InImmutable is true.
}

// ImmutableValueInfo holds details about a specific immutable variable's location and the value found there.
type ImmutableValueInfo struct {
	Name   string // Human-readable name (best effort via AST)
	Offset int    // Byte offset where the immutable value starts.
	Length int    // Length of the immutable value in bytes.
	Value  string // Actual value found at this location in the deployed bytecode (hex encoded).
}

// VerificationResult encapsulates the outcome of a single verification check.
type VerificationResult struct {
	Type           VerificationType
	ContractName   string
	FieldName      string
	Address        string
	AddressPart2   string
	ArtifactPath   string
	ProcessError   error
	Differences    []BytecodeDifference
	ImmutableInfos []ImmutableValueInfo
	TargetContract string
}

// ArtifactConfig holds configuration related to finding contract artifacts.
type ArtifactConfig struct {
	ArtifactsDir            string
	ImplementationOverrides map[string]string
	BlueprintOverrides      map[string]string
	DefaultOPCMArtifactName string
}

// ContractArtifact holds the relevant data extracted from a single artifact JSON file.
type ContractArtifact struct {
	ContractName     string
	DeployedBytecode string
	CreationBytecode string
	ImmutableRefs    map[string][]immutableLocation
	RawAST           map[string]any
}

// immutableLocation is an internal helper struct used during artifact parsing and comparison.
type immutableLocation struct {
	Offset int
	Length int
	Value  string
}

// currentDiff is a temporary helper struct used internally by findDifferences logic
type currentDiff struct {
	Start         int
	Expected      []string
	Actual        []string
	InImmutable   bool
	ImmutableName string
}

// Defaults and constants.
const defaultArtifactsDir = "forge-artifacts"
const defaultOPCMContractName = "OPContractsManager"
const blueprintPreamble = "0xFE7100"
const maxInitCodeSize = 24573 // 24 KiB - 3 byte preamble

// trailingDigitsRegex matches the last sequence of digits in a string.
var trailingDigitsRegex = regexp.MustCompile(`\d+$`)

// Function variable for dependency injection / mocking in tests
var getOnchainBytecodeImpl = getOnchainBytecode

// main is the entrypoint for the verify-bytecode CLI tool.
func main() {
	// Default override maps
	// These map OPCM struct field names to artifact file paths relative to artifacts-dir
	implementationArtifactOverrides := map[string]string{
		"OptimismPortalImpl": "OptimismPortal2.sol/OptimismPortal2.json",
		// Add other overrides if needed
	}
	blueprintArtifactOverrides := map[string]string{
		"PermissionlessDisputeGame1": "FaultDisputeGame.sol/FaultDisputeGame.json",
		"PermissionlessDisputeGame2": "FaultDisputeGame.sol/FaultDisputeGame.json",
		"PermissionedDisputeGame1":   "PermissionedDisputeGame.sol/PermissionedDisputeGame.json",
		"PermissionedDisputeGame2":   "PermissionedDisputeGame.sol/PermissionedDisputeGame.json",
		// Add other overrides if needed
	}

	app := &cli.App{
		Name:  "verify-bytecode",
		Usage: "Verify onchain contract bytecode against local build artifacts",
		Commands: []*cli.Command{
			{
				Name:  "single",
				Usage: "Verify a single deployed contract",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "address",
						Usage:    "Contract address to check",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "artifact",
						Usage:    "Path to the contract artifact JSON file (can be absolute or relative to artifacts-dir)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "rpc-url",
						Usage:    "RPC URL for the network",
						Required: true,
						EnvVars:  []string{"ETH_RPC_URL"},
					},
					&cli.StringFlag{
						Name:    "artifacts-dir",
						Usage:   "Base directory containing the forge compilation artifacts",
						Value:   defaultArtifactsDir,
						EnvVars: []string{"ARTIFACTS_DIR"},
					},
					&cli.BoolFlag{
						Name:  "verbose",
						Usage: "Print detailed immutable diff information even on success",
						Value: false,
					},
				},
				Action: func(c *cli.Context) error {
					return runVerifySingle(c, implementationArtifactOverrides, blueprintArtifactOverrides)
				},
			},
			{
				Name:  "opcm",
				Usage: "Verify OPContractsManager and its managed implementations and blueprints",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "opcm-address",
						Usage:    "OPContractsManager contract address",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "rpc-url",
						Usage:    "RPC URL for the network",
						Required: true,
						EnvVars:  []string{"ETH_RPC_URL"},
					},
					&cli.StringFlag{
						Name:    "artifacts-dir",
						Usage:   "Base directory containing the forge compilation artifacts",
						Value:   defaultArtifactsDir,
						EnvVars: []string{"ARTIFACTS_DIR"},
					},
					&cli.BoolFlag{
						Name:  "verbose",
						Usage: "Print detailed immutable diff information even on success",
						Value: false,
					},
				},
				Action: func(c *cli.Context) error {
					return runVerifyOPCM(c, implementationArtifactOverrides, blueprintArtifactOverrides)
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		color.Unset()
		os.Exit(1)
	}
}

// runVerifySingle performs verification for a single deployed contract.
func runVerifySingle(c *cli.Context, implOverrides, bpOverrides map[string]string) error {
	rpcURL := c.String("rpc-url")
	artifactsDir := c.String("artifacts-dir")
	addressHex := c.String("address")
	artifactPathArg := c.String("artifact")
	verbose := c.Bool("verbose")

	// Resolve artifact path
	artifactPath, err := resolvePath(artifactPathArg, artifactsDir)
	if err != nil {
		// Print error directly as this is a setup failure before core logic runs
		color.Red("Error resolving artifact path: %v", err)
		return cli.Exit("", 1)
	}

	// Create Ethereum client
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		color.Red("Error connecting to RPC %s: %v", rpcURL, err)
		return cli.Exit("", 1)
	}
	defer client.Close()

	// Load artifact
	artifact, err := ccom.ReadForgeArtifact(artifactPath)
	if err != nil {
		// Handle artifact loading error before verification
		contractName := strings.TrimSuffix(filepath.Base(artifactPath), ".json") // Best effort name from path
		color.Red("Error loading artifact %s: %v", artifactPath, err)
		printResults([]*VerificationResult{{
			Type:         DeployedContract,
			Address:      addressHex,
			ArtifactPath: artifactPath,
			ContractName: contractName,
			ProcessError: fmt.Errorf("loading artifact: %w", err),
		}}, verbose)
		return cli.Exit("", 1) // Exit with error code 1 if artifact fails to load
	}
	// Derive name from path after successful load
	contractName := strings.TrimSuffix(filepath.Base(artifactPath), ".json")

	// Perform verification
	addr := common.HexToAddress(addressHex)
	// Pass loaded artifact, derived name, and original path
	result := verifyDeployedContractLogic(client, artifact, contractName, artifactPath, addr)

	// Print results
	printResults([]*VerificationResult{result}, verbose)

	// Determine exit code
	exitCode := 0
	if result.ProcessError != nil || hasCodeDifferences(result) {
		exitCode = 1
	}
	return cli.Exit("", exitCode)
}

func runVerifyOPCM(c *cli.Context, implOverrides, bpOverrides map[string]string) error {
	rpcURL := c.String("rpc-url")
	artifactsDir := c.String("artifacts-dir")
	opcmAddressHex := c.String("opcm-address")
	verbose := c.Bool("verbose")

	// Resolve base artifact directory path
	baseArtifactsDir, err := resolvePath("", artifactsDir) // Resolve artifactsDir itself
	if err != nil {
		color.Red("Error resolving artifacts directory path: %v", err)
		return cli.Exit("", 1)
	}

	// Create ArtifactConfig
	config := ArtifactConfig{
		ArtifactsDir:            baseArtifactsDir,
		ImplementationOverrides: implOverrides,
		BlueprintOverrides:      bpOverrides,
		DefaultOPCMArtifactName: defaultOPCMContractName,
	}

	// Create Ethereum client
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		color.Red("Error connecting to RPC %s: %v", rpcURL, err)
		return cli.Exit("", 1)
	}
	defer client.Close()

	// Perform verification for OPCM and all its managed contracts
	opcmAddr := common.HexToAddress(opcmAddressHex)
	results := runOPCMVerificationLogic(client, opcmAddr, config)

	// Print results
	printResults(results, verbose)

	// Determine exit code
	exitCode := 0
	for _, result := range results {
		if result.ProcessError != nil || hasCodeDifferences(result) {
			exitCode = 1
			break
		}
	}
	return cli.Exit("", exitCode)
}

// verifyDeployedContractLogic performs verification for a standard deployed contract.
// It now accepts a pre-loaded artifact and contract name.
func verifyDeployedContractLogic(client *ethclient.Client, artifact *solc.ForgeArtifact, contractName string, artifactPath string, address common.Address) *VerificationResult {
	result := &VerificationResult{
		Type:         DeployedContract,
		Address:      address.Hex(),
		ArtifactPath: artifactPath, // Still store the path for reporting
		ContractName: contractName, // Use passed-in name
	}

	// Get onchain bytecode
	actualBytecode, err := getOnchainBytecodeImpl(client, address)
	if err != nil {
		result.ProcessError = fmt.Errorf("getting onchain bytecode: %w", err)
		return result
	}

	// Compare bytecode
	differences, immutables, err := compareBytecode(artifact, true, artifact.DeployedBytecode.Object, actualBytecode)
	if err != nil {
		result.ProcessError = fmt.Errorf("comparing bytecode: %w", err)
		return result
	}
	result.Differences = differences
	result.ImmutableInfos = immutables

	return result
}

// verifyBlueprintLogic performs verification for a single ERC-5202 blueprint.
// It now accepts a pre-loaded artifact and target contract name.
func verifyBlueprintLogic(client *ethclient.Client, artifact *solc.ForgeArtifact, targetContractName string, targetArtifactPath string, blueprintAddress common.Address, blueprintFieldName string) *VerificationResult {
	result := &VerificationResult{
		Type:           Blueprint,
		FieldName:      blueprintFieldName,
		Address:        blueprintAddress.Hex(),
		ArtifactPath:   targetArtifactPath,                                  // Path to the artifact of the contract *created* by the blueprint
		TargetContract: targetContractName,                                  // Use passed-in name
		ContractName:   fmt.Sprintf("Blueprint for %s", targetContractName), // Use passed-in name
	}

	// Error if no creation bytecode found
	if artifact.Bytecode.Object == "" || artifact.Bytecode.Object == "0x" {
		result.ProcessError = fmt.Errorf("no creation bytecode found in target artifact %s", targetArtifactPath)
		return result
	}

	// Construct expected blueprint bytecode
	expectedBlueprintBytecode := blueprintPreamble + strings.TrimPrefix(artifact.Bytecode.Object, "0x")

	// Get actual blueprint bytecode from chain
	actualBytecode, err := getOnchainBytecodeImpl(client, blueprintAddress)
	if err != nil {
		result.ProcessError = fmt.Errorf("getting onchain bytecode for blueprint %s: %w", blueprintAddress.Hex(), err)
		return result
	}

	// Compare bytecode (no immutables for blueprints)
	differences, _, err := compareBytecode(artifact, false, expectedBlueprintBytecode, actualBytecode)
	if err != nil {
		result.ProcessError = errors.Join(result.ProcessError, fmt.Errorf("comparing blueprint bytecode: %w", err))
		return result
	}
	result.Differences = differences

	return result
}

// verifySplitBlueprintLogic verifies a blueprint split into two parts.
// It now accepts a pre-loaded artifact and target contract name.
func verifySplitBlueprintLogic(client *ethclient.Client, artifact *solc.ForgeArtifact, targetContractName string, targetArtifactPath string, address1, address2 common.Address, fieldName1, fieldName2 string) (*VerificationResult, *VerificationResult) {
	result1 := &VerificationResult{
		Type:           SplitBlueprintPart1,
		FieldName:      fieldName1,
		Address:        address1.Hex(),
		AddressPart2:   address2.Hex(),
		ArtifactPath:   targetArtifactPath,
		TargetContract: targetContractName,                                     // Use passed-in name
		ContractName:   fmt.Sprintf("Split BP 1/2 for %s", targetContractName), // Use passed-in name
	}
	result2 := &VerificationResult{
		Type:           SplitBlueprintPart2,
		FieldName:      fieldName2,
		Address:        address2.Hex(),
		ArtifactPath:   targetArtifactPath,
		TargetContract: targetContractName,                                     // Use passed-in name
		ContractName:   fmt.Sprintf("Split BP 2/2 for %s", targetContractName), // Use passed-in name
	}

	// Error if no creation bytecode found
	if artifact.Bytecode.Object == "" || artifact.Bytecode.Object == "0x" {
		err := fmt.Errorf("no creation bytecode found in target artifact %s", targetArtifactPath)
		result1.ProcessError = err
		result2.ProcessError = err
		return result1, result2
	}

	// Split creation code
	fullCreationCodeHex := strings.TrimPrefix(artifact.Bytecode.Object, "0x")
	fullCreationCodeBytes, err := hex.DecodeString(fullCreationCodeHex)
	if err != nil {
		err = fmt.Errorf("failed to decode creation code hex from %s: %w", targetArtifactPath, err)
		result1.ProcessError = err
		result2.ProcessError = err
		return result1, result2
	}

	// Split up creation code
	part1Bytes := fullCreationCodeBytes
	var part2Bytes []byte
	if len(fullCreationCodeBytes) > maxInitCodeSize {
		part1Bytes = fullCreationCodeBytes[:maxInitCodeSize]
		part2Bytes = fullCreationCodeBytes[maxInitCodeSize:]
	} else {
		// This case should ideally be handled by the caller (runOPCMVerificationLogic)
		// If it gets here, treat part 2 as empty.
		part2Bytes = []byte{}
	}

	// Construct expected bytecodes
	expectedBytecode1 := blueprintPreamble + hex.EncodeToString(part1Bytes)
	expectedBytecode2 := blueprintPreamble + hex.EncodeToString(part2Bytes)

	// Fetch actual bytecode for address 1
	actualBytecode1, err1 := getOnchainBytecodeImpl(client, address1)
	if err1 != nil {
		result1.ProcessError = fmt.Errorf("getting onchain code for part 1 (%s): %w", address1.Hex(), err1)
	}

	// Fetch actual bytecode for address 2
	actualBytecode2, err2 := getOnchainBytecodeImpl(client, address2)
	if err2 != nil {
		result2.ProcessError = fmt.Errorf("getting onchain code for part 2 (%s): %w", address2.Hex(), err2)
	}

	// Compare Part 1
	if result1.ProcessError == nil {
		diffs1, _, cmpErr1 := compareBytecode(artifact, false, expectedBytecode1, actualBytecode1)
		if cmpErr1 != nil {
			result1.ProcessError = errors.Join(result1.ProcessError, fmt.Errorf("comparing part 1 bytecode: %w", cmpErr1))
		}
		result1.Differences = diffs1
	}

	// Compare Part 2
	if result2.ProcessError == nil {
		diffs2, _, cmpErr2 := compareBytecode(artifact, false, expectedBytecode2, actualBytecode2)
		if cmpErr2 != nil {
			result2.ProcessError = errors.Join(result2.ProcessError, fmt.Errorf("comparing part 2 bytecode: %w", cmpErr2))
		}
		result2.Differences = diffs2
	}

	return result1, result2
}

// runOPCMVerificationLogic orchestrates verification for OPCM and its managed contracts.
func runOPCMVerificationLogic(client *ethclient.Client, opcmAddress common.Address, config ArtifactConfig) []*VerificationResult {
	results := []*VerificationResult{}

	// Verify OPCM itself
	opcmArtifactBase := config.DefaultOPCMArtifactName
	opcmArtifactRelative := filepath.Join(fmt.Sprintf("%s.sol", opcmArtifactBase), fmt.Sprintf("%s.json", opcmArtifactBase))
	opcmArtifactPath := filepath.Join(config.ArtifactsDir, opcmArtifactRelative)

	// Load OPCM artifact first
	opcmArtifact, err := ccom.ReadForgeArtifact(opcmArtifactPath)
	// Derive name from path before potentially erroring out
	opcmContractName := config.DefaultOPCMArtifactName // Use default name as fallback
	// Attempt to refine name from path
	if nameFromPath := strings.TrimSuffix(filepath.Base(opcmArtifactPath), ".json"); nameFromPath != "" {
		opcmContractName = nameFromPath
	}
	if err != nil {
		// If OPCM artifact fails to load, create an error result and cannot proceed
		results = append(results, &VerificationResult{
			Type:         OPContractsManager,
			Address:      opcmAddress.Hex(),
			ArtifactPath: opcmArtifactPath,
			ContractName: opcmContractName, // Use derived name
			ProcessError: fmt.Errorf("loading OPCM artifact %s: %w", opcmArtifactPath, err),
		})
		return results
	}
	// Name is already derived above

	// Verify OPCM itself using the loaded artifact
	opcmResult := verifyDeployedContractLogic(client, opcmArtifact, opcmContractName, opcmArtifactPath, opcmAddress)
	opcmResult.Type = OPContractsManager // Override type
	// ContractName is already set correctly by verifyDeployedContractLogic
	results = append(results, opcmResult)

	// Cannot proceed if OPCM verification itself had a processing error (e.g., RPC down)
	// or if the OPCM address has no code (can't call it).
	if opcmResult.ProcessError != nil {
		opcmResult.ProcessError = errors.Join(opcmResult.ProcessError, errors.New("cannot query implementations/blueprints due to OPCM verification error"))
		return results
	}

	// Assuming OPCM verification passed or had only bytecode diffs, we can try to bind
	opcmCaller, err := bindings.NewOpcm200Caller(opcmAddress, client)
	if err != nil {
		// Add a synthetic result to indicate this failure
		results = append(results, &VerificationResult{
			Type:         OPContractsManager,
			ContractName: config.DefaultOPCMArtifactName,
			Address:      opcmAddress.Hex(),
			ProcessError: fmt.Errorf("failed to bind OPCM caller: %w", err),
		})
		return results // Cannot proceed without caller
	}

	// Verify implementations
	implementationsResult, err := opcmCaller.Implementations(nil)
	if err != nil {
		results = append(results, &VerificationResult{
			Type:         OPContractsManager, // Attributing error to OPCM interaction
			ContractName: config.DefaultOPCMArtifactName,
			Address:      opcmAddress.Hex(),
			ProcessError: fmt.Errorf("failed to call implementations() on OPCM: %w", err),
		})
	} else {
		implValue := reflect.ValueOf(implementationsResult)
		implType := implValue.Type()

		for i := 0; i < implValue.NumField(); i++ {
			fieldName := implType.Field(i).Name
			fieldValue := implValue.Field(i).Interface().(common.Address)

			if fieldValue == (common.Address{}) {
				continue // Skip zero addresses silently
			}

			implAddressStr := fieldValue.Hex()
			var relativePath string
			var ok bool

			// Determine artifact path using overrides or convention
			if relativePath, ok = config.ImplementationOverrides[fieldName]; !ok {
				if strings.HasSuffix(fieldName, "Impl") {
					baseName := strings.TrimSuffix(fieldName, "Impl")
					relativePath = filepath.Join(fmt.Sprintf("%s.sol", baseName), fmt.Sprintf("%s.json", baseName))
				} else {
					// Cannot determine path
					results = append(results, &VerificationResult{
						Type:         UnknownImplementation,
						FieldName:    fieldName,
						ContractName: fmt.Sprintf("Unknown (%s)", fieldName),
						Address:      implAddressStr,
						ProcessError: fmt.Errorf("cannot infer artifact path for implementation field '%s' (no override and doesn't end in 'Impl')", fieldName),
					})
					continue
				}
			}

			artifactPath := filepath.Join(config.ArtifactsDir, relativePath)

			// Load implementation artifact
			implArtifact, err := ccom.ReadForgeArtifact(artifactPath)
			// Derive name from path before potentially erroring out
			implContractName := strings.TrimSuffix(filepath.Base(artifactPath), ".json")
			if err != nil {
				// Handle artifact loading error for this implementation
				results = append(results, &VerificationResult{
					Type:         Implementation,
					FieldName:    fieldName,
					ContractName: implContractName, // Use derived name
					Address:      implAddressStr,
					ArtifactPath: artifactPath,
					ProcessError: fmt.Errorf("loading implementation artifact %s: %w", artifactPath, err),
				})
				continue // Skip to next implementation
			}
			// Name is already derived

			// Verify implementation using loaded artifact
			implResult := verifyDeployedContractLogic(client, implArtifact, implContractName, artifactPath, fieldValue)
			implResult.Type = Implementation // Override type
			implResult.FieldName = fieldName // Store the field name
			// ContractName is already set correctly
			results = append(results, implResult)
		}
	}

	// Verify blueprints
	blueprintsResult, err := opcmCaller.Blueprints(nil)
	if err != nil {
		results = append(results, &VerificationResult{
			Type:         OPContractsManager, // Attributing error to OPCM interaction
			ContractName: config.DefaultOPCMArtifactName,
			Address:      opcmAddress.Hex(),
			ProcessError: fmt.Errorf("failed to call blueprints() on OPCM: %w", err),
		})
	} else {
		blueprintValue := reflect.ValueOf(blueprintsResult)
		blueprintType := blueprintValue.Type()
		blueprintFields := make(map[string]common.Address)
		processedPart2 := make(map[string]bool) // Track part 2 blueprints already handled

		// First pass: collect all blueprint addresses
		for i := 0; i < blueprintValue.NumField(); i++ {
			fieldName := blueprintType.Field(i).Name
			fieldValue := blueprintValue.Field(i).Interface().(common.Address)
			blueprintFields[fieldName] = fieldValue
		}

		// Second pass: verify each blueprint, handling splits
		for i := 0; i < blueprintValue.NumField(); i++ {
			fieldName := blueprintType.Field(i).Name
			fieldValue := blueprintValue.Field(i).Interface().(common.Address)

			if processedPart2[fieldName] || fieldValue == (common.Address{}) {
				continue // Skip zero addresses and already processed part 2s
			}

			blueprintAddressStr := fieldValue.Hex()
			var relativePath string
			var baseName string
			var ok bool

			// Determine artifact path for the *target* contract
			if relativePath, ok = config.BlueprintOverrides[fieldName]; !ok {
				baseName = trailingDigitsRegex.ReplaceAllString(fieldName, "")
				if baseName == "" {
					results = append(results, &VerificationResult{
						Type:         UnknownBlueprint,
						FieldName:    fieldName,
						ContractName: fmt.Sprintf("Unknown (%s)", fieldName),
						Address:      blueprintAddressStr,
						ProcessError: fmt.Errorf("cannot infer artifact path for blueprint field '%s' (no override and empty base name)", fieldName),
					})
					continue
				}
				relativePath = filepath.Join(fmt.Sprintf("%s.sol", baseName), fmt.Sprintf("%s.json", baseName))
			} else {
				// Infer baseName from override path if possible, fallback to field name
				parts := strings.Split(filepath.ToSlash(relativePath), "/")
				if len(parts) == 2 && strings.HasSuffix(parts[0], ".sol") && strings.HasSuffix(parts[1], ".json") {
					baseName = strings.TrimSuffix(parts[1], ".json")
				} else {
					baseName = trailingDigitsRegex.ReplaceAllString(fieldName, "") // Fallback
				}
			}

			targetArtifactPath := filepath.Join(config.ArtifactsDir, relativePath)

			// Load target artifact (used for both single and split blueprints)
			targetArtifact, err := ccom.ReadForgeArtifact(targetArtifactPath)
			// Derive target contract name from baseName determined earlier
			// baseName was derived from field name or override path
			targetContractName := baseName
			if err != nil {
				// Handle artifact loading error
				errResult := VerificationResult{
					Type:      UnknownBlueprint, // Or SplitBlueprintPart1 if applicable
					FieldName: fieldName,
					// Use baseName for the failed contract's name if possible
					ContractName: fmt.Sprintf("Unknown (%s - loading failed)", targetContractName), // Indicate loading failure
					Address:      blueprintAddressStr,
					ArtifactPath: targetArtifactPath,
					ProcessError: fmt.Errorf("loading target artifact %s: %w", targetArtifactPath, err),
				}
				results = append(results, &errResult)
				// If it was potentially a split, maybe add a placeholder for part 2? Less clear.
				// For now, just report the loading error once.
				continue // Skip to next blueprint field
			}
			// Name is already derived

			// Check for split blueprint
			if strings.HasSuffix(fieldName, "1") {
				part2FieldName := strings.TrimSuffix(fieldName, "1") + "2"
				if part2Addr, exists := blueprintFields[part2FieldName]; exists && part2Addr != (common.Address{}) {
					// Verify as split blueprint using loaded artifact
					res1, res2 := verifySplitBlueprintLogic(client, targetArtifact, targetContractName, targetArtifactPath, fieldValue, part2Addr, fieldName, part2FieldName)
					results = append(results, res1, res2)
					processedPart2[part2FieldName] = true // Mark part 2 as handled
					continue                              // Move to next field
				} else {
					// Error out
					results = append(results, &VerificationResult{
						Type:         UnknownBlueprint,
						FieldName:    fieldName,
						ContractName: fmt.Sprintf("Unknown (%s)", fieldName),
						Address:      blueprintAddressStr,
						ProcessError: fmt.Errorf("split blueprint part 2 not found for %s", fieldName),
					})
				}
			}

			// Verify as a standard (single) blueprint using loaded artifact
			bpResult := verifyBlueprintLogic(client, targetArtifact, targetContractName, targetArtifactPath, fieldValue, fieldName)
			results = append(results, bpResult)
		}
	}

	return results
}

// getOnchainBytecode fetches bytecode from the chain. Returns hex string or error.
func getOnchainBytecode(client *ethclient.Client, address common.Address) (string, error) {
	if client == nil {
		return "", errors.New("ethereum client is nil")
	}
	code, err := client.CodeAt(context.Background(), address, nil)
	if err != nil {
		return "", err
	}
	if len(code) == 0 {
		return "0x", errors.New("no code found at address")
	}
	return "0x" + hex.EncodeToString(code), nil
}

// compareBytecode compares expected and actual bytecode, handling immutables.
// It uses the artifact to find immutable names via the AST.
func compareBytecode(
	artifact *solc.ForgeArtifact,
	checkImmutables bool,
	expectedBytecodeHex string,
	actualBytecodeHex string,
) ([]BytecodeDifference, []ImmutableValueInfo, error) {
	// Input validation and decoding
	expectedClean := strings.TrimPrefix(expectedBytecodeHex, "0x")
	actualClean := strings.TrimPrefix(actualBytecodeHex, "0x")

	if len(expectedClean)%2 != 0 {
		return nil, nil, fmt.Errorf("invalid expected bytecode hex length: %d", len(expectedClean))
	}
	// Allow empty or odd length for actual if it came from chain (e.g., "0x")?
	// For now, strict check on actual too. If empty actual is valid, adjust here.
	if actualClean != "" && len(actualClean)%2 != 0 {
		// return nil, nil, fmt.Errorf("invalid actual bytecode hex length: %d", len(actualClean))
		// Or treat as empty if needed: actualClean = ""
	}

	expectedBytes, err := hex.DecodeString(expectedClean)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode expected bytecode: %w", err)
	}

	actualBytes, err := hex.DecodeString(actualClean)
	if err != nil {
		// Allow comparison if actual is empty/invalid hex from chain (treat as empty)
		if actualClean == "" {
			actualBytes = []byte{}
		} else {
			return nil, nil, fmt.Errorf("failed to decode actual bytecode '%s': %w", actualBytecodeHex, err)
		}
	}

	// Error if bytecode lengths don't match
	if len(expectedBytes) != len(actualBytes) {
		return nil, nil, fmt.Errorf("bytecode length mismatch, expected: %d, actual: %d", len(expectedBytes), len(actualBytes))
	}

	// Precompute immutable locations
	type immutableByteInfo struct {
		Name string
	}
	immutableBytes := make(map[int]immutableByteInfo)

	if artifact != nil && artifact.DeployedBytecode.ImmutableReferences != nil {
		for refKey, locations := range artifact.DeployedBytecode.ImmutableReferences {
			name := getImmutableName(artifact, refKey)
			if name == "" {
				name = fmt.Sprintf("immutable(id:%s)", refKey) // Fallback name
			}

			info := immutableByteInfo{Name: name}
			for _, loc := range locations {
				start := int(loc.Start)
				length := int(loc.Length)
				if length <= 0 {
					continue // Skip invalid length
				}

				// Mark each byte within this location using 'j'
				for j := 0; j < length; j++ {
					offset := start + j
					if existing, exists := immutableBytes[offset]; exists {
						if existing.Name != info.Name {
							fmt.Fprintf(os.Stderr, "Warning: Overlapping immutable reference at offset %d. Prev: '%s', New: '%s'\n",
								offset, existing.Name, info.Name)
						}
					}
					immutableBytes[offset] = info
				}
			}
		}
	}

	// Compare byte by byte
	differences := []BytecodeDifference{}
	var currDiff *currentDiff = nil // Tracks the current block of differences
	maxLength := max(len(expectedBytes), len(actualBytes))

	for i := 0; i < maxLength; i++ {
		// Determine immutable status for the current byte offset
		inImmutableRange := false
		immName := ""
		if info, ok := immutableBytes[i]; ok {
			inImmutableRange = true
			immName = info.Name
		}

		// Get bytes and hex representations, handling out-of-bounds access
		var expectedByte, actualByte byte
		var expectedHex, actualHex string = "..", ".." // Use ".." for out-of-bounds

		if i < len(expectedBytes) {
			expectedByte = expectedBytes[i]
			expectedHex = fmt.Sprintf("%02x", expectedByte)
		}
		if i < len(actualBytes) {
			actualByte = actualBytes[i]
			actualHex = fmt.Sprintf("%02x", actualByte)
		}

		bytesDiffer := expectedByte != actualByte

		// State machine logic for tracking differences
		if bytesDiffer {
			if currDiff == nil {
				// Start a new difference block
				currDiff = &currentDiff{
					Start:         i,
					Expected:      []string{expectedHex},
					Actual:        []string{actualHex},
					InImmutable:   inImmutableRange,
					ImmutableName: immName,
				}
			} else if currDiff.InImmutable != inImmutableRange || (inImmutableRange && currDiff.ImmutableName != immName) {
				// End the previous block because immutable status or name changed, then start a new one
				differences = append(differences, BytecodeDifference{
					Start:         currDiff.Start,
					Length:        len(currDiff.Expected),
					Expected:      strings.Join(currDiff.Expected, ""),
					Actual:        strings.Join(currDiff.Actual, ""),
					InImmutable:   currDiff.InImmutable,
					ImmutableName: currDiff.ImmutableName,
				})
				// Start new diff block
				currDiff = &currentDiff{
					Start:         i,
					Expected:      []string{expectedHex},
					Actual:        []string{actualHex},
					InImmutable:   inImmutableRange,
					ImmutableName: immName,
				}
			} else {
				// Extend the current difference block (same immutable status/name)
				currDiff.Expected = append(currDiff.Expected, expectedHex)
				currDiff.Actual = append(currDiff.Actual, actualHex)
			}
		} else { // Bytes match
			if currDiff != nil {
				// End the current difference block as the mismatch ended
				differences = append(differences, BytecodeDifference{
					Start:         currDiff.Start,
					Length:        len(currDiff.Expected),
					Expected:      strings.Join(currDiff.Expected, ""),
					Actual:        strings.Join(currDiff.Actual, ""),
					InImmutable:   currDiff.InImmutable,
					ImmutableName: currDiff.ImmutableName,
				})
				currDiff = nil // Reset tracker
			}
			// No action needed if bytes match and not in a diff block
		}
	}

	// Record the final difference block if the loop ended while in a diff
	if currDiff != nil {
		differences = append(differences, BytecodeDifference{
			Start:         currDiff.Start,
			Length:        len(currDiff.Expected),
			Expected:      strings.Join(currDiff.Expected, ""),
			Actual:        strings.Join(currDiff.Actual, ""),
			InImmutable:   currDiff.InImmutable,
			ImmutableName: currDiff.ImmutableName,
		})
	}

	// Collect and return immutable differences
	immutableValues := []ImmutableValueInfo{}
	if checkImmutables && artifact != nil && artifact.DeployedBytecode.ImmutableReferences != nil {
		// Iterate through the defined locations in the artifact
		for refKey, locations := range artifact.DeployedBytecode.ImmutableReferences {
			name := getImmutableName(artifact, refKey)
			if name == "" {
				name = fmt.Sprintf("immutable(id:%s)", refKey) // Fallback
			}

			for _, loc := range locations {
				start := int(loc.Start)
				length := int(loc.Length)
				if length <= 0 {
					// Invalid length - return an error
					return nil, nil, fmt.Errorf(
						"immutable '%s' location (offset %d, length %d) has invalid length %d",
						name, start, length, length,
					)
				}

				// Extract the actual value directly from actualBytes based on this location
				var actualValueHex string
				upperBound := start + length
				if start >= 0 && upperBound <= len(actualBytes) {
					// Safely extract the slice
					actualValueBytes := actualBytes[start:upperBound]
					actualValueHex = "0x" + hex.EncodeToString(actualValueBytes)
				} else {
					// Fundamental mismatch - return an error
					return nil, nil, fmt.Errorf(
						"immutable '%s' location (offset %d, length %d) is out of bounds for actual bytecode length %d",
						name, start, length, len(actualBytes),
					)
				}

				immutableValues = append(immutableValues, ImmutableValueInfo{
					Name:   name,
					Offset: start,
					Length: length,
					Value:  actualValueHex,
				})
			}
		}
	}

	return differences, immutableValues, nil
}

// printResults formats and prints the outcomes of verification checks.
func printResults(results []*VerificationResult, verbose bool) {
	// Track if any *code* mismatches occurred
	overallSuccess := true

	for i, result := range results {
		if i > 0 {
			// Add spacing between results
			fmt.Println()
		}

		// Print header
		printResultHeader(result)

		// Handle and print process errors
		if result.ProcessError != nil {
			color.Red("  ERROR during verification: %v", result.ProcessError)
			overallSuccess = false
			continue
		}

		// Analyze differences
		codeDiffs, immDiffs := categorizeDifferences(result)

		// Print status and details
		if len(codeDiffs) > 0 {
			overallSuccess = false
			color.Red("  ✗ Verification FAILED: Found unexpected differences in code.")
			printCodeDifferences(codeDiffs)
			if verbose && (result.Type == DeployedContract || result.Type == Implementation || result.Type == OPContractsManager) && len(result.ImmutableInfos) > 0 {
				printImmutableDetails(result.ImmutableInfos, immDiffs, true) // Pass true for includeConsistencyWarning
			}
		} else if len(immDiffs) > 0 && (result.Type == DeployedContract || result.Type == Implementation || result.Type == OPContractsManager) {
			color.Green("  ✓ Verification successful (differences only in known immutable locations).")
			if verbose {
				printImmutableDetails(result.ImmutableInfos, immDiffs, true)
			}
		} else {
			// Exact match (or blueprint match where immutables aren't checked)
			color.Green("  ✓ Verification successful (exact match).")
			if verbose && (result.Type == DeployedContract || result.Type == Implementation || result.Type == OPContractsManager) && len(result.ImmutableInfos) > 0 {
				printImmutableDetails(result.ImmutableInfos, nil, verbose)
			}
		}
	}

	// Print overall summary
	fmt.Println()
	if overallSuccess {
		color.Green("OK")
	} else {
		color.Red("FAILED")
	}
}

// printResultHeader prints the title section for a single result.
func printResultHeader(result *VerificationResult) {
	title := ""
	switch result.Type {
	case OPContractsManager:
		title = fmt.Sprintf("Verifying %s: %s", result.Type, result.Address)
	case Implementation:
		title = fmt.Sprintf("Verifying %s (%s): %s", result.FieldName, result.ContractName, result.Address)
	case DeployedContract:
		title = fmt.Sprintf("Verifying %s: %s (%s)", result.Type, result.ContractName, result.Address)
	case Blueprint:
		title = fmt.Sprintf("Verifying %s (%s for %s): %s", result.Type, result.FieldName, result.TargetContract, result.Address)
	case SplitBlueprintPart1:
		title = fmt.Sprintf("Verifying %s (%s for %s): %s", result.Type, result.FieldName, result.TargetContract, result.Address)
	case SplitBlueprintPart2:
		title = fmt.Sprintf("Verifying %s (%s for %s): %s", result.Type, result.FieldName, result.TargetContract, result.Address)
	default:
		title = fmt.Sprintf("Verifying %s (%s): %s", result.Type, result.ContractName, result.Address)
	}

	color.Cyan(title)
	fmt.Printf("  Artifact: %s\n", result.ArtifactPath)
}

// categorizeDifferences separates differences into code/unknown and immutable.
func categorizeDifferences(result *VerificationResult) (codeDiffs, immutableDiffs []BytecodeDifference) {
	isDeployed := result.Type == DeployedContract || result.Type == Implementation || result.Type == OPContractsManager
	codeDiffs = make([]BytecodeDifference, 0)
	immutableDiffs = make([]BytecodeDifference, 0)
	for _, diff := range result.Differences {
		if isDeployed && diff.InImmutable {
			immutableDiffs = append(immutableDiffs, diff)
		} else {
			codeDiffs = append(codeDiffs, diff)
		}
	}
	return codeDiffs, immutableDiffs
}

// hasCodeDifferences checks if a result has any non-immutable differences.
func hasCodeDifferences(result *VerificationResult) bool {
	codeDiffs, _ := categorizeDifferences(result)
	return len(codeDiffs) > 0
}

// printCodeDifferences formats and prints code/unknown differences.
func printCodeDifferences(diffs []BytecodeDifference) {
	color.Set(color.FgRed)
	fmt.Println("  --- Code Differences Found ---")
	color.Unset()
	for _, diff := range diffs {
		endPos := diff.Start + diff.Length - 1
		color.Red("    Byte %d-%d (%d bytes):", diff.Start, endPos, diff.Length)
		fmt.Printf("      Expected: 0x%s\n", maybeTruncate(diff.Expected, 64))
		fmt.Printf("      Actual:   0x%s\n", maybeTruncate(diff.Actual, 64))
	}
}

// printImmutableDetails formats and prints immutable variable info.
func printImmutableDetails(infos []ImmutableValueInfo, diffs []BytecodeDifference, verbose bool) {
	if len(infos) == 0 {
		return
	}

	color.Cyan("  --- Immutable Reference Details ---")

	// Group infos by name for consistency check
	infosByName := make(map[string][]ImmutableValueInfo)
	var names []string
	for _, info := range infos {
		if _, exists := infosByName[info.Name]; !exists {
			names = append(names, info.Name) // Keep order of first appearance
		}
		infosByName[info.Name] = append(infosByName[info.Name], info)
	}

	for _, name := range names {
		locations := infosByName[name]
		color.Yellow("    Variable: %s", name)
		inconsistent := false
		firstValue := ""
		populatedCount := 0

		for i, loc := range locations {
			fmt.Printf("      [%d] Location: Offset %d, Length %d bytes\n", i, loc.Offset, loc.Length)
			if loc.Value != "" && loc.Value != "0x" {
				fmt.Printf("          Actual Value: %s\n", loc.Value)
				if populatedCount == 0 {
					firstValue = loc.Value
				} else if loc.Value != firstValue {
					inconsistent = true
				}
				populatedCount++
			} else {
				fmt.Printf("          Actual Value: (Not populated - check comparison logic or bytecode length)\n")
			}
		}

		if inconsistent {
			color.Red("        ! Consistency WARNING: Found differing values for '%s' across its locations.", name)
		} else if verbose && populatedCount > 1 {
			color.Green("        ✓ Consistency: All populated values for '%s' are identical.", name)
		} else if verbose && populatedCount == 0 && len(locations) > 0 {
			color.Yellow("        - Consistency: No values populated for '%s'.", name)
		}
	}
}

// maybeTruncate shortens a string if it exceeds maxLen.
func maybeTruncate(s string, maxLen int) string {
	if len(s) > maxLen && maxLen > 3 {
		return s[:maxLen-3] + "..."
	}
	return s
}

// resolvePath resolves a potentially relative path against a base directory.
func resolvePath(path, baseDir string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	absBaseDir := baseDir
	if !filepath.IsAbs(absBaseDir) {
		absBaseDir = filepath.Join(cwd, absBaseDir)
	}
	return filepath.Join(absBaseDir, path), nil
}

// max returns the greater of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// getImmutableName finds the human-readable name of an immutable variable within a ForgeArtifact's AST,
// given the reference key from the artifact's ImmutableReferences map.
// The refKey is usually a string representation of the variable's AST node ID (e.g., "36").
func getImmutableName(artifact *solc.ForgeArtifact, refKey string) string {
	if artifact == nil {
		fmt.Fprintln(os.Stderr, "Warning: Cannot get immutable name, artifact is nil")
		return ""
	}

	// Extract the numeric ID from the key.
	// Handles formats like "36" or "t_int256:36".
	parts := strings.Split(refKey, ":")
	idStr := parts[len(parts)-1] // Take the last part after splitting by ':'

	numericID, err := strconv.Atoi(idStr)
	if err != nil {
		// If the key itself wasn't purely numeric and splitting didn't help, try parsing the whole key.
		numericID, err = strconv.Atoi(refKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not parse numeric ID from immutable reference key '%s': %v\n", refKey, err)
			return "" // Cannot parse numeric ID, cannot search AST
		}
	}

	// Search the AST nodes recursively
	return findAstNodeNameByID(artifact.Ast.Nodes, numericID)
}

// findAstNodeNameByID recursively searches a slice of AST nodes for a node with the target ID
// and returns its name.
func findAstNodeNameByID(nodes []solc.AstNode, targetID int) string {
	for _, node := range nodes {
		// Check if the current node matches the target ID
		if node.Id == targetID {
			// Ensure the node is a variable declaration, as IDs can be reused for other node types
			if node.NodeType == "VariableDeclaration" && node.Name != "" {
				return node.Name // Found the name
			}
			// If the ID matches but it's not a VariableDeclaration or has no name,
			// we might still find the right node deeper, so we don't return early.
			// However, typically the ID in immutable references points directly to the VariableDeclaration.
		}

		// Recursively search within nested nodes
		// Common places for nested declarations or structures:
		// 1. Direct children (`node.Nodes`) - Covers ContractDefinition, StructDefinition, etc.
		if len(node.Nodes) > 0 {
			if name := findAstNodeNameByID(node.Nodes, targetID); name != "" {
				return name
			}
		}

		// 2. Function bodies (`node.Body.Statements`)
		if node.Body != nil && len(node.Body.Statements) > 0 {
			// Note: Immutables are state variables, usually not declared inside function bodies,
			// but searching here for completeness doesn't hurt.
			if name := findAstNodeNameByID(node.Body.Statements, targetID); name != "" {
				return name
			}
		}

		// 3. Blocks within control structures (If, For, While - less likely for immutables)
		if node.TrueBody != nil && len(node.TrueBody.Statements) > 0 {
			if name := findAstNodeNameByID(node.TrueBody.Statements, targetID); name != "" {
				return name
			}
		}
		if node.FalseBody != nil && len(node.FalseBody.Statements) > 0 {
			if name := findAstNodeNameByID(node.FalseBody.Statements, targetID); name != "" {
				return name
			}
		}
	}

	return "" // Not found in this slice or its children
}
