package opcm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

type ReadSuperchainDeploymentInput struct {
	OPCMAddress common.Address `abi:"opcmAddress"`
}

type ReadSuperchainDeploymentOutput struct {
	ProtocolVersionsImpl  common.Address
	ProtocolVersionsProxy common.Address
	SuperchainConfigImpl  common.Address
	SuperchainConfigProxy common.Address
	SuperchainProxyAdmin  common.Address

	Guardian                   common.Address
	ProtocolVersionsOwner      common.Address
	SuperchainProxyAdminOwner  common.Address
	RecommendedProtocolVersion [32]byte
	RequiredProtocolVersion    [32]byte
}

// Function signatures
var (
	// OPContractsManager functions
	funcProtocolVersions     = w3.MustNewFunc("protocolVersions()", "address")
	funcSuperchainConfig     = w3.MustNewFunc("superchainConfig()", "address")
	funcSuperchainProxyAdmin = w3.MustNewFunc("superchainProxyAdmin()", "address")

	// Proxy functions
	funcImplementation = w3.MustNewFunc("implementation()", "address")

	// Ownable functions
	funcOwner = w3.MustNewFunc("owner()", "address")

	// SuperchainConfig functions
	funcGuardian = w3.MustNewFunc("guardian()", "address")

	// ProtocolVersions functions
	funcRecommended = w3.MustNewFunc("recommended()", "uint256")
	funcRequired    = w3.MustNewFunc("required()", "uint256")
)

func ReadSuperchainDeployment(ctx context.Context, rpcClient *rpc.Client, input ReadSuperchainDeploymentInput) (*ReadSuperchainDeploymentOutput, error) {
	if input.OPCMAddress == (common.Address{}) {
		return nil, fmt.Errorf("ReadSuperchainDeployment: opcmAddress not set")
	}

	w3Client := w3.NewClient(rpcClient)
	defer w3Client.Close()

	output := &ReadSuperchainDeploymentOutput{}

	// Step 1: Get proxy addresses from OPCM
	if err := w3Client.Call(
		eth.CallFunc(input.OPCMAddress, funcProtocolVersions).Returns(&output.ProtocolVersionsProxy),
		eth.CallFunc(input.OPCMAddress, funcSuperchainConfig).Returns(&output.SuperchainConfigProxy),
		eth.CallFunc(input.OPCMAddress, funcSuperchainProxyAdmin).Returns(&output.SuperchainProxyAdmin),
	); err != nil {
		return nil, fmt.Errorf("failed to get proxy addresses: %w", err)
	}

	// Step 2: Get implementation addresses from proxies
	if err := w3Client.Call(
		eth.CallFunc(output.ProtocolVersionsProxy, funcImplementation).Returns(&output.ProtocolVersionsImpl),
		eth.CallFunc(output.SuperchainConfigProxy, funcImplementation).Returns(&output.SuperchainConfigImpl),
	); err != nil {
		return nil, fmt.Errorf("failed to get implementation addresses: %w", err)
	}

	// Step 3: Get owner/guardian information and protocol versions
	var recommendedBig, requiredBig *big.Int
	if err := w3Client.Call(
		eth.CallFunc(output.SuperchainConfigProxy, funcGuardian).Returns(&output.Guardian),
		eth.CallFunc(output.ProtocolVersionsProxy, funcOwner).Returns(&output.ProtocolVersionsOwner),
		eth.CallFunc(output.SuperchainProxyAdmin, funcOwner).Returns(&output.SuperchainProxyAdminOwner),
		eth.CallFunc(output.ProtocolVersionsProxy, funcRecommended).Returns(&recommendedBig),
		eth.CallFunc(output.ProtocolVersionsProxy, funcRequired).Returns(&requiredBig),
	); err != nil {
		return nil, fmt.Errorf("failed to get owner/guardian info: %w", err)
	}

	// Convert big.Int to [32]byte
	if recommendedBig != nil {
		recommendedBytes := recommendedBig.Bytes()
		copy(output.RecommendedProtocolVersion[32-len(recommendedBytes):], recommendedBytes)
	}
	if requiredBig != nil {
		requiredBytes := requiredBig.Bytes()
		copy(output.RequiredProtocolVersion[32-len(requiredBytes):], requiredBytes)
	}

	return output, nil
}
