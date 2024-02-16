package etherscan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/retry"
)

type client struct {
	baseUrl    string
	httpClient *http.Client
}

type apiResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

type rpcResponse struct {
	JsonRpc string          `json:"jsonrpc"`
	Id      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
}

type Transaction struct {
	Hash  string `json:"hash"`
	Input string `json:"input"`
	To    string `json:"to"`
}

const apiMaxRetries = 3
const apiRetryDelay = time.Duration(2) * time.Second
const errRateLimited = "Max rate limit reached"

func NewClient(baseUrl, apiKey string) *client {
	return &client{
		baseUrl: baseUrl + "/api?apikey=" + apiKey + "&",
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func NewEthereumClient(apiKey string) *client {
	return NewClient("https://api.etherscan.io", apiKey)
}

func NewOptimismClient(apiKey string) *client {
	return NewClient("https://api-optimistic.etherscan.io", apiKey)
}

func (c *client) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
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

func (c *client) fetchEtherscanApi(ctx context.Context, url string) (apiResponse, error) {
	return retry.Do[apiResponse](ctx, apiMaxRetries, retry.Fixed(apiRetryDelay), func() (apiResponse, error) {
		body, err := c.fetch(ctx, url)
		if err != nil {
			return apiResponse{}, err
		}

		var response apiResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return apiResponse{}, fmt.Errorf("failed to unmarshal as apiResponse: %w", err)
		}

		if response.Message != "OK" {
			var resultString string
			err = json.Unmarshal(response.Result, &resultString)
			if err != nil {
				return apiResponse{}, fmt.Errorf("response for %s not OK, returned message: %s", url, response.Message)
			}

			if resultString == errRateLimited {
				return apiResponse{}, errors.New(errRateLimited)
			}

			return apiResponse{}, fmt.Errorf("there was an issue with the Etherscan request to %s, received response: %v", url, response)
		}

		return response, nil
	})
}

func (c *client) fetchEtherscanRpc(ctx context.Context, url string) (rpcResponse, error) {
	return retry.Do[rpcResponse](ctx, apiMaxRetries, retry.Fixed(apiRetryDelay), func() (rpcResponse, error) {
		body, err := c.fetch(ctx, url)
		if err != nil {
			return rpcResponse{}, err
		}

		var response rpcResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return rpcResponse{}, fmt.Errorf("failed to unmarshal as rpcResponse: %w", err)
		}

		var resultString string
		_ = json.Unmarshal(response.Result, &resultString)
		if resultString == errRateLimited {
			return rpcResponse{}, errors.New(errRateLimited)
		}

		return response, nil
	})
}

func (c *client) FetchAbi(ctx context.Context, address string) (string, error) {
	params := url.Values{}
	params.Set("address", address)
	url := constructUrl(c.baseUrl, "getabi", "contract", params)
	response, err := c.fetchEtherscanApi(ctx, url)
	if err != nil {
		return "", err
	}

	var abi string
	err = json.Unmarshal(response.Result, &abi)
	if err != nil {
		return "", fmt.Errorf("API response result is not expected ABI string: %w", err)
	}

	return abi, nil
}

func (c *client) FetchDeployedBytecode(ctx context.Context, address string) (string, error) {
	params := url.Values{}
	params.Set("address", address)
	url := constructUrl(c.baseUrl, "eth_getCode", "proxy", params)
	response, err := c.fetchEtherscanRpc(ctx, url)
	if err != nil {
		return "", fmt.Errorf("error fetching deployed bytecode: %w", err)
	}

	var bytecode string
	err = json.Unmarshal(response.Result, &bytecode)
	if err != nil {
		return "", errors.New("API response result is not expected bytecode string")
	}

	return bytecode, nil
}

func (c *client) FetchDeploymentTxHash(ctx context.Context, address string) (string, error) {
	params := url.Values{}
	params.Set("contractaddresses", address)
	url := constructUrl(c.baseUrl, "getcontractcreation", "contract", params)
	response, err := c.fetchEtherscanApi(ctx, url)
	if err != nil {
		return "", err
	}

	var results []struct {
		Hash string `json:"txHash"`
	}
	err = json.Unmarshal(response.Result, &results)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal API response as []txInfo: %w", err)
	}

	if len(results) == 0 {
		return "", fmt.Errorf("API response result is an empty array")
	}

	return results[0].Hash, nil
}

func (c *client) FetchDeploymentTx(ctx context.Context, txHash string) (Transaction, error) {
	params := url.Values{}
	params.Set("txHash", txHash)
	params.Set("tag", "latest")
	url := constructUrl(c.baseUrl, "eth_getTransactionByHash", "proxy", params)
	response, err := c.fetchEtherscanRpc(ctx, url)
	if err != nil {
		return Transaction{}, err
	}

	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to marshal Result into JSON: %w", err)
	}

	var tx Transaction
	err = json.Unmarshal(resultBytes, &tx)
	if err != nil {
		return Transaction{}, fmt.Errorf("API response result is not expected txInfo struct: %w", err)
	}

	return tx, nil
}

func constructUrl(baseUrl, action, module string, params url.Values) string {
	params.Set("action", action)
	params.Set("module", module)
	queryFragment := params.Encode()
	return baseUrl + queryFragment
}
