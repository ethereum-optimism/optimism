package bindgen

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindgen"
	"github.com/ethereum-optimism/optimism/op-bindings/etherscan"
	"github.com/ethereum/go-ethereum/ethclient"
)

var generator bindgen.BindGenGeneratorRemote = bindgen.BindGenGeneratorRemote{}

func configureGenerator(t *testing.T) error {
	generator.ContractDataClients.Eth = etherscan.NewEthereumClient(os.Getenv("ETHERSCAN_APIKEY_ETH"))
	generator.ContractDataClients.Op = etherscan.NewOptimismClient(os.Getenv("ETHERSCAN_APIKEY_OP"))

	var err error
	if generator.RpcClients.Eth, err = ethclient.Dial(os.Getenv("RPC_URL_ETH")); err != nil {
		return fmt.Errorf("error initializing Ethereum client: %w", err)
	}
	if generator.RpcClients.Op, err = ethclient.Dial(os.Getenv("RPC_URL_OP")); err != nil {
		return fmt.Errorf("error initializing Optimism client: %w", err)
	}

	return nil
}

// configureGenerator initializes the generator with Ethereum and Optimism clients
// and RPC clients for both Ethereum and Optimism. It uses environment variables
// for configuration.
func configureGenerator(t *testing.T) error {
    // Initialize Ethereum client with Etherscan API key.
    generator.ContractDataClients.Eth = etherscan.NewEthereumClient(os.Getenv("ETHERSCAN_APIKEY_ETH"))
    // Initialize Optimism client with Etherscan API key.
    generator.ContractDataClients.Op = etherscan.NewOptimismClient(os.Getenv("ETHERSCAN_APIKEY_OP"))

    var err error
    // Initialize Ethereum RPC client.
    if generator.RpcClients.Eth, err = ethclient.Dial(os.Getenv("RPC_URL_ETH")); err != nil {
        return fmt.Errorf("error initializing Ethereum client: %w", err)
    }
    // Initialize Optimism RPC client.
    if generator.RpcClients.Op, err = ethclient.Dial(os.Getenv("RPC_URL_OP")); err != nil {
        return fmt.Errorf("error initializing Optimism client: %w", err)
    }

    return nil
}

// TestFetchContractData tests the FetchContractData function with various test cases.
func TestFetchContractData(t *testing.T) {
    // Configure the generator for each test case.
    if err := configureGenerator(t); err != nil {
        t.Error(err)
    }

    // Iterate over each test case.
    for _, tt := range fetchContractDataTests {
        t.Run(tt.name, func(t *testing.T) {
            // Fetch contract data and check for errors.
            contractData, err := generator.FetchContractData(tt.contractVerified, tt.chain, tt.deploymentAddress)
            if err != nil {
                t.Error(err)
            }
            // Compare the retrieved contract data with the expected data.
            if !reflect.DeepEqual(contractData, tt.expectedContractData) {
                t.Errorf("Retrieved contract data doesn't match expected. Expected: %s Retrieved: %s", tt.expectedContractData, contractData)
            }
        })
    }
}

// TestFetchContractDataFailures tests the FetchContractData function for failure scenarios.
func TestFetchContractDataFailures(t *testing.T) {
    // Configure the generator for each test case.
    if err := configureGenerator(t); err != nil {
        t.Error(err)
    }

    // Iterate over each test case.
    for _, tt := range fetchContractDataTestsFailures {
        t.Run(tt.name, func(t *testing.T) {
            // Attempt to fetch contract data expecting an error.
            _, err := generator.FetchContractData(tt.contractVerified, tt.chain, tt.deploymentAddress)
            if err == nil {
                t.Errorf("Expected error: %s but didn't receive it", tt.expectedError)
                return
            }

            // Check if the error message contains the expected error.
            if !strings.Contains(err.Error(), tt.expectedError) {
                t.Errorf("Expected error: %s Received: %s", tt.expectedError, err)
                return
            }
        })
    }
}

// TestCompareInitBytecodeWithOp tests the CompareInitBytecodeWithOp function with various test cases.
func TestCompareInitBytecodeWithOp(t *testing.T) {
    // Configure the generator for each test case.
    if err := configureGenerator(t); err != nil {
        t.Error(err)
    }

    // Iterate over each test case.
    for _, tt := range compareInitBytecodeWithOpTests {
        t.Run(tt.name, func(t *testing.T) {
            // Compare the initialization bytecode with the expected operation.
            err := generator.CompareInitBytecodeWithOp(&tt.contractMetadataEth, tt.initCodeShouldMatch)
            if err != nil {
                t.Error(err)
            }
        })
    }
}


func TestCompareInitBytecodeWithOpFailures(t *testing.T) {
	if err := configureGenerator(t); err != nil {
		t.Error(err)
	}

	for _, tt := range compareInitBytecodeWithOpTestsFailures {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.CompareInitBytecodeWithOp(&tt.contractMetadataEth, tt.initCodeShouldMatch)
			if err == nil {
				t.Errorf("Expected error: %s but didn't receive it", tt.expectedError)
				return
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error: %s Received: %s", tt.expectedError, err)
				return
			}
		})
	}
}

func TestCompareDeployedBytecodeWithOp(t *testing.T) {
	if err := configureGenerator(t); err != nil {
		t.Error(err)
	}

	for _, tt := range compareDeployedBytecodeWithOpTests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.CompareDeployedBytecodeWithOp(&tt.contractMetadataEth, tt.deployedCodeShouldMatch)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestCompareDeployedBytecodeWithOpFailures(t *testing.T) {
	if err := configureGenerator(t); err != nil {
		t.Error(err)
	}

	for _, tt := range compareDeployedBytecodeWithOpTestsFailures {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.CompareDeployedBytecodeWithOp(&tt.contractMetadataEth, tt.deployedCodeShouldMatch)
			if err == nil {
				t.Errorf("Expected error: %s but didn't receive it", tt.expectedError)
				return
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error: %s Received: %s", tt.expectedError, err)
				return
			}
		})
	}
}

func TestCompareDeployedBytecodeWithRpc(t *testing.T) {
	if err := configureGenerator(t); err != nil {
		t.Error(err)
	}

	for _, tt := range compareDeployedBytecodeWithRpcTests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.CompareDeployedBytecodeWithRpc(&tt.contractMetadataEth, tt.chain)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestCompareDeployedBytecodeWithRpcFailures(t *testing.T) {
	if err := configureGenerator(t); err != nil {
		t.Error(err)
	}

	for _, tt := range compareDeployedBytecodeWithRpcTestsFailures {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.CompareDeployedBytecodeWithRpc(&tt.contractMetadataEth, tt.chain)
			if err == nil {
				t.Errorf("Expected error: %s but didn't receive it", tt.expectedError)
				return
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error: %s Received: %s", tt.expectedError, err)
				return
			}
		})
	}
}
