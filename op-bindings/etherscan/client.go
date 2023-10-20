package etherscan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/retry"
)

type Client struct {
	apiMaxRetries int
	apiRetryDelay time.Duration
	baseUrls      map[int]string
	apiKeys       map[int]string
}

type apiResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

type rpcResponse struct {
	JsonRpc string      `json:"jsonrpc"`
	Id      int         `json:"id"`
	Result  interface{} `json:"result"`
}

type txInfo struct {
	Input string `json:"input"`
}

// validateChainId checks if the provided chain ID is supported by the application.
// Currently, this function only supports chain IDs 1 and 10.
//
// This validation is crucial to ensure that the application interacts only with
// recognized and supported blockchain networks, preventing errors or unexpected
// behavior when dealing with blockchain data from unsupported networks.
//
// Parameters:
// - chainId: An integer representing the chain ID to be validated.
//
// The function returns an error in the following scenario:
// - If the chain ID is not one of the supported IDs.
func validateChainId(chainId int) error {
	switch chainId {
	case 1, 10:
	default:
		return fmt.Errorf("unsupported source chain ID: %d", chainId)
	}

	return nil
}

// NewClient initializes a new Client struct with the provided parameters. This client is used to interact with the Etherscan API for both the source and compare chains.
//
// Parameters:
// - sourceChainId: An integer representing the chain ID of the source chain.
// - compareChainId: An integer representing the chain ID of the compare chain.
// - sourceApiKey: A string representing the Etherscan API key for the source chain.
// - compareApiKey: A string representing the Etherscan API key for the compare chain (if needed).
// - apiMaxRetries: An integer representing the maximum number of retries for an API call.
// - apiRetryDelay: An integer representing the delay (in seconds) between API call retries.
//
// The function returns a pointer to a Client struct and an error in the following scenarios:
// - If either the source or compare API key is missing when required.
// - If either the source or compare chain ID is not supported.
func NewClient(
	sourceChainId,
	compareChainId int,
	sourceApiKey,
	compareApiKey string,
	apiMaxRetries,
	apiRetryDelay int,
) (*Client, error) {
	if sourceApiKey == "" {
		return nil, errors.New("an Etherscan API key for the source chain is required, but none was provided")
	}

	if compareChainId > 0 && compareApiKey == "" {
		return nil, errors.New("an Etherscan API key for the compare chain is required, but none was provided")
	}

	if err := validateChainId(sourceChainId); err != nil {
		return nil, err
	}
	if err := validateChainId(compareChainId); err != nil {
		return nil, err
	}

	return &Client{
		apiMaxRetries: apiMaxRetries,
		apiRetryDelay: time.Duration(apiRetryDelay) * time.Second,
		baseUrls: map[int]string{
			1:  "https://api.etherscan.io/api",
			10: "https://api-optimistic.etherscan.io/api",
		},
		apiKeys: map[int]string{
			sourceChainId:  sourceApiKey,
			compareChainId: compareApiKey,
		},
	}, nil
}

// fetch performs an HTTP GET request to the specified URL and retrieves the response body.
//
// Parameters:
// - url: A string representing the URL to which the HTTP GET request is sent.
//
// The function returns a byte slice and an error in the following scenarios:
// - The byte slice contains the response body if the HTTP request is successful.
// - An error is returned if any issues occur during the HTTP request or while reading the response body.
func (client *Client) fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// fetchEtherscanApi attempts to fetch data from an Etherscan RPC endpoint using the specified URL.
// The method makes HTTP GET requests and handles retries if the rate limit is reached. It parses
// the response into an apiResponse struct. The method implements a retry mechanism with delays
// to manage API rate limits effectively.
//
// Parameters:
// - url: A string representing the Etherscan API endpoint to which the request is sent.
//
// Returns:
// - apiResponse: A structured representation of the Etherscan API response.
// - error: An error object that captures any issues encountered during the request and parsing process.
//
// The function executes up to client.apiMaxRetries times, with a delay of client.apiRetryDelay between retries
// if the API rate limit is reached. It unmarshals the response body into an apiResponse struct, checking for
// any issues indicated by the API (e.g., reaching the rate limit or other errors).
//
// The method returns an error if it fails to fetch a valid response after the maximum number of retries or
// if the response contains an error message different from "OK".
func (client *Client) fetchEtherscanApi(url string) (apiResponse, error) {
	return retry.Do[apiResponse](context.Background(), client.apiMaxRetries, retry.Fixed(client.apiRetryDelay), func() (apiResponse, error) {
		body, err := client.fetch(url)
		if err != nil {
			return apiResponse{}, err
		}

		var response apiResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return apiResponse{}, fmt.Errorf("failed to unmarshal as apiResponse: %w", err)
		}

		if response.Message != "OK" && response.Result != "Max rate limit reached" {
			return apiResponse{}, fmt.Errorf("there was an issue with the Etherscan request to %s, received response: %v", url, response)
		}

		return response, nil
	})
}

// fetchEtherscanRpc attempts to fetch data from an Etherscan RPC endpoint using the specified URL.
// The method makes HTTP GET requests and handles retries if the rate limit is reached. It parses
// the response into an rpcResponse struct. The method implements a retry mechanism with delays
// to manage API rate limits effectively.
//
// Parameters:
// - url: A string representing the Etherscan RPC endpoint URL.
//
// Returns:
// - rpcResponse: A struct that represents the unmarshaled JSON response from the Etherscan RPC endpoint.
// - error: An error indicating issues encountered during the fetch operation or unmarshaling process.
//
// The function executes up to client.apiMaxRetries times, waiting client.apiRetryDelay between retries
// if the rate limit is reached or if unmarshaling the response fails. The function unmarshals the response
// body into an rpcResponse struct and checks for rate limit errors.
//
// If the function fails to fetch a valid response after the maximum number of retries or encounters issues
// in parsing the response, it returns an error.
func (client *Client) fetchEtherscanRpc(url string) (rpcResponse, error) {
	return retry.Do[rpcResponse](context.Background(), client.apiMaxRetries, retry.Fixed(client.apiRetryDelay), func() (rpcResponse, error) {
		body, err := client.fetch(url)
		if err != nil {
			return rpcResponse{}, err
		}

		var response rpcResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return rpcResponse{}, fmt.Errorf("failed to unmarshal as rpcResponse: %w", err)
		}

		return response, nil
	})
}

// FetchAbi retrieves the ABI for a contract via the Etherscan API using the provided chain ID and address.
//
// Parameters:
// - chainId: An integer representing the blockchain's chain ID where the contract is deployed.
// - address: A string representing the Ethereum address of the contract.
//
// Returns:
// - string: The ABI string of the specified smart contract.
// - error: An error object that captures any issues encountered during the ABI retrieval process.
//
// If the response contains a valid ABI string, it is returned; otherwise, the function reports
// an error indicating either a problem in fetching data from Etherscan or an unexpected response format.
func (client *Client) FetchAbi(chainId int, address string) (string, error) {
	url, err := client.getAbiUrl(chainId, address)
	if err != nil {
		return "", err
	}

	response, err := client.fetchEtherscanApi(url)
	if err != nil {
		return "", err
	}

	abi, ok := response.Result.(string)
	if !ok {
		return "", fmt.Errorf("API response result is not expected ABI string")
	}

	return abi, nil
}

// FetchDeployedBytecode retrieves the deployed bytecode of a smart contract from the
// Etherscan API using the provided chain ID and address.
//
// Parameters:
// - chainId: An integer representing the blockchain's chain ID where the contract is deployed.
// - address: A string representing the Ethereum address of the contract.
//
// Returns:
// - string: The deployed bytecode of the specified smart contract.
// - error: An error object that captures any issues encountered during the bytecode retrieval process.
//
// If the response contains a valid bytecode string, it is returned; otherwise, the function reports
// an error indicating either a problem in fetching data from Etherscan or an unexpected response format.
func (client *Client) FetchDeployedBytecode(chainId int, address string) (string, error) {
	url, err := client.getDeployedBytecodeUrl(chainId, address)
	if err != nil {
		return "", err
	}

	response, err := client.fetchEtherscanRpc(url)
	if err != nil {
		return "", fmt.Errorf("error fetching deployed bytecode: %w", err)
	}

	bytecode, ok := response.Result.(string)
	if !ok {
		return "", errors.New("API response result is not expected bytecode string")
	}

	return bytecode, nil
}

// FetchDeploymentTxHash retrieves the transaction hash of a contract's deployment from the
// Etherscan API using the provided chain ID and address.
//
// Parameters:
// - chainId: An integer representing the blockchain's chain ID where the contract is deployed.
// - address: A string representing the Ethereum address of the contract.
//
// Returns:
// - string: The transaction hash of the contract's deployment.
// - error: An error object that captures issues encountered during the transaction hash retrieval process.
//
// The method constructs a URL to query the deployment transaction hash, then fetches data from Etherscan.
// It expects the response to be []txInfo, from which the transaction hash is extracted from txInfo[0].
// If the response does not match the expected format, an error is returned.
func (client *Client) FetchDeploymentTxHash(chainId int, address string) (string, error) {
	url, err := client.getDeploymentTxHashUrl(chainId, address)
	if err != nil {
		return "", err
	}

	response, err := client.fetchEtherscanApi(url)
	if err != nil {
		return "", err
	}

	results, ok := response.Result.([]interface{})
	if !ok {
		return "", fmt.Errorf("failed to assert API response result is type of []txInfo")
	}
	if len(results) == 0 {
		return "", fmt.Errorf("API response result is an empty array")
	}

	deploymentTxInfo, ok := results[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("failed to assert API response result[0] is type of txInfo")
	}

	txHash, ok := deploymentTxInfo["txHash"].(string)
	if !ok {
		return "", fmt.Errorf("failed to assert API response result[0][\"txHash\"] is type of string")
	}

	return txHash, nil
}

// FetchDeploymentData retrieves the input data of a transaction from the Etherscan RPC API
// using the provided chain ID and address.
//
// Parameters:
// - chainId: An integer representing the blockchain's chain ID where the contract is deployed.
// - txHash: A string representing the hash of the transaction.
//
// Returns:
// - string: The input data of the specified transaction.
// - error: An error object that captures any issues encountered during the data retrieval process.
//
// This function is used for fetching the initialization bytecode of a smart contract.
// The response is expected to be of the structure txInfo, from which the transaction input data is extracted.
func (client *Client) FetchDeploymentData(chainId int, txHash string) (string, error) {
	url, err := client.getTxByHashUrl(chainId, txHash)
	if err != nil {
		return "", err
	}

	response, err := client.fetchEtherscanRpc(url)
	if err != nil {
		return "", err
	}

	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Result into JSON: %w", err)
	}

	var tx txInfo
	err = json.Unmarshal(resultBytes, &tx)
	if err != nil {
		return "", fmt.Errorf("API response result is not expected txInfo struct: %w", err)
	}

	return tx.Input, nil
}
