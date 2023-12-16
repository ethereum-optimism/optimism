package main

import (
	"github.com/ethereum-optimism/optimism/op-bindings/etherscan"
	"github.com/ethereum/go-ethereum/common"
)

var fetchContractDataTests = []struct {
	name                 string
	contractVerified     bool
	chain                string
	deploymentAddress    string
	expectedContractData contractData
}{
	{
		"MultiCall3 on ETH",
		true,
		"eth",
		"0xcA11bde05977b3631167028862bE2a173976CA11",
		contractData{
			MultiCall3Abi,
			MultiCall3DeployedBytecode,
			etherscan.Transaction{
				Input: MultiCall3InitBytecode,
				Hash:  "0x00d9fcb7848f6f6b0aae4fb709c133d69262b902156c85a473ef23faa60760bd",
				To:    "",
			},
		},
	},
	{
		"MultiCall3 on OP",
		true,
		"op",
		"0xcA11bde05977b3631167028862bE2a173976CA11",
		contractData{
			MultiCall3Abi,
			MultiCall3DeployedBytecode,
			etherscan.Transaction{
				Input: MultiCall3InitBytecode,
				Hash:  "0xb62f9191a2cf399c0d2afd33f5b8baf7c6b52af6dd2386e44121b1bab91b80e5",
				To:    "",
			},
		},
	},
	{
		"SafeSingletonFactory on ETH",
		false,
		"eth",
		"0x914d7Fec6aaC8cd542e72Bca78B30650d45643d7",
		contractData{
			"",
			SafeSingletonFactoryDeployedBytecode,
			etherscan.Transaction{
				Input: SafeSingletonFactoryInitBytecode,
				Hash:  "0x69c275b5304db980105b7a6d731f9e1157a3fe29e7ff6ff95235297df53e9928",
				To:    "",
			},
		},
	},
	{
		"Permit2 on ETH",
		true,
		"eth",
		"0x000000000022D473030F116dDEE9F6B43aC78BA3",
		contractData{
			Permit2Abi,
			Permit2DeployedBytecode,
			etherscan.Transaction{
				Input: Permit2InitBytecode,
				Hash:  "0xf2f1fe96c16ee674bb7fcee166be52465a418927d124f5f1d231b36eae65d377",
				To:    "0x4e59b44847b379578588920ca78fbf26c0b4956c",
			},
		},
	},
}

// Not currently being tested due to complexity of test setup:
//   - FetchDeploymentTxHash failure
//     Not being tested because the contract would need to have deployed bytecode to
//     pass FetchDeployedBytecode, which means Etherscan should have indexed the deployment tx
//   - FetchDeploymentTx failure
//     Not being tested for the same reason and there would be no way to pass FetchDeploymentTxHash,
//     but not be able to retrieve tx details
var fetchContractDataTestsFailures = []struct {
	name              string
	contractVerified  bool
	chain             string
	deploymentAddress string
	expectedError     string
}{
	{
		"MultiCall3 on Foo",
		true,
		"foo",
		"0xcA11bde05977b3631167028862bE2a173976CA11",
		"unknown chain, unable to retrieve a contract data client for chain: foo",
	},
	{
		// This test case is covering fetching an ABI for a non-verified contract that's we're saying is verified
		"SafeSingletonFactory on ETH",
		true,
		"eth",
		"0x914d7Fec6aaC8cd542e72Bca78B30650d45643d7",
		"error fetching ABI: operation failed permanently after 3 attempts: there was an issue with the Etherscan request",
	},
	{
		// This test case is covering fetching the deployed bytecode for a non-existent contract
		"Nonexistent on ETH",
		false,
		"eth",
		"0x914d7Fec6aaC8cd542e72Bca78B30650d455555",
		"error fetching deployed bytecode: API response result is not expected bytecode string",
	},
}

// The Init bytecode used for these tests can either be sourced
// on-chain using the deployment tx of these contracts, or can be
// found in the bindings output from BindGen (../bindings/)
var removeDeploymentSaltTests = []struct {
	name           string
	deploymentData string
	deploymentSalt string
	expected       string
}{
	{
		"Case #1",
		Safe_v130InitBytecode,
		"0000000000000000000000000000000000000000000000000000000000000000",
		Safe_v130InitBytecodeNoSalt,
	},
	{
		"Case #2",
		Permit2InitBytecode,
		"0000000000000000000000000000000000000000d3af2663da51c10215000000",
		Permit2InitBytecodeNoSalt,
	},
	{
		"Case #3",
		EntryPointInitBytecode,
		"0000000000000000000000000000000000000000000000000000000000000000",
		EntryPointInitBytecodeNoSalt,
	},
}

var removeDeploymentSaltTestsFailures = []struct {
	name           string
	deploymentData string
	deploymentSalt string
	expectedError  string
}{
	{
		"Failure Case #1 Invalid Regex",
		"0x1234abc",
		"[invalid-regex",
		"failed to compile regular expression: error parsing regexp: missing closing ]: `[invalid-regex)`",
	},
	{
		"Failure Case #2 Salt Not Found",
		"0x1234abc",
		"4567",
		"expected salt: 4567 to be at the beginning of the contract initialization code: 0x1234abc, but it wasn't",
	},
}

var compareInitBytecodeWithOpTests = []struct {
	name                string
	contractMetadataEth remoteContractMetadata
	initCodeShouldMatch bool
}{
	{
		name: "Safe_v130 Init Bytecode Should Match",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op:  common.HexToAddress("0xd9Db270c1B5E3Bd161E8c8503c55cEABeE709552"),
					Eth: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     Safe_v130InitBytecodeNoSalt,
			DeployedBin: "",
		},
		initCodeShouldMatch: true,
	},
	{
		name: "Safe_v130 Compare Init Bytecode Only On OP",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     Safe_v130InitBytecodeNoSalt,
			DeployedBin: "",
		},
		initCodeShouldMatch: true,
	},
	{
		name: "Create2Deployer's Init Bytecode Should Not Match",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Create2Deployer",
				Verified: true,
				Deployments: deployments{
					Op:  common.HexToAddress("0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2"),
					Eth: common.HexToAddress("0xF49600926c7109BD66Ab97a2c036bf696e58Dbc2"),
				},
				Deployer:     common.Address{},
				ABI:          "",
				InitBytecode: "",
			},
			Package:     "bindings",
			InitBin:     Create2DeployerInitBytecode,
			DeployedBin: Create2DeployerDeployedBytecode,
		},
		initCodeShouldMatch: false,
	},
}

var compareInitBytecodeWithOpTestsFailures = []struct {
	name                string
	contractMetadataEth remoteContractMetadata
	initCodeShouldMatch bool
	expectedError       string
}{
	{
		name: "Safe_v130 Mismatch Init Bytecode",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op:  common.HexToAddress("0xd9Db270c1B5E3Bd161E8c8503c55cEABeE709552"),
					Eth: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     Permit2InitBytecodeNoSalt,
			DeployedBin: "",
		},
		initCodeShouldMatch: true,
		expectedError:       "expected initialization bytecode to match on Ethereum and Optimism, but it doesn't.",
	},
	{
		name: "Safe_v130 No Deployment on Optimism",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Eth: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     Safe_v130InitBytecode,
			DeployedBin: Safe_v130DeployedBytecode,
		},
		initCodeShouldMatch: true,
		expectedError:       "no deployment address on Optimism provided for Safe_v130",
	},
	{
		name: "MultiCall3 Expected Init Code Not to Match, but it Does",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "MultiCall3",
				Verified: true,
				Deployments: deployments{
					Op:  common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"),
					Eth: common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"),
				},
				Deployer:     common.Address{},
				ABI:          "",
				InitBytecode: "",
			},
			Package:     "bindings",
			InitBin:     MultiCall3InitBytecode,
			DeployedBin: MultiCall3DeployedBytecode,
		},
		initCodeShouldMatch: false,
		expectedError:       "expected initialization bytecode on Ethereum to not match on Optimism, but it did.",
	},
	{
		name: "Safe_v130 No Init Bytecode Provided",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op:  common.HexToAddress("0xd9Db270c1B5E3Bd161E8c8503c55cEABeE709552"),
					Eth: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     "",
			DeployedBin: Safe_v130DeployedBytecode,
		},
		initCodeShouldMatch: false,
		expectedError:       "no initialization bytecode provided for ETH deployment for comparison",
	},
}

var compareDeployedBytecodeWithOpTests = []struct {
	name                    string
	contractMetadataEth     remoteContractMetadata
	deployedCodeShouldMatch bool
}{
	{
		name: "Safe_v130 Deployed Bytecode Should Match",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op:  common.HexToAddress("0xd9Db270c1B5E3Bd161E8c8503c55cEABeE709552"),
					Eth: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     "",
			DeployedBin: Safe_v130DeployedBytecode,
		},
		deployedCodeShouldMatch: true,
	},
	{
		name: "Safe_v130 Compare Deployed Bytecode Only On OP",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     Safe_v130InitBytecodeNoSalt,
			DeployedBin: Safe_v130DeployedBytecode,
		},
		deployedCodeShouldMatch: true,
	},
	{
		name: "Permit2's Deployed Bytecode Should Not Match",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Permit2",
				Verified: true,
				Deployments: deployments{
					Op:  common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3"),
					Eth: common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3"),
				},
				Deployer:     common.Address{},
				ABI:          "",
				InitBytecode: "",
			},
			Package:     "bindings",
			InitBin:     Permit2InitBytecode,
			DeployedBin: Permit2DeployedBytecode,
		},
		deployedCodeShouldMatch: false,
	},
}

var compareDeployedBytecodeWithOpTestsFailures = []struct {
	name                    string
	contractMetadataEth     remoteContractMetadata
	deployedCodeShouldMatch bool
	expectedError           string
}{
	{
		name: "Safe_v130 Mismatch Deplolyed Bytecode",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op:  common.HexToAddress("0xd9Db270c1B5E3Bd161E8c8503c55cEABeE709552"),
					Eth: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     "",
			DeployedBin: Permit2DeployedBytecode,
		},
		deployedCodeShouldMatch: true,
		expectedError:           "expected deployed bytecode to match on Ethereum and Optimism, but it doesn't.",
	},
	{
		name: "Safe_v130 No Deployment on Optimism",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Eth: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     "",
			DeployedBin: Permit2DeployedBytecode,
		},
		deployedCodeShouldMatch: true,
		expectedError:           "no deployment address on Optimism provided for Safe_v130",
	},
	{
		name: "Safe_v130 Expected Deployed Code Not to Match, but it Does",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op:  common.HexToAddress("0xd9Db270c1B5E3Bd161E8c8503c55cEABeE709552"),
					Eth: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     Safe_v130InitBytecode,
			DeployedBin: Safe_v130DeployedBytecode,
		},
		deployedCodeShouldMatch: false,
		expectedError:           "expected deployed bytecode on Ethereum to not match on Optimism, but it does.",
	},
	{
		name: "Safe_v130 No Deployed Bytecode Provided",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op:  common.HexToAddress("0xd9Db270c1B5E3Bd161E8c8503c55cEABeE709552"),
					Eth: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     Safe_v130InitBytecode,
			DeployedBin: "",
		},
		deployedCodeShouldMatch: false,
		expectedError:           "no deployed bytecode provided for ETH deployment for comparison",
	},
}

var compareDeployedBytecodeWithRpcTests = []struct {
	name                string
	contractMetadataEth remoteContractMetadata
	chain               string
}{
	{
		name: "Safe_v130 Compare Against ETH",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op:  common.Address{},
					Eth: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     "",
			DeployedBin: Safe_v130DeployedBytecode,
		},
		chain: "eth",
	},
	{
		name: "Safe_v130 Compare Against OP",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op:  common.HexToAddress("0xd9Db270c1B5E3Bd161E8c8503c55cEABeE709552"),
					Eth: common.Address{},
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     "",
			DeployedBin: Safe_v130DeployedBytecode,
		},
		chain: "op",
	},
}

var compareDeployedBytecodeWithRpcTestsFailures = []struct {
	name                string
	contractMetadataEth remoteContractMetadata
	chain               string
	expectedError       string
}{
	{
		name: "Safe_v130 Compare Against foo",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op:  common.Address{},
					Eth: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     "",
			DeployedBin: "",
		},
		chain:         "foo",
		expectedError: "unknown chain: foo, unable to retrieve a RPC client",
	},
	{
		name: "Safe_v130 Bytecode Mismatch",
		contractMetadataEth: remoteContractMetadata{
			remoteContract: remoteContract{
				Name:     "Safe_v130",
				Verified: true,
				Deployments: deployments{
					Op:  common.Address{},
					Eth: common.HexToAddress("0x69f4D1788e39c87893C980c06EdF4b7f686e2938"),
				},
				DeploymentSalt: "0000000000000000000000000000000000000000000000000000000000000000",
				Deployer:       common.Address{},
				ABI:            "",
				InitBytecode:   "",
			},
			Package:     "bindings",
			InitBin:     "",
			DeployedBin: Permit2DeployedBytecode,
		},
		chain:         "eth",
		expectedError: "Safe_v130 deployment bytecode from RPC doesn't match bytecode from Etherscan.",
	},
}
