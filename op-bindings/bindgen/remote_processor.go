package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type remoteContractMetadata struct {
	Name            string
	Package         string
	DeployedBin     string
	InitBin         string
	DeploymentSalt  string
	DeployerAddress string
}

// writeContractMetadata generates and writes metadata files for remote contracts.
//
// This function takes a contractMetadata struct, the name of the contract, and a Go text template. It then:
// - Constructs the file path for the metadata file, naming it based on the contract name and placing it in the designated output directory.
// - Opens or creates a new file at this path with read/write permissions.
// - Executes the provided template with the contract metadata, writing the processed output to the file.
//
// Parameters:
// - contractMetaData: A struct holding the necessary metadata for a contract.
// - contractName: The name of the contract. Used to name the metadata file and as a part of the template execution.
// - fileTemplate: A pointer to a Go text template that defines the structure of the metadata file.
//
// Returns:
// - An error if it encounters issues with file operations or template execution.
func (gen *remoteBindingsGenerator) writeContractMetadata(contractMetaData remoteContractMetadata, contractName string, fileTemplate *template.Template) error {
	metadataFilePath := filepath.Join(gen.contractMetadataOutputDir, strings.ToLower(contractName)+"_more.go")
	metadataFile, err := os.OpenFile(
		metadataFilePath,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	if err != nil {
		return fmt.Errorf("error opening %s's metadata file at %s: %w", contractName, metadataFilePath, err)
	}
	defer metadataFile.Close()

	if err := fileTemplate.Execute(metadataFile, contractMetaData); err != nil {
		return fmt.Errorf("error writing %s's contract metadata at %s: %w", contractName, metadataFilePath, err)
	}

	gen.logger.Debug("Successfully wrote contract metadata", "contractName", contractName, "metadataFilePath", metadataFilePath)
	return nil
}

// checkDeployedBytecode compares the deployed bytecode from the source chain against the compare chain to validate
// the consistency of bytecode. If there is a mismatch of bytecode between the chains, there's a possibility that using
// the bytecode from the source chain may not work as intended on an OP chain. In most cases the compareChain will be an OP chain
// where the contract is known to have a successful deployment.
//
// The function performs the following steps:
// - Fetches the deployed bytecode of a contract from the comparison chain using the contract's address.
// - Compares this fetched bytecode with the previously obtained deployed bytecode the source chain.
// - If there's a discrepancy, the function decides which bytecode to use based on the contract's UseDeploymentBytecodeFromChainId.
//
// Parameters:
// - sourceAddress: The contract's address on the source chain.
// - compareAddress: The contract's address on the comparison chain.
// - contract: A pointer to a remoteContract struct, containing details about the contract, including its deployed bytecode.
//
// The function returns an error in the following scenarios:
// - If fetching the deployed bytecode from the comparison chain fails.
// - If the deployed bytecode differ and no explicit configuration is provided on which chain's bytecode to default to.
// - If the configuration specifies a chain ID that is not recognized.
func (gen *remoteBindingsGenerator) checkDeployedBytecode(sourceAddress, compareAddress string, contract *remoteContract) error {
	compareBytecode, err := gen.contractDataClient.FetchDeployedBytecode(gen.compareChainId, compareAddress)
	if err != nil {
		return err
	}

	if contract.DeployedBytecode != compareBytecode {
		if contract.UseDeploymentBytecodeFromChainId > 0 {
			switch contract.UseDeploymentBytecodeFromChainId {
			case gen.sourceChainId:
				// contract.DeployedBytecode already set, skip
			case gen.compareChainId:
				contract.DeployedBytecode = compareBytecode
			default:
				return fmt.Errorf("unknown chain %d, don't know what bytecode to default to", contract.UseDeploymentBytecodeFromChainId)
			}
			gen.logger.Warn(
				fmt.Sprintf("%s's deployed bytecode mismatch, using bytecode from chain: %d", contract.Name, contract.UseDeploymentBytecodeFromChainId),
				"sourceChainId", gen.sourceChainId,
				"sourceAddress", sourceAddress,
				"compareChainId", gen.compareChainId,
				"compareAddress", compareAddress,
			)
		} else {
			return fmt.Errorf(
				"%s's deployed bytecode mismatch between source chain %d at address: %s and compare chain %d at address: %s",
				contract.Name,
				gen.sourceChainId, sourceAddress,
				gen.compareChainId, compareAddress,
			)
		}
	}

	return nil
}

// checkInitBytecode compares the initialization bytecode from the source chain against the compare chain to validate
// the consistency of bytecode. If there is a mismatch of bytecode between the chains, there's a possibility that using
// the bytecode from the source chain may not work as intended on an OP chain. In most cases the compareChain will be an OP chain
// where the contract is known to have a successful deployment.
//
// Parameters:
// - sourceTxHash: The transaction hash on the source chain for the contract's deployment.
// - compareAddress: The contract's address on the comparison chain.
// - contract: A pointer to a remoteContract struct, containing details about the contract, including its initialization bytecode.
//
// The function returns an error in the following scenarios:
// - If fetching the deployment transaction hash or initialization bytecode from the comparison chain fails.
// - If the initialization bytecode differ and no explicit configuration is provided on which chain's bytecode to default to.
// - If the configuration specifies a chain ID that is not recognized.
func (gen *remoteBindingsGenerator) checkInitBytecode(sourceTxHash, compareAddress string, contract *remoteContract) error {
	compareTxHash, ok := contract.DeploymentTxHashes[fmt.Sprint(gen.compareChainId)]
	var err error
	if !ok {
		compareTxHash, err = gen.contractDataClient.FetchDeploymentTxHash(gen.compareChainId, compareAddress)
		if err != nil {
			return err
		}
	}

	compareBytecode, err := gen.contractDataClient.FetchDeploymentData(gen.compareChainId, compareTxHash)
	if err != nil {
		return err
	}

	if contract.InitBytecode != compareBytecode {
		if contract.UseInitBytecodeFromChainId > 0 {
			switch contract.UseInitBytecodeFromChainId {
			case gen.sourceChainId:
				// contract.InitBytecode already set, skip
			case gen.compareChainId:
				contract.InitBytecode = compareBytecode
			default:
				return fmt.Errorf("unknown chain %d, don't know what bytecode to default to", contract.UseInitBytecodeFromChainId)
			}
			gen.logger.Debug(
				fmt.Sprintf("%s's initialization bytecode mismatch, using bytecode from chain: %d", contract.Name, contract.UseInitBytecodeFromChainId),
				"sourceChainId", gen.sourceChainId,
				"sourceTxHash", sourceTxHash,
				"compareChainId", gen.compareChainId,
				"compareTxHash", compareTxHash,
			)
		} else {
			return fmt.Errorf(
				"%s's initialization bytecode mismatch between source chain %d at tx hash: %s and compare chain %d at tx hash: %s",
				contract.Name,
				gen.sourceChainId, sourceTxHash,
				gen.compareChainId, compareTxHash,
			)
		}
	}

	return nil
}

// fetchContractData retrieves the ABI, deployed bytecode, deployment transaction hash, and initialization bytecode for a contract.
// It also conducts comparisons of deployed and initialization bytecodes across different chains, if configured to do so.
//
// Parameters:
// - contract: A pointer to a remoteContract struct that contains the contract's metadata and where the fetched data will be stored.
//
// This function returns an error in several scenarios, such as:
// - Failure to find a deployment address or transaction hash on the source chain.
// - Errors in fetching ABI, deployed bytecode, or initialization bytecode from the blockchain.
// - Mismatches in deployed or initialization bytecode when comparisons are enabled.
// - Missing or incorrect salt in the initialization bytecode for proxy contracts.
func (gen *remoteBindingsGenerator) fetchContractData(contract *remoteContract) error {
	sourceAddress, ok := contract.Deployments[fmt.Sprint(gen.sourceChainId)]
	if !ok {
		return fmt.Errorf("no deployment address was found for %s on chain ID: %d", contract.Name, gen.sourceChainId)
	}

	var compareAddress string
	skipComparisons := false
	if gen.compareDeploymentBytecode || gen.compareInitBytecode {
		compareAddress, ok = contract.Deployments[fmt.Sprint(gen.compareChainId)]
		if !ok {
			gen.logger.Warn(fmt.Sprintf("No deployment address was found for %s for chain ID: %d, skipping bytecode comparisons", contract.Name, gen.compareChainId))
			skipComparisons = true
		}
	}

	var err error
	if contract.Verified {
		contract.Abi, err = gen.contractDataClient.FetchAbi(gen.sourceChainId, sourceAddress)
		if err != nil {
			return err
		}
	}

	if !contract.ManuallyResolveImmutables {
		contract.DeployedBytecode, err = gen.contractDataClient.FetchDeployedBytecode(gen.sourceChainId, sourceAddress)
		if err != nil {
			return err
		}

		if !skipComparisons && gen.compareDeploymentBytecode {
			if err = gen.checkDeployedBytecode(sourceAddress, compareAddress, contract); err != nil {
				return err
			}
		}
	}

	contract.DeploymentTxHashes = make(map[string]string)
	contract.DeploymentTxHashes[fmt.Sprint(gen.sourceChainId)], err = gen.contractDataClient.FetchDeploymentTxHash(gen.sourceChainId, sourceAddress)
	if err != nil {
		return err
	}
	sourceTxHash, ok := contract.DeploymentTxHashes[fmt.Sprint(gen.sourceChainId)]
	if !ok {
		return fmt.Errorf("no deployment tx hash was found for %s on chain ID: %d", contract.Name, gen.sourceChainId)
	}

	contract.InitBytecode, err = gen.contractDataClient.FetchDeploymentData(gen.sourceChainId, sourceTxHash)
	if err != nil {
		return err
	}

	if !skipComparisons && gen.compareInitBytecode {
		if err = gen.checkInitBytecode(sourceTxHash, compareAddress, contract); err != nil {
			return err
		}
	}

	if contract.Create2ProxyDeployed && contract.DeploymentSalt != "" {
		re := regexp.MustCompile(fmt.Sprintf("^0x(%s)", contract.DeploymentSalt))
		if !re.MatchString(contract.InitBytecode) {
			return fmt.Errorf("expected salt: %s to be at the beginning of the contract initialization code: %s, but it wasn't", contract.DeploymentSalt, contract.InitBytecode)
		}
		contract.InitBytecode = re.ReplaceAllString(contract.InitBytecode, "")
	}

	return nil
}

// processContracts iterates through a slice of remoteContract structs, generating Go bindings and metadata for each.
//
// Parameters:
// - contracts: A slice of remoteContract structs, each representing a contract for which bindings and metadata need to be generated.
//
// This method returns an error in various scenarios, including:
// - Failures in fetching contract data.
// - Issues in writing contract artifacts to files.
// - Errors during the Go bindings generation process.
// - Problems in writing contract metadata to files.
func (gen *remoteBindingsGenerator) processContracts(contracts []remoteContract) error {
	for _, contract := range contracts {
		gen.logger.Info("Generating bindings and metadata for remote contract", "contractName", contract.Name)

		if err := gen.fetchContractData(&contract); err != nil {
			return fmt.Errorf("error fetching contract data for %s: %w", contract.Name, err)
		}

		abiFilePath, bytecodeFilePath, err := writeContractArtifacts(gen.logger, gen.tempArtifactsDir, contract.Name, []byte(contract.Abi), []byte(contract.InitBytecode))
		if err != nil {
			return err
		}

		err = genContractBindings(gen.logger, abiFilePath, bytecodeFilePath, gen.bindingsPackageName, contract.Name)
		if err != nil {
			return err
		}

		contractMetaData := remoteContractMetadata{
			Name:    contract.Name,
			Package: gen.bindingsPackageName,
		}

		if !contract.ManuallyResolveImmutables {
			contractMetaData.DeployedBin = contract.DeployedBytecode
		} else {
			contractMetaData.InitBin = contract.InitBytecode
		}

		if contract.Create2ProxyDeployed {
			contractMetaData.DeployerAddress = contract.Create2DeployerAddress
			contractMetaData.DeploymentSalt = contract.DeploymentSalt
		}

		if contract.ManuallyResolveImmutables && contract.Create2ProxyDeployed {
			contractMetadataWithImmutablesFileTemplate := template.Must(template.New("contractMetadataWithImmutables").Parse(remoteContractMetadataImmutablesTemplate))
			if err := gen.writeContractMetadata(contractMetaData, contract.Name, contractMetadataWithImmutablesFileTemplate); err != nil {
				return err
			}
		} else {
			if err := gen.writeContractMetadata(contractMetaData, contract.Name, gen.contractMetadataFileTemplate); err != nil {
				return err
			}
		}
	}

	return nil
}
