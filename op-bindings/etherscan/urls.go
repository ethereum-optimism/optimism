package etherscan

import "fmt"

// constructUrl builds a complete Etherscan API URL based on the provided parameters.
// It is a utility function used internally to generate specific API endpoint URLs.
//
// Parameters:
// - chainId: An integer representing the networks's chain ID.
// - action: A string specifying the Etherscan API action to be performed.
// - address: A string representing the Ethereum address related to the action.
// - module: A string denoting the Etherscan API module being accessed.
// - params: Additional string parameters to include in the URL.
//
// Returns:
// - string: The complete URL for the Etherscan API request.
// - error: An error object, returned if the chain ID is not supported or if other issues occur in URL construction.
//
// The function first retrieves the base URL and API key for the specified chain ID. It then constructs the URL
// by formatting these values along with the module, action, and additional parameters.
func (client *Client) constructUrl(chainId int, action, address, module, params string) (string, error) {
	baseUrl, ok := client.baseUrls[chainId]
	if !ok {
		return "", fmt.Errorf("unsupported chain ID provided: %d", chainId)
	}

	apiKey, ok := client.apiKeys[chainId]
	if !ok {
		return "", fmt.Errorf("unsupported chain ID provided: %d", chainId)
	}

	return fmt.Sprintf("%s?module=%s&action=%s&%s&apikey=%s", baseUrl, module, action, params, apiKey), nil
}

// getAbiUrl generates the Etherscan API URL to fetch the ABI of a smart contract.
//
// Parameters:
// - chainId: An integer representing the networks's chain ID.
// - contractAddress: A string representing the Ethereum address of the contract.
//
// Returns:
// - string: The URL to fetch the contract ABI.
// - error: An error object, returned if the URL construction fails.
func (client *Client) getAbiUrl(chainId int, contractAddress string) (string, error) {
	return client.constructUrl(chainId, "getabi", contractAddress, "contract", fmt.Sprintf("address=%s", contractAddress))
}

// getDeploymentTxHashUrl creates the Etherscan API URL to fetch the transaction hash of a contract's deployment.
//
// Parameters:
// - chainId: An integer representing the network's chain ID.
// - contractAddress: A string representing the Ethereum address of the contract.
//
// Returns:
// - string: The URL to fetch the deployment transaction hash.
// - error: An error object, returned if the URL construction fails.
func (client *Client) getDeploymentTxHashUrl(chainId int, contractAddress string) (string, error) {
	return client.constructUrl(chainId, "getcontractcreation", contractAddress, "contract", fmt.Sprintf("contractaddresses=%s", contractAddress))
}

// getDeployedBytecodeUrl forms the Etherscan API URL to fetch the deployed bytecode of a smart contract.
//
// Parameters:
// - chainId: An integer representing the network's chain ID.
// - contractAddress: A string representing the Ethereum address of the contract.
//
// Returns:
// - string: The URL to fetch the deployed bytecode.
// - error: An error object, returned if the URL construction fails.
func (client *Client) getDeployedBytecodeUrl(chainId int, contractAddress string) (string, error) {
	return client.constructUrl(chainId, "eth_getCode", contractAddress, "proxy", fmt.Sprintf("address=%s", contractAddress))
}

// getTxByHashUrl generates the Etherscan API URL to fetch transaction data based on its hash.
//
// Parameters:
// - chainId: An integer representing the network's chain ID.
// - txHash: A string representing the hash of the transaction.
//
// Returns:
// - string: The URL to fetch the transaction data.
// - error: An error object, returned if the URL construction fails.
func (client *Client) getTxByHashUrl(chainId int, txHash string) (string, error) {
	return client.constructUrl(chainId, "eth_getTransactionByHash", txHash, "proxy", fmt.Sprintf("txHash=%s&tag=latest", txHash))
}
