// Package main implements a CLI tool to verify deployed Ethereum contract bytecode
// against local build artifacts. It supports verifying single contracts, blueprints
// (ERC-5202), and the contracts managed by an OPContractsManager instance.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
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

	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/scripts/verify/verify-bytecode/bindings"
)

// ImmutableReference represents a single location within expected contract bytecode
// where an immutable variable is injected. The actual value is populated during comparison.
type ImmutableReference struct {
	Offset int    // Byte offset where the immutable value starts.
	Length int    // Length of the immutable value in bytes.
	Value  string // Actual value found at this location in the deployed bytecode (hex encoded).
}

// BytecodeDifference represents a contiguous block of differing bytes found during comparison.
type BytecodeDifference struct {
	Start         int    // Byte offset where the difference begins.
	Length        int    // Length of the differing block in bytes.
	Expected      string // Expected bytes (hex encoded).
	Actual        string // Actual bytes found onchain (hex encoded).
	InImmutable   bool   // True if this difference falls within a known immutable reference range.
	ImmutableName string // Name of the immutable variable if InImmutable is true.
}

// currentDiff is a temporary helper struct used internally by findDifferences
// to track an ongoing sequence of differing bytes during the comparison loop.
type currentDiff struct {
	Start         int      // Starting byte offset of the current difference block.
	Expected      []string // Accumulated expected hex bytes in the current block.
	Actual        []string // Accumulated actual hex bytes in the current block.
	InImmutable   bool     // True if the current block is within an immutable reference range.
	ImmutableName string   // Name of the immutable variable if InImmutable is true.
}

// defaultArtifactsDir is the default directory name expected to contain forge build artifacts.
const defaultArtifactsDir = "forge-artifacts"

// defaultOPCMArtifactFilename is the default path relative to the artifacts directory
// for the OPContractsManager contract artifact.
const defaultOPCMArtifactFilename = "OPContractsManager.sol/OPContractsManager.json"

// blueprintPreamble is the ERC-5202 preamble (0xFE71) followed by version (00).
const blueprintPreamble = "0xFE7100"

// maxInitCodeSize defines the maximum size in bytes for the init code (creation code)
// that can be stored in a single blueprint slot, according to the split blueprint standard.
// (24576 - 3 byte preamble).
const maxInitCodeSize = 24573

// implementationArtifactOverrides maps specific field names from the OPCM Implementations struct
// (as defined in the Go bindings) to their corresponding artifact file paths (relative to artifacts-dir).
// This is used ONLY when the default naming convention (FieldName ending in "Impl" -> "BaseName.sol/BaseName.json")
// does not apply.
var implementationArtifactOverrides = map[string]string{
	"OptimismPortalImpl": "OptimismPortal2.sol/OptimismPortal2.json",
}

// blueprintArtifactOverrides maps specific field names from the OPCM Blueprints struct
// (as defined in the Go bindings) to their corresponding artifact file paths (relative to artifacts-dir).
// This is used ONLY when the default naming convention (FieldName -> "FieldName.sol/FieldName.json"
// after removing trailing digits) does not apply.
var blueprintArtifactOverrides = map[string]string{
	"Proxy":                      "Proxy.sol/Proxy.json",
	"PermissionlessDisputeGame1": "FaultDisputeGame.sol/FaultDisputeGame.json",
	"PermissionlessDisputeGame2": "FaultDisputeGame.sol/FaultDisputeGame.json",
	"PermissionedDisputeGame1":   "PermissionedDisputeGame.sol/PermissionedDisputeGame.json",
	"PermissionedDisputeGame2":   "PermissionedDisputeGame.sol/PermissionedDisputeGame.json",
}

// trailingDigitsRegex is used to remove trailing digits (e.g., '1' or '2') from blueprint field names
// to infer the base contract name for finding artifacts (e.g., "PermissionedDisputeGame1" -> "PermissionedDisputeGame").
var trailingDigitsRegex = regexp.MustCompile(`\d+$`)

// main sets up the CLI application using urfave/cli/v2 and defines the available commands.
func main() {
	app := &cli.App{
		Name:  "verify-bytecode",
		Usage: "Verify onchain contract bytecode against build artifacts",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "rpc",
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
		},
		Commands: []*cli.Command{
			{
				Name:  "single",
				Usage: "Verify a single contract (compares deployed bytecode)",
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
					&cli.BoolFlag{
						Name:  "verbose",
						Usage: "Print detailed immutable diff information even on success",
						Value: false,
					},
				},
				Action: func(c *cli.Context) error {
					rpcURL := c.String("rpc")
					artifactsDir := c.String("artifacts-dir")
					address := c.String("address")
					artifactPath := c.String("artifact")
					verbose := c.Bool("verbose")

					// Resolve artifact path relative to artifacts-dir if not absolute
					if !filepath.IsAbs(artifactPath) {
						cwd, _ := os.Getwd()
						baseDir := artifactsDir
						if !filepath.IsAbs(baseDir) {
							baseDir = filepath.Join(cwd, baseDir)
						}
						artifactPath = filepath.Join(baseDir, artifactPath)
					}

					color.Cyan("Comparing contract at %s with artifact %s", address, artifactPath)
					err := verifyDeployedContract(address, artifactPath, rpcURL, verbose)
					if err != nil {
						// Error is already printed within verifyDeployedContract or its callees
						return cli.Exit("", 1) // Indicate failure to the shell
					}
					// Success message is printed within verifyDeployedContract
					return nil
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
					&cli.BoolFlag{
						Name:  "verbose",
						Usage: "Print detailed immutable diff information even on success",
						Value: false,
					},
				},
				Action: func(c *cli.Context) error {
					rpcURL := c.String("rpc")
					artifactsDir := c.String("artifacts-dir")
					opcmAddress := c.String("opcm-address")
					verbose := c.Bool("verbose")

					// Resolve base directory for artifacts
					cwd, _ := os.Getwd()
					baseDir := artifactsDir
					if !filepath.IsAbs(baseDir) {
						baseDir = filepath.Join(cwd, baseDir)
					}
					opcmArtifactPath := filepath.Join(baseDir, defaultOPCMArtifactFilename)

					err := runOPCMVerification(opcmAddress, opcmArtifactPath, rpcURL, baseDir, verbose)
					if err != nil {
						// Error is already printed within runOPCMVerification or its callees
						return cli.Exit("", 1) // Indicate failure to the shell
					}
					// Success message is printed within runOPCMVerification
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		// Errors should be printed by the Action handlers or cli library itself.
		// Exit with non-zero status if Run returns an error.
		os.Exit(1)
	}
}

// runOPCMVerification orchestrates the verification process for the OPContractsManager (OPCM)
// and all the implementation and blueprint contracts it references.
// It first verifies the OPCM contract itself, then queries the OPCM for implementation
// and blueprint addresses, resolves their artifact paths, and calls the appropriate
// verification function (verifyDeployedContract, verifyBlueprint, or verifySplitBlueprint) for each.
// It aggregates errors encountered during the process.
func runOPCMVerification(opcmAddressHex, opcmArtifactPath, rpcURL, artifactsBaseDir string, verbose bool) error {
	var combinedErr error
	opcmAddress := common.HexToAddress(opcmAddressHex)

	// --- Verify OPContractsManager itself ---
	color.Yellow("--- Verifying OPContractsManager ---")
	err := verifyDeployedContract(opcmAddressHex, opcmArtifactPath, rpcURL, verbose)
	if err != nil {
		err = fmt.Errorf("failed to verify OPContractsManager contract: %w", err)
		color.Red("Error: %v", err)
		combinedErr = errors.Join(combinedErr, err)
		// Continue verification even if OPCM fails, but report overall failure later.
	}

	// --- Set up Ethereum client and OPCM caller ---
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		err = fmt.Errorf("failed to connect to RPC at %s: %w", rpcURL, err)
		color.Red("Error: %v", err)
		return errors.Join(combinedErr, err) // Cannot proceed without client
	}
	defer client.Close()

	opcmCaller, err := bindings.NewOpcm200Caller(opcmAddress, client)
	if err != nil {
		err = fmt.Errorf("failed to bind Opcm200 caller to address %s: %w", opcmAddressHex, err)
		color.Red("Error: %v", err)
		return errors.Join(combinedErr, err) // Cannot proceed without caller
	}

	// --- Verify Implementations ---
	color.Yellow("\n--- Verifying Implementations ---")
	implementationsResult, err := opcmCaller.Implementations(nil)
	if err != nil {
		err = fmt.Errorf("failed to call implementations() on OPCM contract %s: %w", opcmAddressHex, err)
		color.Red("Error: %v", err)
		combinedErr = errors.Join(combinedErr, err)
	} else {
		if verbose {
			color.Green("✓ Successfully retrieved implementation addresses.")
		}
		implValue := reflect.ValueOf(implementationsResult)
		implType := implValue.Type()

		// Iterate through the fields of the Implementations struct
		for i := 0; i < implValue.NumField(); i++ {
			fieldName := implType.Field(i).Name
			fieldValue := implValue.Field(i).Interface().(common.Address)
			implAddressStr := fieldValue.Hex()

			// Skip zero addresses
			if fieldValue == (common.Address{}) {
				if verbose {
					color.Yellow("  Skipping zero address for implementation: %s", fieldName)
				}
				continue
			}

			// Determine the artifact path for this implementation
			var relativePath string
			var ok bool
			if relativePath, ok = implementationArtifactOverrides[fieldName]; !ok {
				// Apply default naming convention if no override exists
				if strings.HasSuffix(fieldName, "Impl") {
					baseName := strings.TrimSuffix(fieldName, "Impl")
					relativePath = filepath.Join(fmt.Sprintf("%s.sol", baseName), fmt.Sprintf("%s.json", baseName))
				} else {
					// Error if convention doesn't apply and no override exists
					err = fmt.Errorf("cannot infer artifact path for implementation field '%s' (doesn't end in Impl) and no override exists", fieldName)
					color.Red("Error: %v", err)
					combinedErr = errors.Join(combinedErr, err)
					continue // Skip verification for this implementation
				}
			}

			artifactPath := filepath.Join(artifactsBaseDir, relativePath)
			// Verify the implementation contract
			err := verifyDeployedContract(implAddressStr, artifactPath, rpcURL, verbose)
			if err != nil {
				// Combine errors for overall reporting
				combinedErr = errors.Join(combinedErr, fmt.Errorf("implementation %s (%s): %w", fieldName, implAddressStr, err))
			}
		}
	}

	// --- Verify Blueprints ---
	color.Yellow("\n--- Verifying Blueprints ---")
	blueprintsResult, err := opcmCaller.Blueprints(nil)
	if err != nil {
		err = fmt.Errorf("failed to call blueprints() on OPCM contract %s: %w", opcmAddressHex, err)
		color.Red("Error: %v", err)
		combinedErr = errors.Join(combinedErr, err)
	} else {
		if verbose {
			color.Green("✓ Successfully retrieved blueprint addresses.")
		}
		blueprintValue := reflect.ValueOf(blueprintsResult)
		blueprintType := blueprintValue.Type()
		// Store blueprint fields for easy lookup (needed for split blueprint check)
		blueprintFields := make(map[string]common.Address)
		processedPart2 := make(map[string]bool) // Track part 2 blueprints already handled

		for i := 0; i < blueprintValue.NumField(); i++ {
			fieldName := blueprintType.Field(i).Name
			fieldValue := blueprintValue.Field(i).Interface().(common.Address)
			blueprintFields[fieldName] = fieldValue
		}

		// Iterate through the fields of the Blueprints struct again for verification
		for i := 0; i < blueprintValue.NumField(); i++ {
			fieldName := blueprintType.Field(i).Name
			fieldValue := blueprintValue.Field(i).Interface().(common.Address)
			blueprintAddressStr := fieldValue.Hex()

			// Skip if this field was already processed as part 2 of a split blueprint
			if processedPart2[fieldName] {
				continue
			}

			// Skip zero addresses
			if fieldValue == (common.Address{}) {
				if verbose {
					color.Yellow("  Skipping zero address for blueprint: %s", fieldName)
				}
				continue
			}

			// Determine the artifact path for this blueprint's target contract
			var relativePath string
			var baseName string // Base contract name inferred from field or override
			var ok bool
			if relativePath, ok = blueprintArtifactOverrides[fieldName]; !ok {
				// Apply default naming convention if no override exists
				baseName = trailingDigitsRegex.ReplaceAllString(fieldName, "")
				if baseName == "" {
					err = fmt.Errorf("cannot infer artifact path for blueprint field '%s' (empty after removing digits) and no override exists", fieldName)
					color.Red("Error: %v", err)
					combinedErr = errors.Join(combinedErr, err)
					continue // Skip verification for this blueprint
				}
				relativePath = filepath.Join(fmt.Sprintf("%s.sol", baseName), fmt.Sprintf("%s.json", baseName))
			} else {
				// If override exists, try to infer baseName from it, otherwise use field name
				parts := strings.Split(filepath.ToSlash(relativePath), "/")
				if len(parts) == 2 && strings.HasSuffix(parts[0], ".sol") && strings.HasSuffix(parts[1], ".json") {
					baseName = strings.TrimSuffix(parts[1], ".json")
				} else {
					// Fallback if override path doesn't match expected pattern
					baseName = trailingDigitsRegex.ReplaceAllString(fieldName, "")
				}
			}

			artifactPath := filepath.Join(artifactsBaseDir, relativePath)

			// Check if this is part 1 of a split blueprint
			if strings.HasSuffix(fieldName, "1") {
				part2FieldName := strings.TrimSuffix(fieldName, "1") + "2"
				if part2Addr, exists := blueprintFields[part2FieldName]; exists && part2Addr != (common.Address{}) {
					// If corresponding part 2 exists and is non-zero, verify as split blueprint
					err := verifySplitBlueprint(fieldName, blueprintAddressStr, part2FieldName, part2Addr.Hex(), artifactPath, rpcURL, verbose)
					if err != nil {
						combinedErr = errors.Join(combinedErr, fmt.Errorf("split blueprint %s/%s: %w", fieldName, part2FieldName, err))
					}
					processedPart2[part2FieldName] = true // Mark part 2 as handled
					continue                              // Move to next field
				}
				// Warn if part 1 exists but part 2 doesn't (or is zero address)
				color.Yellow("Warning: Found blueprint %s ending in '1' but no valid corresponding '%s' found. Verifying as single blueprint.", fieldName, part2FieldName)
			}

			// Verify as a standard (single) blueprint
			err := verifyBlueprint(fieldName, blueprintAddressStr, artifactPath, rpcURL, verbose)
			if err != nil {
				combinedErr = errors.Join(combinedErr, fmt.Errorf("blueprint %s (%s): %w", fieldName, blueprintAddressStr, err))
			}
		}
	}

	// Return combined error if any verification step failed
	if combinedErr != nil {
		return fmt.Errorf("one or more OPCM verification steps failed")
	}

	return nil
}

// verifyDeployedContract performs bytecode verification for a standard deployed contract.
// It loads the artifact, extracts the expected deployed bytecode and immutable references,
// fetches the actual bytecode from the chain, compares them, and prints the results.
// It handles differences within immutable variable locations specifically.
func verifyDeployedContract(address, artifactPath, rpcURL string, verbose bool) error {
	fmt.Println() // Add spacing for readability
	contractName := strings.TrimSuffix(filepath.Base(artifactPath), ".json")
	color.Cyan("Verifying %s (%s)", contractName, common.HexToAddress(address).String())

	// Load artifact JSON
	artifact, err := loadArtifact(artifactPath)
	if err != nil {
		color.Red("  Error loading artifact: %v", err)
		return err
	}

	// Extract expected deployed bytecode from artifact
	expectedBytecode, err := getDeployedBytecode(artifact)
	if err != nil {
		color.Red("  Error getting deployed bytecode from %s: %v", artifactPath, err)
		return err
	}

	// Extract immutable reference locations from artifact
	immutableRefs, err := getImmutableReferences(artifact)
	if err != nil {
		// Log error but continue comparison, treating differences as code errors
		color.Red("  Error getting immutable references from %s: %v", artifactPath, err)
		immutableRefs = nil // Ensure it's nil so findDifferences doesn't use partial data
	}

	// Fetch actual bytecode from the blockchain
	actualBytecode, err := getOnchainBytecode(address, rpcURL)
	if err != nil {
		color.Red("  Error getting onchain bytecode for %s: %v", address, err)
		return err
	}

	// Compare expected and actual bytecode, considering immutables
	differences, err := findDifferences(expectedBytecode, actualBytecode, immutableRefs)
	if err != nil {
		color.Red("  Error comparing bytecode for %s: %v", address, err)
		return err
	}

	// Print the comparison results (summary, errors, immutable details)
	printDifferences(differences, immutableRefs, verbose)

	// Determine if verification failed due to non-immutable differences
	hasCodeDifferences := false
	for _, diff := range differences {
		if !diff.InImmutable {
			hasCodeDifferences = true
			break
		}
	}

	if hasCodeDifferences {
		color.Red("  ✗ Verification FAILED for %s: Found unexpected differences in code.", address)
		return fmt.Errorf("bytecode mismatch for %s", address)
	} else {
		successMsg := fmt.Sprintf("  ✓ Verification successful")
		if len(differences) > 0 {
			successMsg += " (differences only in immutables)"
		} else {
			successMsg += " (exact match)"
		}
		color.Green(successMsg)
	}

	return nil
}

// verifyBlueprint performs bytecode verification for an ERC-5202 blueprint contract.
// It loads the target contract's artifact, extracts its creation code (initcode),
// prepends the ERC-5202 preamble, fetches the blueprint's bytecode from the chain,
// compares them, and prints the results. Immutable references are not considered for blueprints.
func verifyBlueprint(fieldName, address, targetArtifactPath, rpcURL string, verbose bool) error {
	fmt.Println() // Add spacing for readability
	targetContractName := strings.TrimSuffix(filepath.Base(targetArtifactPath), ".json")
	color.Cyan("Verifying blueprint %s (for %s) at %s", fieldName, targetContractName, common.HexToAddress(address).String())

	// Load the artifact of the contract the blueprint creates
	artifact, err := loadArtifact(targetArtifactPath)
	if err != nil {
		color.Red("  Error loading target artifact: %v", err)
		return err
	}

	// Extract the creation code (initcode) from the artifact
	creationCode, err := getCreationBytecode(artifact)
	if err != nil {
		color.Red("  Error getting creation code from %s: %v", targetArtifactPath, err)
		return err
	}

	// Construct the expected blueprint bytecode (Preamble + Creation Code)
	expectedBlueprintBytecode := blueprintPreamble + strings.TrimPrefix(creationCode, "0x")

	// Fetch the actual bytecode stored at the blueprint address
	actualBytecode, err := getOnchainBytecode(address, rpcURL)
	if err != nil {
		color.Red("  Error getting onchain bytecode for blueprint %s: %v", address, err)
		// Attempt comparison even if fetch failed, might compare against "0x"
		cmpErr := compareBlueprintCode(address, fieldName, expectedBlueprintBytecode, actualBytecode, verbose)
		return errors.Join(err, cmpErr) // Return both fetch and compare errors if any
	}

	// Compare expected blueprint bytecode with actual onchain bytecode
	return compareBlueprintCode(address, fieldName, expectedBlueprintBytecode, actualBytecode, verbose)
}

// verifySplitBlueprint verifies a blueprint that has been split into two parts due to size limits.
// It loads the target contract's artifact, extracts the full creation code, splits it into two parts
// based on maxInitCodeSize, prepends the preamble to each part, fetches the bytecode for both
// blueprint addresses (part 1 and part 2), and compares each part individually.
func verifySplitBlueprint(fieldName1, address1, fieldName2, address2, targetArtifactPath, rpcURL string, verbose bool) error {
	fmt.Println() // Add spacing for readability
	targetContractName := strings.TrimSuffix(filepath.Base(targetArtifactPath), ".json")
	color.Cyan("Verifying split blueprint %s/%s (for %s)",
		fieldName1, fieldName2, targetContractName)
	color.Cyan("  Part 1 Address: %s", common.HexToAddress(address1).String())
	color.Cyan("  Part 2 Address: %s", common.HexToAddress(address2).String())

	// Load the artifact of the contract the blueprint creates
	artifact, err := loadArtifact(targetArtifactPath)
	if err != nil {
		color.Red("  Error loading target artifact: %v", err)
		return err
	}

	// Extract the full creation code (initcode) from the artifact
	fullCreationCodeHex, err := getCreationBytecode(artifact)
	if err != nil {
		color.Red("  Error getting full creation code from %s: %v", targetArtifactPath, err)
		return err
	}
	fullCreationCodeHex = strings.TrimPrefix(fullCreationCodeHex, "0x")

	// Decode the creation code from hex to bytes
	fullCreationCodeBytes, err := hex.DecodeString(fullCreationCodeHex)
	if err != nil {
		return fmt.Errorf("failed to decode creation code from artifact %s: %w", targetArtifactPath, err)
	}

	// Split the creation code into two parts
	part1Bytes := fullCreationCodeBytes
	var part2Bytes []byte
	if len(fullCreationCodeBytes) > maxInitCodeSize {
		part1Bytes = fullCreationCodeBytes[:maxInitCodeSize]
		part2Bytes = fullCreationCodeBytes[maxInitCodeSize:]
	} else {
		// Warn if it was expected to be split but wasn't large enough
		color.Yellow("  Warning: Expected split blueprint %s/%s, but total initcode size (%d bytes) <= max size (%d bytes)",
			fieldName1, fieldName2, len(fullCreationCodeBytes), maxInitCodeSize)
		part2Bytes = []byte{} // Part 2 should be empty in this case
	}

	// Construct expected bytecode for each part (Preamble + Part Code)
	expectedBytecode1 := blueprintPreamble + hex.EncodeToString(part1Bytes)
	expectedBytecode2 := blueprintPreamble + hex.EncodeToString(part2Bytes)

	// Fetch actual bytecode for both blueprint addresses
	actualBytecode1, err1 := getOnchainBytecode(address1, rpcURL)
	actualBytecode2, err2 := getOnchainBytecode(address2, rpcURL)

	// Verify Part 1
	color.Cyan("  Verifying part 1 (%s)", fieldName1)
	errPart1 := compareBlueprintCode(address1, fieldName1, expectedBytecode1, actualBytecode1, verbose)
	if err1 != nil {
		// Combine fetch error with comparison error if any
		errPart1 = errors.Join(errPart1, fmt.Errorf("failed to get onchain bytecode for %s: %w", address1, err1))
		color.Red("    Error getting onchain bytecode for part 1: %v", err1)
	}

	// Verify Part 2
	color.Cyan("  Verifying part 2 (%s)", fieldName2)
	errPart2 := compareBlueprintCode(address2, fieldName2, expectedBytecode2, actualBytecode2, verbose)
	if err2 != nil {
		// Combine fetch error with comparison error if any
		errPart2 = errors.Join(errPart2, fmt.Errorf("failed to get onchain bytecode for %s: %w", address2, err2))
		color.Red("    Error getting onchain bytecode for part 2: %v", err2)
	}

	// Return combined errors from both parts
	return errors.Join(errPart1, errPart2)
}

// compareBlueprintCode performs the direct bytecode comparison for a single blueprint part.
// It compares the expected bytecode (preamble + creation code fragment) with the actual
// bytecode fetched from the chain for the given blueprint address. It prints success or failure messages.
// Immutable references are ignored in this comparison.
func compareBlueprintCode(address, fieldName, expectedBytecode, actualBytecode string, verbose bool) error {
	expectedClean := strings.ToLower(strings.TrimPrefix(expectedBytecode, "0x"))
	actualClean := strings.ToLower(strings.TrimPrefix(actualBytecode, "0x"))

	// Compare the expected and actual bytecode
	if expectedClean == actualClean {
		color.Green("    ✓ Verification successful (exact match)")
		return nil
	} else {
		color.Red("    ✗ Verification FAILED: Bytecode mismatch for blueprint %s (%s)", fieldName, address)
		// Use findDifferences to show where the mismatch occurs, ignoring immutables (nil map)
		differences, diffErr := findDifferences(expectedBytecode, actualBytecode, nil)
		if diffErr == nil && len(differences) > 0 {
			// Print the first block of differences found
			diff := differences[0]
			endPos := diff.Start + diff.Length - 1
			color.Red("      Difference found at byte %d-%d (%d bytes):", diff.Start, endPos, diff.Length)

			// Limit printed length for readability
			maxLen := 50 // Print up to 50 hex chars (25 bytes)
			expectedPrint := diff.Expected
			actualPrint := diff.Actual
			if len(expectedPrint) > maxLen {
				expectedPrint = expectedPrint[:maxLen] + "..."
			}
			if len(actualPrint) > maxLen {
				actualPrint = actualPrint[:maxLen] + "..."
			}
			fmt.Printf("        Expected: 0x%s\n", expectedPrint)
			fmt.Printf("        Actual:   0x%s\n", actualPrint)

			// If verbose, show full expected/actual (truncated)
			if verbose {
				maxLenTotal := 100
				expectedSuffix := ""
				actualSuffix := ""
				expectedTotalPrint := expectedBytecode
				actualTotalPrint := actualBytecode
				if len(expectedTotalPrint) > maxLenTotal {
					expectedTotalPrint = expectedTotalPrint[:maxLenTotal]
					expectedSuffix = "..."
				}
				if len(actualTotalPrint) > maxLenTotal {
					actualTotalPrint = actualTotalPrint[:maxLenTotal]
					actualSuffix = "..."
				}
				fmt.Printf("      Expected Full (start): %s%s\n", expectedTotalPrint, expectedSuffix)
				fmt.Printf("      Actual Full (start):   %s%s\n", actualTotalPrint, actualSuffix)
			}

		} else {
			// Fallback if findDifferences fails or finds no diffs (e.g., length mismatch only)
			color.Red("      Comparison failed, unable to generate detailed diff: %v", diffErr)
			fmt.Printf("      Expected Length: %d bytes\n", len(expectedClean)/2)
			fmt.Printf("      Actual Length:   %d bytes\n", len(actualClean)/2)
			if verbose {
				maxLenTotal := 100
				expectedSuffix := ""
				actualSuffix := ""
				expectedTotalPrint := expectedBytecode
				actualTotalPrint := actualBytecode
				if len(expectedTotalPrint) > maxLenTotal {
					expectedTotalPrint = expectedTotalPrint[:maxLenTotal]
					expectedSuffix = "..."
				}
				if len(actualTotalPrint) > maxLenTotal {
					actualTotalPrint = actualTotalPrint[:maxLenTotal]
					actualSuffix = "..."
				}
				fmt.Printf("      Expected (start): %s%s\n", expectedTotalPrint, expectedSuffix)
				fmt.Printf("      Actual   (start): %s%s\n", actualTotalPrint, actualSuffix)
			}
		}
		return fmt.Errorf("blueprint bytecode mismatch for %s (%s)", fieldName, address)
	}
}

// --- Artifact and Bytecode Handling Helpers ---

// loadArtifact reads and parses a JSON artifact file from the given path.
func loadArtifact(path string) (map[string]any, error) {
	if path == "" {
		return nil, fmt.Errorf("artifact path is required")
	}
	// Check if file exists first for a clearer error message
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("artifact file not found: %s", path)
	} else if err != nil {
		return nil, fmt.Errorf("error checking artifact file %s: %w", path, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact file %s: %w", path, err)
	}

	var artifact map[string]any
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", path, err)
	}

	return artifact, nil
}

// getDeployedBytecode extracts the deployed bytecode string (runtime code) from a parsed artifact.
// It prioritizes `deployedBytecode.object` > `deployedBytecode` (string) > `bytecode.object` > `bytecode` (string).
func getDeployedBytecode(artifact map[string]any) (string, error) {
	// Try deployedBytecode.object first
	if deployedBytecodeMap, ok := artifact["deployedBytecode"].(map[string]any); ok {
		if object, ok := deployedBytecodeMap["object"].(string); ok && object != "" && object != "0x" {
			return object, nil
		}
	}
	// Try deployedBytecode (string) second
	if deployedBytecodeStr, ok := artifact["deployedBytecode"].(string); ok && deployedBytecodeStr != "" && deployedBytecodeStr != "0x" {
		return deployedBytecodeStr, nil
	}
	// Try bytecode.object third (fallback, sometimes used for libraries or older artifacts)
	if bytecodeMap, ok := artifact["bytecode"].(map[string]any); ok {
		if object, ok := bytecodeMap["object"].(string); ok && object != "" && object != "0x" {
			color.Yellow("  Warning: Using bytecode.object as deployed bytecode (deployedBytecode field missing/empty).")
			return object, nil
		}
	}
	// Try bytecode (string) fourth (fallback)
	if bytecodeStr, ok := artifact["bytecode"].(string); ok && bytecodeStr != "" && bytecodeStr != "0x" {
		color.Yellow("  Warning: Using bytecode string as deployed bytecode (deployedBytecode field missing/empty).")
		return bytecodeStr, nil
	}
	return "", fmt.Errorf("could not find non-empty deployedBytecode or bytecode in artifact")
}

// getCreationBytecode extracts the creation bytecode string (initcode) from a parsed artifact.
// It prioritizes `bytecode.object` > `bytecode` (string).
func getCreationBytecode(artifact map[string]any) (string, error) {
	// Try bytecode.object first
	if bytecodeMap, ok := artifact["bytecode"].(map[string]any); ok {
		if object, ok := bytecodeMap["object"].(string); ok && object != "" && object != "0x" {
			return object, nil
		}
	}
	// Try bytecode (string) second
	if bytecodeStr, ok := artifact["bytecode"].(string); ok && bytecodeStr != "" && bytecodeStr != "0x" {
		return bytecodeStr, nil
	}
	return "", fmt.Errorf("could not find non-empty bytecode.object or bytecode string in artifact")
}

// getVariableNameFromAST attempts to find the human-readable variable name corresponding
// to an immutable reference ID by searching the contract's AST (Abstract Syntax Tree)
// included in the artifact. Falls back to returning the original ID if not found.
func getVariableNameFromAST(artifact map[string]any, varID string) string {
	// Sometimes IDs have prefixes like "t_string_storage:", remove them.
	cleanID := varID
	if strings.Contains(varID, ":") {
		parts := strings.Split(varID, ":")
		cleanID = parts[len(parts)-1]
	}

	// Convert the numeric part of the ID to an integer for matching AST node IDs.
	idInt, err := strconv.Atoi(cleanID)
	if err != nil {
		// If conversion fails, return the original ID as the name.
		color.Yellow("  Warning: Could not parse integer ID from immutable reference '%s'. Using original ID.", varID)
		return varID
	}

	// Search the AST recursively.
	if ast, ok := artifact["ast"].(map[string]any); ok {
		name := findNodeName(ast, idInt)
		if name != "" {
			return name // Found the name.
		}
	}

	// If AST is missing or name not found, return the original ID.
	color.Yellow("  Warning: Could not find variable name for immutable ID %s (numeric: %d) in AST. Using original ID.", varID, idInt)
	return varID
}

// findNodeName is a recursive helper function to search the AST (represented as nested maps/slices)
// for a node with a specific `id` and return its associated `name`.
func findNodeName(node any, targetID int) string {
	switch n := node.(type) {
	case map[string]any:
		// Check if the current node has the target ID.
		if idFloat, ok := n["id"].(float64); ok && int(idFloat) == targetID {
			// If ID matches, try to find the name in 'name' or 'attributes.name'.
			if name, ok := n["name"].(string); ok && name != "" {
				return name
			}
			if attributes, ok := n["attributes"].(map[string]any); ok {
				if name, ok := attributes["name"].(string); ok && name != "" {
					return name
				}
			}
			// Optionally log the node type if ID matched but name wasn't found.
			// if nodeType, ok := n["nodeType"].(string); ok {
			//  	fmt.Printf("Debug: Found node %d, type %s, but no name\n", targetID, nodeType)
			// }
		}

		// Recursively search child nodes (values in the map).
		for _, value := range n {
			switch v := value.(type) {
			case map[string]any, []any: // Only recurse into nested maps or slices.
				result := findNodeName(v, targetID)
				if result != "" {
					return result // Found in child node.
				}
			}
		}
	case []any:
		// Recursively search items in the slice.
		for _, item := range n {
			switch i := item.(type) {
			case map[string]any, []any: // Only recurse into nested maps or slices.
				result := findNodeName(i, targetID)
				if result != "" {
					return result // Found in slice item.
				}
			}
		}
	}
	return "" // Not found in this branch.
}

// getImmutableReferences parses the `immutableReferences` section of a contract artifact
// (preferring the one under `deployedBytecode`) and constructs a map where keys are
// variable names (resolved via AST) and values are slices of `ImmutableReference` structs
// indicating the location(s) of each immutable variable in the bytecode.
func getImmutableReferences(artifact map[string]any) (map[string][]ImmutableReference, error) {
	references := make(map[string][]ImmutableReference)
	var immutableRefsData any

	// Prefer immutable references from deployedBytecode section if available.
	if deployedBytecode, ok := artifact["deployedBytecode"].(map[string]any); ok {
		if refs, ok := deployedBytecode["immutableReferences"]; ok {
			immutableRefsData = refs
		}
	}

	// Fallback to top-level immutableReferences if not found under deployedBytecode.
	if immutableRefsData == nil {
		if refs, ok := artifact["immutableReferences"]; ok {
			immutableRefsData = refs
		} else {
			// No immutable references found, return empty map.
			return references, nil
		}
	}

	// Ensure the data is in the expected map[string]any format.
	immutableRefsMap, ok := immutableRefsData.(map[string]any)
	if !ok {
		if immutableRefsData != nil {
			// Warn about unexpected format but return empty map.
			color.Yellow("  Warning: Unexpected type for immutableReferences data: %T. Skipping.", immutableRefsData)
		}
		return references, nil // Return empty map if format is wrong.
	}

	// Iterate over each variable ID found in the immutable references map.
	for varID, refs := range immutableRefsMap {
		// Resolve the human-readable variable name from the AST.
		varName := getVariableNameFromAST(artifact, varID)
		references[varName] = []ImmutableReference{} // Initialize slice for this variable.

		// Ensure the references for this variable are in the expected []any format.
		refsList, ok := refs.([]any)
		if !ok {
			color.Yellow("  Warning:  Expected list for immutable references of variable '%s' (ID: %s), got %T. Skipping.", varName, varID, refs)
			continue // Skip this variable if format is wrong.
		}

		// Iterate over each reference location for the current variable.
		for refIdx, ref := range refsList {
			var start, length int
			validFormat := false

			// Try parsing as map[string]any {"start": N, "length": M}
			if refMap, ok := ref.(map[string]any); ok {
				startVal, startOk := refMap["start"].(float64)
				lengthVal, lengthOk := refMap["length"].(float64)
				if startOk && lengthOk {
					start = int(startVal)
					length = int(lengthVal)
					validFormat = true
				} else {
					color.Yellow("  Warning: Missing or invalid 'start'/'length' in map-style immutable reference %d for variable '%s'. Skipping.", refIdx, varName)
					continue
				}
			}

			// Try parsing as []any {N, M} if map parsing failed
			if !validFormat {
				if refSlice, ok := ref.([]any); ok && len(refSlice) == 2 {
					startVal, startOk := refSlice[0].(float64)
					lengthVal, lengthOk := refSlice[1].(float64)
					if startOk && lengthOk {
						start = int(startVal)
						length = int(lengthVal)
						validFormat = true
					} else {
						color.Yellow("  Warning: Invalid numeric types in slice-style immutable reference %d for variable '%s'. Skipping.", refIdx, varName)
						continue
					}
				}
			}

			// If neither format matched, issue warning and skip
			if !validFormat {
				color.Yellow("  Warning: Unrecognized immutable reference format at index %d for variable '%s': %T. Skipping.", refIdx, varName, ref)
				continue // Skip this specific reference location.
			}

			// Basic validation for length.
			if length <= 0 {
				color.Yellow("  Warning: Invalid length %d in immutable reference %d for variable '%s'. Skipping reference.", length, refIdx, varName)
				continue
			}

			// Add the valid reference location to the list for this variable.
			references[varName] = append(references[varName], ImmutableReference{
				Offset: start,
				Length: length,
				Value:  "", // Value will be populated later during comparison.
			})
		}

		// If, after processing, a variable has no valid references, remove it from the map.
		if len(references[varName]) == 0 {
			delete(references, varName)
		}
	}

	return references, nil
}

// getOnchainBytecode connects to the specified RPC URL and fetches the bytecode
// deployed at the given contract address. Returns the bytecode as a hex string ("0x...")
// or "0x" if no code exists at the address.
func getOnchainBytecode(address string, rpcURL string) (string, error) {
	if address == "" {
		return "", fmt.Errorf("contract address is required")
	}
	addr := common.HexToAddress(address)

	// Dial the RPC endpoint.
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return "", fmt.Errorf("failed to connect to RPC at %s: %w", rpcURL, err)
	}
	defer client.Close()

	// Fetch the code.
	code, err := client.CodeAt(context.Background(), addr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get code at address %s: %w", address, err)
	}

	// Handle case where contract exists but has no code (e.g., EOA or destroyed contract).
	if len(code) == 0 {
		color.Yellow("  Warning: No bytecode found at address %s on chain.", address)
		return "0x", nil // Return "0x" to represent empty code.
	}

	// Return hex-encoded bytecode string.
	return "0x" + hex.EncodeToString(code), nil
}

// --- Bytecode Comparison Logic ---

// isInImmutableReference checks if a given byte position falls within the range
// of any known immutable variable reference. If it does, it returns true,
// the variable name, and a pointer to the specific ImmutableReference struct.
func isInImmutableReference(
	position int,
	immutableRefs map[string][]ImmutableReference, // Assumed to be non-nil if called
) (bool, string, *ImmutableReference) {
	// Iterate through each variable and its reference locations.
	for varName, refs := range immutableRefs {
		// Check references in reverse order - might slightly optimize if overlaps exist,
		// though true overlaps shouldn't occur in valid compiler output.
		for i := len(refs) - 1; i >= 0; i-- {
			ref := &refs[i] // Get pointer to modify Value later.
			// Check if the position is within the [Offset, Offset + Length) range.
			if ref.Length > 0 && ref.Offset <= position && position < ref.Offset+ref.Length {
				return true, varName, ref
			}
		}
	}
	// Position does not fall within any immutable reference range.
	return false, "", nil
}

// findDifferences compares the expected bytecode (from artifact) with the actual bytecode
// (from chain), byte by byte. It identifies contiguous blocks of differing bytes
// and returns them as a slice of BytecodeDifference structs.
// It uses the `immutableRefs` map to classify differences that occur within immutable
// variable locations and populates the `Value` field of the corresponding `ImmutableReference` structs.
// Handles cases where bytecode lengths differ.
func findDifferences(
	expectedBytecode string,
	actualBytecode string,
	immutableRefs map[string][]ImmutableReference, // Map can be nil for blueprint checks where immutables are ignored.
) ([]BytecodeDifference, error) {
	// Normalize hex strings by removing "0x" prefix.
	expected := strings.TrimPrefix(expectedBytecode, "0x")
	actual := strings.TrimPrefix(actualBytecode, "0x")

	// Handle trivial case: both empty.
	if len(expected) == 0 && len(actual) == 0 {
		return []BytecodeDifference{}, nil
	}

	// Decode hex strings into byte slices.
	expectedBytes, err := hex.DecodeString(expected)
	if err != nil {
		return nil, fmt.Errorf("failed to decode expected bytecode: %w", err)
	}

	actualBytes, err := hex.DecodeString(actual)
	if err != nil {
		// Allow comparison to proceed if actual bytecode is empty or "0x".
		if actual == "" || actual == "0x" {
			actualBytes = []byte{} // Treat as empty byte slice.
		} else {
			return nil, fmt.Errorf("failed to decode actual bytecode: %w", err)
		}
	}

	// Warn if lengths differ, as comparison beyond the shorter length will show differences.
	if len(expectedBytes) != len(actualBytes) {
		color.Yellow("  Warning: Bytecode length mismatch. Expected: %d bytes, Actual: %d bytes.",
			len(expectedBytes), len(actualBytes))
	}

	// Determine the maximum length to iterate over.
	maxLength := len(expectedBytes)
	if len(actualBytes) > maxLength {
		maxLength = len(actualBytes)
	}

	// Reset collected values in immutableRefs before comparison.
	if immutableRefs != nil {
		for varName := range immutableRefs {
			refs := immutableRefs[varName]
			if refs != nil {
				for i := range refs {
					// Ensure Value is reset for each comparison run.
					refs[i].Value = ""
				}
			}
		}
	}

	differences := []BytecodeDifference{}
	var currDiff *currentDiff = nil // Tracks the current contiguous difference block.

	// Iterate through each byte position up to the maximum length.
	for i := 0; i < maxLength; i++ {
		inImmutable := false
		varName := ""
		var ref *ImmutableReference = nil
		// Check if this position is within an immutable reference, if applicable.
		if immutableRefs != nil {
			inImmutable, varName, ref = isInImmutableReference(i, immutableRefs)
		}

		// Get expected and actual bytes/hex strings, handling out-of-bounds access.
		var expectedByte byte = 0
		var actualByte byte = 0
		var expectedHex string = ".." // Placeholder for out-of-bounds bytes.
		var actualHex string = ".."   // Placeholder for out-of-bounds bytes.

		if i < len(expectedBytes) {
			expectedByte = expectedBytes[i]
			expectedHex = fmt.Sprintf("%02x", expectedByte)
		}
		if i < len(actualBytes) {
			actualByte = actualBytes[i]
			actualHex = fmt.Sprintf("%02x", actualByte)
		}

		bytesDiffer := expectedByte != actualByte

		// --- State machine logic for handling differences ---

		if inImmutable {
			// If inside an immutable, append the actual byte to its value regardless of diff.
			if i < len(actualBytes) {
				ref.Value += actualHex // Populate the value from actual bytecode.
			}

			if bytesDiffer {
				// Start or extend an immutable difference block.
				if currDiff == nil {
					// Start a new immutable diff block.
					currDiff = &currentDiff{Start: i, Expected: []string{expectedHex}, Actual: []string{actualHex}, InImmutable: true, ImmutableName: varName}
				} else if !currDiff.InImmutable {
					// End previous code diff block, start new immutable diff block.
					differences = append(differences, BytecodeDifference{currDiff.Start, len(currDiff.Expected), strings.Join(currDiff.Expected, ""), strings.Join(currDiff.Actual, ""), false, ""})
					currDiff = &currentDiff{Start: i, Expected: []string{expectedHex}, Actual: []string{actualHex}, InImmutable: true, ImmutableName: varName}
				} else if currDiff.ImmutableName != varName {
					// End previous immutable diff block (different variable), start new one.
					differences = append(differences, BytecodeDifference{currDiff.Start, len(currDiff.Expected), strings.Join(currDiff.Expected, ""), strings.Join(currDiff.Actual, ""), true, currDiff.ImmutableName})
					currDiff = &currentDiff{Start: i, Expected: []string{expectedHex}, Actual: []string{actualHex}, InImmutable: true, ImmutableName: varName}
				} else {
					// Extend the current immutable diff block.
					currDiff.Expected = append(currDiff.Expected, expectedHex)
					currDiff.Actual = append(currDiff.Actual, actualHex)
				}
			} else { // Bytes match within an immutable range.
				// If we were tracking an immutable diff, end it now.
				if currDiff != nil && currDiff.InImmutable {
					differences = append(differences, BytecodeDifference{currDiff.Start, len(currDiff.Expected), strings.Join(currDiff.Expected, ""), strings.Join(currDiff.Actual, ""), true, currDiff.ImmutableName})
					currDiff = nil // Reset tracker.
				}
				// Otherwise, do nothing (matching bytes in immutable range).
			}
		} else { // Not in an immutable reference range.
			if bytesDiffer {
				// Start or extend a code difference block.
				if currDiff == nil {
					// Start a new code diff block.
					currDiff = &currentDiff{Start: i, Expected: []string{expectedHex}, Actual: []string{actualHex}, InImmutable: false, ImmutableName: ""}
				} else if currDiff.InImmutable {
					// End previous immutable diff block, start new code diff block.
					differences = append(differences, BytecodeDifference{currDiff.Start, len(currDiff.Expected), strings.Join(currDiff.Expected, ""), strings.Join(currDiff.Actual, ""), true, currDiff.ImmutableName})
					currDiff = &currentDiff{Start: i, Expected: []string{expectedHex}, Actual: []string{actualHex}, InImmutable: false, ImmutableName: ""}
				} else {
					// Extend the current code diff block.
					currDiff.Expected = append(currDiff.Expected, expectedHex)
					currDiff.Actual = append(currDiff.Actual, actualHex)
				}
			} else { // Bytes match outside an immutable range.
				// If we were tracking any diff (code or immutable), end it now.
				if currDiff != nil {
					diffType := currDiff.InImmutable
					immName := currDiff.ImmutableName
					if !diffType {
						immName = "" // Ensure name is empty for code diffs
					}
					differences = append(differences, BytecodeDifference{currDiff.Start, len(currDiff.Expected), strings.Join(currDiff.Expected, ""), strings.Join(currDiff.Actual, ""), diffType, immName})
					currDiff = nil // Reset tracker.
				}
				// Otherwise, do nothing (matching bytes in code range).
			}
		}
	} // End of byte loop

	// If the loop finishes while tracking a difference, record the final block.
	if currDiff != nil {
		diffType := currDiff.InImmutable
		immName := currDiff.ImmutableName
		if !diffType {
			immName = ""
		}
		differences = append(differences, BytecodeDifference{currDiff.Start, len(currDiff.Expected), strings.Join(currDiff.Expected, ""), strings.Join(currDiff.Actual, ""), diffType, immName})
	}

	return differences, nil
}

// printDifferences formats and prints the results of the bytecode comparison.
// It provides a summary, details of any code differences (errors), warnings about
// inconsistent immutable values, and (if verbose or errors exist) a detailed breakdown
// of the values found for each immutable variable.
func printDifferences(
	differences []BytecodeDifference,
	immutableRefs map[string][]ImmutableReference, // Map can be nil for blueprint checks.
	verbose bool,
) {
	var nonImmutableDiffs []BytecodeDifference
	var immutableDiffs []BytecodeDifference

	// Separate differences into code/unknown and immutable categories.
	for _, diff := range differences {
		if diff.InImmutable {
			immutableDiffs = append(immutableDiffs, diff)
		} else {
			nonImmutableDiffs = append(nonImmutableDiffs, diff)
		}
	}

	hasCodeErrors := len(nonImmutableDiffs) > 0
	// Determine if detailed output sections should be printed.
	shouldPrintDetails := hasCodeErrors || verbose

	// --- Print Summary ---
	// Print summary only if details are needed or if there were any differences at all.
	if shouldPrintDetails {
		color.Cyan("\n  --- Comparison Summary ---")
		fmt.Printf("  Total difference blocks found: %d\n", len(differences))

		// Print summary breakdown only if printing details.
		if shouldPrintDetails {
			// Immutable differences count.
			immCount := len(immutableDiffs)
			if immutableRefs == nil { // Note if immutables weren't checked (e.g., blueprints)
				fmt.Printf("    - In immutable reference ranges: N/A (not checked)\n")
			} else if immCount > 0 {
				fmt.Printf("    - In immutable reference ranges: %d\n", immCount)
			} else {
				fmt.Printf("    - In immutable reference ranges: 0\n")
			}

			// Code/unknown differences count (colored based on presence of errors).
			if hasCodeErrors {
				color.Set(color.FgRed)
				fmt.Printf("    - In code / unknown areas:     %d <<<< CODE MISMATCH\n", len(nonImmutableDiffs))
				color.Unset()
			} else {
				fmt.Printf("    - In code / unknown areas:       0\n")
				color.Unset()
			}
		}
	} else if len(differences) == 0 {
		// If no differences and not verbose, print a simple confirmation.
		// (The main success message is printed by the calling verify function)
		// color.Green("  No differences found.") // Optional: Can be redundant.
	}
	// --- End Summary ---

	// --- Print Code Errors (Always print if they exist) ---
	if hasCodeErrors {
		color.Red("\n  === CODE ERRORS: Unexpected Differences Found ===")
		for _, diff := range nonImmutableDiffs {
			endPos := diff.Start + diff.Length - 1
			color.Red("  Byte %d-%d (%d bytes):", diff.Start, endPos, diff.Length)
			// Truncate long diffs for readability
			maxLen := 64 // Show max 32 bytes
			expectedPrint := diff.Expected
			actualPrint := diff.Actual
			if len(expectedPrint) > maxLen {
				expectedPrint = expectedPrint[:maxLen] + "..."
			}
			if len(actualPrint) > maxLen {
				actualPrint = actualPrint[:maxLen] + "..."
			}
			fmt.Printf("    Expected: 0x%s\n", expectedPrint)
			fmt.Printf("    Actual:   0x%s\n", actualPrint)
		}
	}
	// --- End Code Errors ---

	// --- Check Immutable Consistency (Always run if immutables were checked) ---
	var inconsistentImmutables []string
	if immutableRefs != nil { // Only perform check if immutables were processed.
		for varName, refs := range immutableRefs {
			if len(refs) <= 1 {
				continue // Need at least two reference points to check consistency.
			}
			firstValue := ""
			nonEmptyValueFound := false
			inconsistent := false
			// Iterate through all reference points for this variable.
			for _, ref := range refs {
				// Only compare non-empty values derived from the actual bytecode.
				if ref.Value != "" {
					if !nonEmptyValueFound {
						// Found the first non-empty value for comparison.
						firstValue = ref.Value
						nonEmptyValueFound = true
					} else if ref.Value != firstValue {
						// Found a subsequent reference with a different non-empty value.
						inconsistent = true
						break // Inconsistency found, no need to check further refs for this var.
					}
				}
			}
			if inconsistent {
				inconsistentImmutables = append(inconsistentImmutables, varName)
			}
		}

		// Print a warning section *if* any inconsistencies were found.
		if len(inconsistentImmutables) > 0 {
			color.Red("\n  === IMMUTABLE WARNING: Inconsistent Values Found ===")
			color.Yellow("      This means the same immutable variable seems to have different values")
			color.Yellow("      at different locations in the deployed bytecode. This is highly unusual.")
			for _, varName := range inconsistentImmutables {
				color.Red("    - Variable '%s' has differing values across its reference points.", varName)
			}
		}
	}
	// --- End Immutable Consistency Check ---

	// --- Print Full Immutable Details (Print only if verbose or code errors exist) ---
	// Print only if: details are requested AND immutables were checked AND there are references found.
	if shouldPrintDetails && immutableRefs != nil && len(immutableRefs) > 0 {
		color.Cyan("\n  === Immutable Reference Values (from Actual Bytecode) ===")
		var varNames []string
		for name := range immutableRefs {
			varNames = append(varNames, name)
		}
		// Consider sorting varNames for consistent output: sort.Strings(varNames)

		for _, varName := range varNames {
			refs := immutableRefs[varName]
			if len(refs) == 0 {
				continue // Should not happen if getImmutableReferences filters, but safeguard.
			}
			color.Yellow("\n  Variable: %s", varName)

			allPopulatedValuesSame := true // Assume consistency until proven otherwise.
			var firstPopulatedValue string = ""
			nonEmptyValueFound := false // Track if *any* value was populated for this var.
			hasMissingValue := false    // Track if *any* ref slot had no value populated.

			// Print details for each reference location of the current variable.
			for i, ref := range refs {
				fmt.Printf("    [%d] Artifact Location: Offset %d, Length %d bytes\n", i, ref.Offset, ref.Length)
				if ref.Value != "" {
					// Value was populated from actual bytecode at this location.
					fmt.Printf("        Actual Value Found: 0x%s\n", ref.Value)
					if !nonEmptyValueFound {
						firstPopulatedValue = ref.Value
						nonEmptyValueFound = true
					} else if ref.Value != firstPopulatedValue {
						// Mark inconsistency (warning already printed previously).
						allPopulatedValuesSame = false
					}
				} else {
					// No value was populated (likely actual bytecode was too short or comparison issue).
					fmt.Printf("        Actual Value Found: (Not present or comparison mismatch at this location)\n")
					hasMissingValue = true // Mark that at least one location had no value.
				}
			}

			// Print a summary line for this variable's consistency within the details section.
			if nonEmptyValueFound {
				if allPopulatedValuesSame {
					color.Green("      ✓ Consistency: All populated values for '%s' are identical.", varName)
				} else {
					// The main warning was printed earlier, just add context here.
					color.Red("      ! Consistency: Found differing values for '%s'. (See warning above)", varName)
				}
			} else if hasMissingValue {
				// Only report missing if *none* were populated but some locations existed.
				color.Yellow("      - Consistency: No values populated for '%s' (check bytecode length/offsets).", varName)
			} else if len(refs) > 0 {
				// Edge case: Refs exist, but none have values and none were marked missing. Unlikely.
				color.Yellow("      - Consistency: No values populated or missing for '%s'.", varName)
			}
		}
	}
	// --- End Full Immutable Details ---
}
