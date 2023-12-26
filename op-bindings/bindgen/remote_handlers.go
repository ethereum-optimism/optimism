package bindgen

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/ethereum-optimism/optimism/op-bindings/etherscan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ContractData struct {
	Abi          string
	DeployedBin  string
	DeploymentTx etherscan.Transaction
}

func (generator *BindGenGeneratorRemote) standardHandler(contractMetadata *RemoteContractMetadata) error {
	fetchedData, err := generator.FetchContractData(contractMetadata.Verified, "eth", contractMetadata.Deployments.Eth.Hex())
	if err != nil {
		return err
	}

	contractMetadata.DeployedBin = fetchedData.DeployedBin
	if err = generator.CompareDeployedBytecodeWithRpc(contractMetadata, "eth"); err != nil {
		return err
	}
	if err = generator.CompareDeployedBytecodeWithRpc(contractMetadata, "op"); err != nil {
		return err
	}

	// If ABI was explicitly provided by config, don't overwrite
	if contractMetadata.ABI == "" {
		contractMetadata.ABI = fetchedData.Abi
	} else if fetchedData.Abi != "" && contractMetadata.ABI != fetchedData.Abi {
		generator.Logger.Debug("ABIs", "given", contractMetadata.ABI, "fetched", fetchedData.Abi)
		return fmt.Errorf("the given ABI for %s differs from what was fetched from Etherscan", contractMetadata.Name)
	}

	if contractMetadata.InitBin, err = generator.removeDeploymentSalt(fetchedData.DeploymentTx.Input, contractMetadata.DeploymentSalt); err != nil {
		return err
	}

	if err := generator.CompareInitBytecodeWithOp(contractMetadata, true); err != nil {
		return fmt.Errorf("%s: %w", contractMetadata.Name, err)
	}
	if err := generator.CompareDeployedBytecodeWithOp(contractMetadata, true); err != nil {
		return fmt.Errorf("%s: %w", contractMetadata.Name, err)
	}

	return generator.writeAllOutputs(contractMetadata, remoteContractMetadataTemplate)
}

func (generator *BindGenGeneratorRemote) create2DeployerHandler(contractMetadata *RemoteContractMetadata) error {
	fetchedData, err := generator.FetchContractData(contractMetadata.Verified, "eth", contractMetadata.Deployments.Eth.Hex())
	if err != nil {
		return err
	}

	contractMetadata.ABI = fetchedData.Abi
	contractMetadata.DeployedBin = fetchedData.DeployedBin
	if contractMetadata.InitBin, err = generator.removeDeploymentSalt(fetchedData.DeploymentTx.Input, contractMetadata.DeploymentSalt); err != nil {
		return err
	}

	// We're expecting the bytecode for Create2Deployer to not match the deployment on OP,
	// because we're predeploying a modified version of Create2Deployer that has not yet been
	// deployed to OP.
	// For context: https://github.com/ethereum-optimism/op-geth/pull/126
	if err := generator.CompareInitBytecodeWithOp(contractMetadata, false); err != nil {
		return fmt.Errorf("%s: %w", contractMetadata.Name, err)
	}
	if err := generator.CompareDeployedBytecodeWithOp(contractMetadata, false); err != nil {
		return fmt.Errorf("%s: %w", contractMetadata.Name, err)
	}

	return generator.writeAllOutputs(contractMetadata, remoteContractMetadataTemplate)
}

func (generator *BindGenGeneratorRemote) multiSendHandler(contractMetadata *RemoteContractMetadata) error {
	// MultiSend has an immutable that resolves to this(address).
	// Because we're predeploying MultiSend to the same address as on OP,
	// we can use the deployed bytecode directly for the predeploy
	fetchedData, err := generator.FetchContractData(contractMetadata.Verified, "op", contractMetadata.Deployments.Op.Hex())
	if err != nil {
		return err
	}

	contractMetadata.ABI = fetchedData.Abi
	contractMetadata.DeployedBin = fetchedData.DeployedBin
	if err = generator.CompareDeployedBytecodeWithRpc(contractMetadata, "op"); err != nil {
		return err
	}
	if contractMetadata.InitBin, err = generator.removeDeploymentSalt(fetchedData.DeploymentTx.Input, contractMetadata.DeploymentSalt); err != nil {
		return err
	}

	return generator.writeAllOutputs(contractMetadata, remoteContractMetadataTemplate)
}

func (generator *BindGenGeneratorRemote) senderCreatorHandler(contractMetadata *RemoteContractMetadata) error {
	var err error
	contractMetadata.DeployedBin, err = generator.ContractDataClients.Eth.FetchDeployedBytecode(context.Background(), contractMetadata.Deployments.Eth.Hex())
	if err != nil {
		return fmt.Errorf("error fetching deployed bytecode: %w", err)
	}
	if err = generator.CompareDeployedBytecodeWithRpc(contractMetadata, "eth"); err != nil {
		return err
	}
	if err = generator.CompareDeployedBytecodeWithRpc(contractMetadata, "op"); err != nil {
		return err
	}

	// The SenderCreator contract is deployed by EntryPoint, so the transaction data
	// from the deployment transaction is for the entire EntryPoint deployment.
	// So, we're manually providing the initialization bytecode and therefore it isn't being compared here
	if err := generator.CompareInitBytecodeWithOp(contractMetadata, false); err != nil {
		return fmt.Errorf("%s: %w", contractMetadata.Name, err)
	}
	if err := generator.CompareDeployedBytecodeWithOp(contractMetadata, true); err != nil {
		return fmt.Errorf("%s: %w", contractMetadata.Name, err)
	}

	return generator.writeAllOutputs(contractMetadata, remoteContractMetadataTemplate)
}

func (generator *BindGenGeneratorRemote) permit2Handler(contractMetadata *RemoteContractMetadata) error {
	fetchedData, err := generator.FetchContractData(contractMetadata.Verified, "eth", contractMetadata.Deployments.Eth.Hex())
	if err != nil {
		return err
	}

	contractMetadata.ABI = fetchedData.Abi
	contractMetadata.DeployedBin = fetchedData.DeployedBin
	if contractMetadata.InitBin, err = generator.removeDeploymentSalt(fetchedData.DeploymentTx.Input, contractMetadata.DeploymentSalt); err != nil {
		return err
	}

	if !strings.EqualFold(contractMetadata.Deployer.Hex(), fetchedData.DeploymentTx.To) {
		return fmt.Errorf(
			"expected deployer address: %s doesn't match the to address: %s for Permit2's proxy deployment transaction",
			contractMetadata.Deployer.Hex(),
			fetchedData.DeploymentTx.To,
		)
	}

	if err := generator.CompareInitBytecodeWithOp(contractMetadata, true); err != nil {
		return fmt.Errorf("%s: %w", contractMetadata.Name, err)
	}
	// We're asserting the deployed bytecode doesn't match, because Permit2 has immutable Solidity variables that
	// are dependent on block.chainid
	if err := generator.CompareDeployedBytecodeWithOp(contractMetadata, false); err != nil {
		return fmt.Errorf("%s: %w", contractMetadata.Name, err)
	}

	return generator.writeAllOutputs(contractMetadata, permit2MetadataTemplate)
}

func (generator *BindGenGeneratorRemote) FetchContractData(contractVerified bool, chain, deploymentAddress string) (ContractData, error) {
	var data ContractData
	var err error

	var client contractDataClient
	switch chain {
	case "eth":
		client = generator.ContractDataClients.Eth
	case "op":
		client = generator.ContractDataClients.Op
	default:
		return data, fmt.Errorf("unknown chain, unable to retrieve a contract data client for chain: %s", chain)
	}

	if contractVerified {
		data.Abi, err = client.FetchAbi(context.Background(), deploymentAddress)
		if err != nil {
			return ContractData{}, fmt.Errorf("error fetching ABI: %w", err)
		}
	}

	data.DeployedBin, err = client.FetchDeployedBytecode(context.Background(), deploymentAddress)
	if err != nil {
		return ContractData{}, fmt.Errorf("error fetching deployed bytecode: %w", err)
	}

	deploymentTxHash, err := client.FetchDeploymentTxHash(context.Background(), deploymentAddress)
	if err != nil {
		return ContractData{}, fmt.Errorf("error fetching deployment transaction hash: %w", err)
	}

	data.DeploymentTx, err = client.FetchDeploymentTx(context.Background(), deploymentTxHash)
	if err != nil {
		return ContractData{}, fmt.Errorf("error fetching deployment transaction data: %w", err)
	}

	return data, nil
}

func (generator *BindGenGeneratorRemote) removeDeploymentSalt(deploymentData, deploymentSalt string) (string, error) {
	if deploymentSalt == "" {
		return deploymentData, nil
	}

	re, err := regexp.Compile(fmt.Sprintf("^0x(%s)", deploymentSalt))
	if err != nil {
		return "", fmt.Errorf("failed to compile regular expression: %w", err)
	}
	if !re.MatchString(deploymentData) {
		return "", fmt.Errorf(
			"expected salt: %s to be at the beginning of the contract initialization code: %s, but it wasn't",
			deploymentSalt, deploymentData,
		)
	}
	return re.ReplaceAllString(deploymentData, ""), nil
}

func (generator *BindGenGeneratorRemote) CompareInitBytecodeWithOp(contractMetadataEth *RemoteContractMetadata, initCodeShouldMatch bool) error {
	if contractMetadataEth.InitBin == "" {
		return fmt.Errorf("no initialization bytecode provided for ETH deployment for comparison")
	}

	var zeroAddress common.Address
	if contractMetadataEth.Deployments.Op == zeroAddress {
		return fmt.Errorf("no deployment address on Optimism provided for %s", contractMetadataEth.Name)
	}

	// Passing false here, because true will retrieve contract's ABI, but we don't need it for bytecode comparison
	opContractData, err := generator.FetchContractData(false, "op", contractMetadataEth.Deployments.Op.Hex())
	if err != nil {
		return err
	}

	if opContractData.DeploymentTx.Input, err = generator.removeDeploymentSalt(opContractData.DeploymentTx.Input, contractMetadataEth.DeploymentSalt); err != nil {
		return err
	}

	initCodeComparison := strings.EqualFold(contractMetadataEth.InitBin, opContractData.DeploymentTx.Input)
	if initCodeShouldMatch && !initCodeComparison {
		return fmt.Errorf(
			"expected initialization bytecode to match on Ethereum and Optimism, but it doesn't. contract=%s bytecodeEth=%s bytecodeOp=%s",
			contractMetadataEth.Name,
			contractMetadataEth.InitBin,
			opContractData.DeploymentTx.Input,
		)
	} else if !initCodeShouldMatch && initCodeComparison {
		return fmt.Errorf(
			"expected initialization bytecode on Ethereum to not match on Optimism, but it did. contract=%s bytecodeEth=%s bytecodeOp=%s",
			contractMetadataEth.Name,
			contractMetadataEth.InitBin,
			opContractData.DeploymentTx.Input,
		)
	}

	return nil
}

func (generator *BindGenGeneratorRemote) CompareDeployedBytecodeWithOp(contractMetadataEth *RemoteContractMetadata, deployedCodeShouldMatch bool) error {
	if contractMetadataEth.DeployedBin == "" {
		return fmt.Errorf("no deployed bytecode provided for ETH deployment for comparison")
	}

	var zeroAddress common.Address
	if contractMetadataEth.Deployments.Op == zeroAddress {
		return fmt.Errorf("no deployment address on Optimism provided for %s", contractMetadataEth.Name)
	}

	// Passing false here, because true will retrieve contract's ABI, but we don't need it for bytecode comparison
	opContractData, err := generator.FetchContractData(false, "op", contractMetadataEth.Deployments.Op.Hex())
	if err != nil {
		return err
	}

	deployedCodeComparison := strings.EqualFold(contractMetadataEth.DeployedBin, opContractData.DeployedBin)
	if deployedCodeShouldMatch && !deployedCodeComparison {
		return fmt.Errorf(
			"expected deployed bytecode to match on Ethereum and Optimism, but it doesn't. contract=%s bytecodeEth=%s bytecodeOp=%s",
			contractMetadataEth.Name,
			contractMetadataEth.DeployedBin,
			opContractData.DeployedBin,
		)
	} else if !deployedCodeShouldMatch && deployedCodeComparison {
		return fmt.Errorf(
			"expected deployed bytecode on Ethereum to not match on Optimism, but it does. contract=%s bytecodeEth=%s bytecodeOp=%s",
			contractMetadataEth.Name,
			contractMetadataEth.DeployedBin,
			opContractData.DeployedBin,
		)
	}

	return nil
}

func (generator *BindGenGeneratorRemote) CompareDeployedBytecodeWithRpc(contractMetadata *RemoteContractMetadata, chain string) error {
	var client *ethclient.Client
	switch chain {
	case "eth":
		client = generator.RpcClients.Eth
	case "op":
		client = generator.RpcClients.Op
	default:
		return fmt.Errorf("unknown chain: %s, unable to retrieve a RPC client", chain)
	}

	var deployment common.Address
	switch chain {
	case "eth":
		deployment = contractMetadata.Deployments.Eth
	case "op":
		deployment = contractMetadata.Deployments.Op
	default:
		generator.Logger.Warn("Unable to compare bytecode from Etherscan against RPC client, no deployment address provided for chain", "chain", chain)
	}

	if deployment != (common.Address{}) {
		bytecode, err := client.CodeAt(context.Background(), common.HexToAddress(deployment.Hex()), nil)
		if err != nil {
			return fmt.Errorf("error getting deployed bytecode from RPC on chain: %s err: %w", chain, err)
		}
		bytecodeHex := common.Bytes2Hex(bytecode)
		if !strings.EqualFold(strings.TrimPrefix(contractMetadata.DeployedBin, "0x"), bytecodeHex) {
			return fmt.Errorf("%s deployment bytecode from RPC doesn't match bytecode from Etherscan. rpcBytecode: %s etherscanBytecode: %s", contractMetadata.Name, bytecodeHex, contractMetadata.DeployedBin)
		}
	}

	return nil
}

func (generator *BindGenGeneratorRemote) writeAllOutputs(contractMetadata *RemoteContractMetadata, fileTemplate string) error {
	abiFilePath, bytecodeFilePath, err := writeContractArtifacts(
		generator.Logger, generator.tempArtifactsDir, contractMetadata.Name,
		[]byte(contractMetadata.ABI), []byte(contractMetadata.InitBin),
	)
	if err != nil {
		return err
	}

	err = genContractBindings(generator.Logger, generator.MonorepoBasePath, abiFilePath, bytecodeFilePath, generator.BindingsPackageName, contractMetadata.Name)
	if err != nil {
		return err
	}

	return generator.writeContractMetadata(
		contractMetadata,
		template.Must(template.New("RemoteContractMetadata").Parse(fileTemplate)),
	)
}

func (generator *BindGenGeneratorRemote) writeContractMetadata(contractMetadata *RemoteContractMetadata, fileTemplate *template.Template) error {
	metadataFilePath := filepath.Join(generator.MetadataOut, strings.ToLower(contractMetadata.Name)+"_more.go")

	var existingOutput []byte
	if _, err := os.Stat(metadataFilePath); err == nil {
		existingOutput, err = os.ReadFile(metadataFilePath)
		if err != nil {
			return fmt.Errorf("error reading existing metadata output file, metadataFilePath: %s err: %w", metadataFilePath, err)
		}
	}

	metadataFile, err := os.OpenFile(
		metadataFilePath,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	if err != nil {
		return fmt.Errorf("error opening %s's metadata file at %s: %w", contractMetadata.Name, metadataFilePath, err)
	}
	defer metadataFile.Close()

	if err := fileTemplate.Execute(metadataFile, contractMetadata); err != nil {
		return fmt.Errorf("error writing %s's contract metadata at %s: %w", contractMetadata.Name, metadataFilePath, err)
	}

	if len(existingOutput) != 0 {
		var newOutput []byte
		newOutput, err = os.ReadFile(metadataFilePath)
		if err != nil {
			return fmt.Errorf("error reading new file: %w", err)
		}

		if bytes.Equal(existingOutput, newOutput) {
			generator.Logger.Debug("No changes detected in the contract metadata", "contract", contractMetadata.Name)
		} else {
			generator.Logger.Warn("Changes detected in the contract metadata, old metadata has been overwritten", "contract", contractMetadata.Name)
		}
	} else {
		generator.Logger.Debug("No existing contract metadata found, skipping comparison", "contract", contractMetadata.Name)
	}

	generator.Logger.Debug("Successfully wrote contract metadata", "contract", contractMetadata.Name, "path", metadataFilePath)
	return nil
}

// remoteContractMetadataTemplate is a Go text template for generating the metadata
// associated with a remotely sourced contracts.
//
// The template expects the following data to be provided:
// - .Package: the name of the Go package.
// - .Name: the name of the contract.
// - .DeployedBin: the binary (hex-encoded) of the deployed contract.
var remoteContractMetadataTemplate = `// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package {{.Package}}

var {{.Name}}DeployedBin = "{{.DeployedBin}}"
func init() {
	deployedBytecodes["{{.Name}}"] = {{.Name}}DeployedBin
}
`

// permit2MetadataTemplate is a Go text template used to generate metadata
// for remotely sourced Permit2 contract. Because Permit2 has an immutable
// Solidity variables that depends on block.chainid, we can't use the deployed
// bytecode, but instead need to generate it specifically for each chain.
// To help with this, the metadata contains the initialization bytecode, the
// deployer address, and the CREATE2 salt, so that deployment can be
// replicated as closely as possible.
//
// The template expects the following data to be provided:
// - .Package: the name of the Go package.
// - .Name: the name of the contract.
// - .InitBin: the binary (hex-encoded) of the contract's initialization code.
// - .DeploymentSalt: the salt used during the contract's deployment.
// - .Deployer: the Ethereum address of the contract's deployer.
var permit2MetadataTemplate = `// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package {{.Package}}

var {{.Name}}InitBin = "{{.InitBin}}"
var {{.Name}}DeploymentSalt = "{{.DeploymentSalt}}"
var {{.Name}}Deployer = "{{.Deployer}}"

func init() {
	initBytecodes["{{.Name}}"] = {{.Name}}InitBin
	deploymentSalts["{{.Name}}"] = {{.Name}}DeploymentSalt
	deployers["{{.Name}}"] = {{.Name}}Deployer
}
`
