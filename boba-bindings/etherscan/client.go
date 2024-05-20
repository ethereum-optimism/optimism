package etherscan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"time"
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
	return Do[apiResponse](ctx, apiMaxRetries, Fixed(apiRetryDelay), func() (apiResponse, error) {
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
	return Do[rpcResponse](ctx, apiMaxRetries, Fixed(apiRetryDelay), func() (rpcResponse, error) {
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

// ErrFailedPermanently is an error raised by Do when the
// underlying Operation has been retried maxAttempts times.
type ErrFailedPermanently struct {
	attempts int
	LastErr  error
}

func (e *ErrFailedPermanently) Error() string {
	return fmt.Sprintf("operation failed permanently after %d attempts: %v", e.attempts, e.LastErr)
}

func (e *ErrFailedPermanently) Unwrap() error {
	return e.LastErr
}

type pair[T, U any] struct {
	a T
	b U
}

func Do2[T, U any](ctx context.Context, maxAttempts int, strategy Strategy, op func() (T, U, error)) (T, U, error) {
	f := func() (pair[T, U], error) {
		a, b, err := op()
		return pair[T, U]{a, b}, err
	}
	res, err := Do(ctx, maxAttempts, strategy, f)
	return res.a, res.b, err
}

// Do performs the provided Operation up to maxAttempts times
// with delays in between each retry according to the provided
// Strategy.
func Do[T any](ctx context.Context, maxAttempts int, strategy Strategy, op func() (T, error)) (T, error) {
	var empty, ret T
	var err error
	if maxAttempts < 1 {
		return empty, fmt.Errorf("need at least 1 attempt to run op, but have %d max attempts", maxAttempts)
	}

	for i := 0; i < maxAttempts; i++ {
		if ctx.Err() != nil {
			return empty, ctx.Err()
		}
		ret, err = op()
		if err == nil {
			return ret, nil
		}
		// Don't sleep when we are about to exit the loop & return ErrFailedPermanently
		if i != maxAttempts-1 {
			time.Sleep(strategy.Duration(i))
		}
	}
	return empty, &ErrFailedPermanently{
		attempts: maxAttempts,
		LastErr:  err,
	}
}

// Strategy is used to calculate how long a particular Operation
// should wait between attempts.
type Strategy interface {
	// Duration returns how long to wait for a given retry attempt.
	Duration(attempt int) time.Duration
}

// ExponentialStrategy performs exponential backoff. The exponential backoff
// function is min(e.Min + (2^attempt * second), e.Max) + randBetween(0, e.MaxJitter)
type ExponentialStrategy struct {
	// Min is the minimum amount of time to wait between attempts.
	Min time.Duration

	// Max is the maximum amount of time to wait between attempts.
	Max time.Duration

	// MaxJitter is the maximum amount of random jitter to insert between attempts.
	// Jitter is added on top of the maximum, if the maximum is reached.
	MaxJitter time.Duration
}

func (e *ExponentialStrategy) Duration(attempt int) time.Duration {
	var jitter time.Duration // non-negative jitter
	if e.MaxJitter > 0 {
		jitter = time.Duration(rand.Int63n(e.MaxJitter.Nanoseconds()))
	}
	if attempt < 0 {
		return e.Min + jitter
	}
	durFloat := float64(e.Min)
	durFloat += math.Pow(2, float64(attempt)) * float64(time.Second)
	dur := time.Duration(durFloat)
	if durFloat > float64(e.Max) {
		dur = e.Max
	}
	dur += jitter

	return dur
}

func Exponential() Strategy {
	return &ExponentialStrategy{
		Min:       0,
		Max:       10 * time.Second,
		MaxJitter: 250 * time.Millisecond,
	}
}

type FixedStrategy struct {
	Dur time.Duration
}

func (f *FixedStrategy) Duration(attempt int) time.Duration {
	return f.Dur
}

func Fixed(dur time.Duration) Strategy {
	return &FixedStrategy{
		Dur: dur,
	}
}
