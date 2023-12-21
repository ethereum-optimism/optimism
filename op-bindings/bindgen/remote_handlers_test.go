package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/etherscan"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

var generator bindGenGeneratorRemote = bindGenGeneratorRemote{}

func configureGenerator(t *testing.T) error {
	if os.Getenv("RUN_E2E") == "" {
		t.Log("Not running test, RUN_E2E env not set")
		t.Skip()
	}

	generator.contractDataClients.eth = etherscan.NewEthereumClient(os.Getenv("ETHERSCAN_APIKEY_ETH"))
	generator.contractDataClients.op = etherscan.NewOptimismClient(os.Getenv("ETHERSCAN_APIKEY_OP"))

	var err error
	if generator.rpcClients.eth, err = ethclient.Dial(os.Getenv("RPC_URL_ETH")); err != nil {
		return fmt.Errorf("error initializing Ethereum client: %w", err)
	}
	if generator.rpcClients.op, err = ethclient.Dial(os.Getenv("RPC_URL_OP")); err != nil {
		return fmt.Errorf("error initializing Optimism client: %w", err)
	}

	return nil
}

func TestFetchContractData(t *testing.T) {
	if err := configureGenerator(t); err != nil {
		t.Error(err)
	}

	for _, tt := range fetchContractDataTests {
		t.Run(tt.name, func(t *testing.T) {
			contractData, err := generator.fetchContractData(tt.contractVerified, tt.chain, tt.deploymentAddress)
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(contractData, tt.expectedContractData) {
				t.Errorf("Retrieved contract data doesn't match expected. Expected: %s Retrieved: %s", tt.expectedContractData, contractData)
			}
		})
	}
}

func TestFetchContractDataFailures(t *testing.T) {
	if err := configureGenerator(t); err != nil {
		t.Error(err)
	}

	for _, tt := range fetchContractDataTestsFailures {
		t.Run(tt.name, func(t *testing.T) {
			_, err := generator.fetchContractData(tt.contractVerified, tt.chain, tt.deploymentAddress)
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

func TestRemoveDeploymentSalt(t *testing.T) {
	for _, tt := range removeDeploymentSaltTests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := generator.removeDeploymentSalt(tt.deploymentData, tt.deploymentSalt)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestRemoveDeploymentSaltFailures(t *testing.T) {
	for _, tt := range removeDeploymentSaltTestsFailures {
		t.Run(tt.name, func(t *testing.T) {
			_, err := generator.removeDeploymentSalt(tt.deploymentData, tt.deploymentSalt)
			require.Equal(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCompareInitBytecodeWithOp(t *testing.T) {
	if err := configureGenerator(t); err != nil {
		t.Error(err)
	}

	for _, tt := range compareInitBytecodeWithOpTests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.compareInitBytecodeWithOp(&tt.contractMetadataEth, tt.initCodeShouldMatch)
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
			err := generator.compareInitBytecodeWithOp(&tt.contractMetadataEth, tt.initCodeShouldMatch)
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
			err := generator.compareDeployedBytecodeWithOp(&tt.contractMetadataEth, tt.deployedCodeShouldMatch)
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
			err := generator.compareDeployedBytecodeWithOp(&tt.contractMetadataEth, tt.deployedCodeShouldMatch)
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
			err := generator.compareDeployedBytecodeWithRpc(&tt.contractMetadataEth, tt.chain)
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
			err := generator.compareDeployedBytecodeWithRpc(&tt.contractMetadataEth, tt.chain)
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
