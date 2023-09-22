package client

import (
	"fmt"
	"io"
	"net/http"

	"encoding/json"

	"github.com/ethereum-optimism/optimism/indexer/api"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum/go-ethereum/common"
)

const (
	urlParams = "?cursor=%s&limit=%d"

	defaultPagingLimit = 100
)

// Config ... Indexer client config struct
type Config struct {
	PaginationLimit int
	URL             string
}

// Client ... Indexer client struct
// TODO: Add metrics
// TODO: Add injectable context support
type Client struct {
	cfg *Config
	c   *http.Client
	m   node.Metricer
}

// NewClient ... Construct a new indexer client
func NewClient(cfg *Config, m node.Metricer) (*Client, error) {
	if cfg.PaginationLimit <= 0 {
		cfg.PaginationLimit = defaultPagingLimit
	}

	c := &http.Client{}
	return &Client{cfg: cfg, c: c, m: m}, nil
}

// HealthCheck ... Checks the health of the indexer
func (c *Client) HealthCheck() error {
	resp, err := c.c.Get(c.cfg.URL + api.HealthPath)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status code %d", resp.StatusCode)
	}

	return nil
}

// GetDepositsByAddress ... Gets a deposit response object provided an L1 address and cursor
func (c *Client) GetDepositsByAddress(l1Address common.Address, cursor string) (*database.L1BridgeDepositsResponse, error) {
	var dResponse *database.L1BridgeDepositsResponse
	url := c.cfg.URL + api.DepositsPath + l1Address.String() + urlParams

	endpoint := fmt.Sprintf(url, cursor, c.cfg.PaginationLimit)
	resp, err := c.c.Get(endpoint)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if err := json.Unmarshal(body, &dResponse); err != nil {
		return nil, err
	}

	return dResponse, nil
}

// GetAllDepositsByAddress ... Gets all deposits provided a L1 address
func (c *Client) GetAllDepositsByAddress(l1Address common.Address) ([]database.L1BridgeDepositWithTransactionHashes, error) {
	var deposits []database.L1BridgeDepositWithTransactionHashes

	cursor := ""
	for {
		dResponse, err := c.GetDepositsByAddress(l1Address, cursor)
		if err != nil {
			return nil, err
		}

		deposits = append(deposits, dResponse.Deposits...)

		if !dResponse.HasNextPage {
			break
		}

		cursor = dResponse.Cursor
	}

	return deposits, nil

}

// GetAllWithdrawalsByAddress ... Gets all withdrawals provided a L2 address
func (c *Client) GetAllWithdrawalsByAddress(l2Address common.Address) ([]database.L2BridgeWithdrawalWithTransactionHashes, error) {
	var withdrawals []database.L2BridgeWithdrawalWithTransactionHashes

	cursor := ""
	for {
		wResponse, err := c.GetWithdrawalsByAddress(l2Address, cursor)
		if err != nil {
			return nil, err
		}

		withdrawals = append(withdrawals, wResponse.Withdrawals...)

		if !wResponse.HasNextPage {
			break
		}

		cursor = wResponse.Cursor
	}

	return withdrawals, nil
}

// GetWithdrawalsByAddress ... Gets a withdrawal response object provided an L2 address and cursor
func (c *Client) GetWithdrawalsByAddress(l2Address common.Address, cursor string) (*database.L2BridgeWithdrawalsResponse, error) {
	var wResponse *database.L2BridgeWithdrawalsResponse
	url := c.cfg.URL + api.WithdrawalsPath + l2Address.String() + urlParams

	endpoint := fmt.Sprintf(url, cursor, c.cfg.PaginationLimit)
	resp, err := c.c.Get(endpoint)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if err := json.Unmarshal(body, &wResponse); err != nil {
		return nil, err
	}

	return wResponse, nil
}
