package verify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

const (
	testAPIKey     = "test_api_key"
	testAddressHex = "0x1234567890123456789012345678901234567890"
	testTxHashHex  = "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testGUID       = "verification_guid_12345"
)

// createTestClient creates a new EtherscanClient with a mock server for testing
func createTestClient(t *testing.T, handler http.HandlerFunc) (*EtherscanClient, *httptest.Server) {
	server := httptest.NewServer(handler)
	t.Cleanup(func() { server.Close() })

	// Use a fast rate limiter for testing
	limiter := rate.NewLimiter(rate.Every(time.Millisecond), 10)
	return NewEtherscanClient(testAPIKey, server.URL, limiter), server
}

// createTestArtifact creates a contract artifact for testing
func createTestArtifact() *contractArtifact {
	return &contractArtifact{
		ContractName:    "TestContract",
		CompilerVersion: "0.8.10",
		EVMVersion:      "london",
		Optimizer: OptimizerSettings{
			Enabled: true,
			Runs:    200,
		},
		Sources: map[string]SourceContent{
			"TestContract.sol": {Content: "contract TestContract {}"},
		},
	}
}

func TestGetAPIEndpoint(t *testing.T) {
	tests := []struct {
		name          string
		chainID       uint64
		expected      string
		expectedError bool
	}{
		{"Mainnet", 1, "https://api.etherscan.io/api", false},
		{"Sepolia", 11155111, "https://api-sepolia.etherscan.io/api", false},
		{"Unknown", 999, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getAPIEndpoint(tt.chainID)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetContractCreation(t *testing.T) {
	client, _ := createTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)
		require.Contains(t, r.URL.String(), "module=contract")
		require.Contains(t, r.URL.String(), "action=getcontractcreation")

		testAddr := common.HexToAddress(testAddressHex)
		require.Contains(t, r.URL.String(), testAddr.Hex())

		resp := EtherscanContractCreationResp{
			Status:  "1",
			Message: "OK",
			Result: []struct {
				ContractCreator string `json:"contractCreator"`
				TxHash          string `json:"txHash"`
			}{
				{
					ContractCreator: "0xabcdef1234567890abcdef1234567890abcdef12",
					TxHash:          testTxHashHex,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	})

	testAddr := common.HexToAddress(testAddressHex)
	txHash, err := client.getContractCreation(testAddr)

	require.NoError(t, err)
	require.Equal(t, testTxHashHex, txHash.Hex())
}

func TestGetContractCreationError(t *testing.T) {
	client, _ := createTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := EtherscanContractCreationResp{
			Status:  "0",
			Message: "Error",
			Result:  nil,
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	})

	testAddr := common.HexToAddress(testAddressHex)
	_, err := client.getContractCreation(testAddr)

	require.Error(t, err)
	require.Contains(t, err.Error(), "contract creation query failed")
}

func TestVerifySourceCode(t *testing.T) {
	client, _ := createTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		err := r.ParseForm()
		require.NoError(t, err)

		require.Equal(t, testAPIKey, r.Form.Get("apikey"))
		require.Equal(t, "contract", r.Form.Get("module"))
		require.Equal(t, "verifysourcecode", r.Form.Get("action"))
		require.Equal(t, testAddressHex, r.Form.Get("contractaddress"))

		resp := EtherscanGenericResp{
			Status:  "1",
			Message: "OK",
			Result:  testGUID,
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	})

	testAddr := common.HexToAddress(testAddressHex)
	artifact := createTestArtifact()

	guid, err := client.verifySourceCode(testAddr, artifact, "0x1234")

	require.NoError(t, err)
	require.Equal(t, testGUID, guid)
}

func TestIsVerified(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus string
		expected       bool
	}{
		{"Verified", "1", true},
		{"Not Verified", "0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := createTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "GET", r.Method)
				require.Contains(t, r.URL.String(), "module=contract")
				require.Contains(t, r.URL.String(), "action=getabi")

				resp := EtherscanGenericResp{
					Status:  tt.responseStatus,
					Message: "OK",
					Result:  "test result",
				}

				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(resp)
				require.NoError(t, err)
			})

			testAddr := common.HexToAddress(testAddressHex)
			result, err := client.isVerified(testAddr)

			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestPollVerificationStatus(t *testing.T) {
	tests := []struct {
		name         string
		responses    []EtherscanGenericResp
		expectError  bool
		errorMessage string
	}{
		{
			name: "Success After Pending",
			responses: []EtherscanGenericResp{
				{Status: "0", Message: "OK", Result: "Pending in queue"},
				{Status: "1", Message: "OK", Result: "Pass - Verified"},
			},
			expectError: false,
		},
		{
			name: "Already Verified",
			responses: []EtherscanGenericResp{
				{Status: "0", Message: "OK", Result: "Already Verified"},
			},
			expectError: false,
		},
		{
			name: "Verification Failed",
			responses: []EtherscanGenericResp{
				{Status: "0", Message: "OK", Result: "Fail - Unable to verify"},
			},
			expectError:  true,
			errorMessage: "verification failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responseIndex := 0

			client, _ := createTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "GET", r.Method)
				require.Contains(t, r.URL.String(), "module=contract")
				require.Contains(t, r.URL.String(), "action=checkverifystatus")

				// Get the appropriate response based on the current index
				resp := tt.responses[responseIndex]
				if responseIndex < len(tt.responses)-1 {
					responseIndex++
				}

				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(resp)
				require.NoError(t, err)
			})

			err := client.pollVerificationStatus("test_guid")

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewStandardInput(t *testing.T) {
	artifact := createTestArtifact()

	result := newStandardInput(artifact)

	require.Equal(t, "Solidity", result.Language)
	require.Equal(t, artifact.Sources, result.Sources)
	require.Equal(t, artifact.Optimizer.Enabled, result.Settings.Optimizer.Enabled)
	require.Equal(t, artifact.Optimizer.Runs, result.Settings.Optimizer.Runs)
	require.Equal(t, artifact.EVMVersion, result.Settings.EVMVersion)
	require.True(t, result.Settings.Metadata.UseLiteralContent)
	require.Equal(t, "none", result.Settings.Metadata.BytecodeHash)

	require.Contains(t, result.Settings.OutputSelection.All["*"].All, "abi")
	require.Contains(t, result.Settings.OutputSelection.All["*"].All, "evm.bytecode.object")
	require.Contains(t, result.Settings.OutputSelection.All["*"].All, "metadata")
}
