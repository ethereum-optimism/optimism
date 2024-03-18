package client

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"encoding/json"

	"github.com/ethereum-optimism/optimism/indexer/api"
	"github.com/ethereum-optimism/optimism/indexer/api/models"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum/go-ethereum/common"
)

const (
	urlParams = "?cursor=%s&limit=%d"

	defaultPagingLimit = 100

	// method names
	healthz     = "get_health"
	deposits    = "get_deposits"
	withdrawals = "get_withdrawals"
	sum         = "get_sum"
)

// Option ... Provides configuration through callback injection
type Option func(*Client) error

// WithMetrics ... Triggers metric optionality
func WithMetrics(m node.Metricer) Option {
	return func(c *Client) error {
		c.metrics = m
		return nil
	}
}

// WithTimeout ... Embeds a timeout limit to request
func WithTimeout(t time.Duration) Option {
	return func(c *Client) error {
		c.c.Timeout = t
		return nil
	}
}

// Config ... Indexer client config struct
type Config struct {
	PaginationLimit int
	BaseURL         string
}

// Client ... Indexer client struct
type Client struct {
	cfg     *Config
	c       *http.Client
	metrics node.Metricer
}

// NewClient ... Construct a new indexer client
func NewClient(cfg *Config, opts ...Option) (*Client, error) {
	if cfg.PaginationLimit <= 0 {
		cfg.PaginationLimit = defaultPagingLimit
	}

	c := &http.Client{}
	client := &Client{cfg: cfg, c: c}

	for _, opt := range opts {
		err := opt(client)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

// doRecordRequest ... Performs a read request on a provided endpoint w/ telemetry
func (c *Client) doRecordRequest(method string, endpoint string) ([]byte, error) {
	var record func(error) = nil
	if c.metrics != nil {
		record = c.metrics.RecordRPCClientRequest(method)
	}

	resp, err := c.c.Get(endpoint)
	if record != nil {
		record(err)
	}

	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("endpoint failed with status code %d", resp.StatusCode)

	}

	return body, resp.Body.Close()
}

// HealthCheck ... Checks the health of the indexer API
func (c *Client) HealthCheck() error {

	_, err := c.doRecordRequest(healthz, c.cfg.BaseURL+api.HealthPath)

	if err != nil {
		return err
	}

	return nil
}

// GetDepositsByAddress ... Gets a deposit response object provided an L1 address and cursor
func (c *Client) GetDepositsByAddress(l1Address common.Address, cursor string) (*models.DepositResponse, error) {
	var response models.DepositResponse
	url := c.cfg.BaseURL + api.DepositsPath + l1Address.String() + urlParams
	endpoint := fmt.Sprintf(url, cursor, c.cfg.PaginationLimit)

	resp, err := c.doRecordRequest(deposits, endpoint)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resp, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetAllDepositsByAddress ... Gets all deposits provided a L1 address
func (c *Client) GetAllDepositsByAddress(l1Address common.Address) ([]models.DepositItem, error) {
	var deposits []models.DepositItem

	cursor := ""
	for {
		dResponse, err := c.GetDepositsByAddress(l1Address, cursor)
		if err != nil {
			return nil, err
		}

		deposits = append(deposits, dResponse.Items...)

		if !dResponse.HasNextPage {
			break
		}

		cursor = dResponse.Cursor
	}

	return deposits, nil

}

// GetSupplyAssessment ... Returns an assessment of the current supply
// on both L1 and L2. This includes the individual sums of
// (L1/L2) deposits and withdrawals
func (c *Client) GetSupplyAssessment() (*models.BridgeSupplyView, error) {
	url := c.cfg.BaseURL + api.SupplyPath

	resp, err := c.doRecordRequest(sum, url)
	if err != nil {
		return nil, err
	}

	var bsv *models.BridgeSupplyView
	if err := json.Unmarshal(resp, &bsv); err != nil {
		return nil, err
	}

	return bsv, nil
}

// GetAllWithdrawalsByAddress ... Gets all withdrawals provided a L2 address
func (c *Client) GetAllWithdrawalsByAddress(l2Address common.Address) ([]models.WithdrawalItem, error) {
	var withdrawals []models.WithdrawalItem

	cursor := ""
	for {
		wResponse, err := c.GetWithdrawalsByAddress(l2Address, cursor)
		if err != nil {
			return nil, err
		}

		withdrawals = append(withdrawals, wResponse.Items...)

		if !wResponse.HasNextPage {
			break
		}

		cursor = wResponse.Cursor
	}

	return withdrawals, nil
}

// GetWithdrawalsByAddress ... Gets a withdrawal response object provided an L2 address and cursor
func (c *Client) GetWithdrawalsByAddress(l2Address common.Address, cursor string) (*models.WithdrawalResponse, error) {
	var wResponse *models.WithdrawalResponse
	url := c.cfg.BaseURL + api.WithdrawalsPath + l2Address.String() + urlParams

	endpoint := fmt.Sprintf(url, cursor, c.cfg.PaginationLimit)
	resp, err := c.doRecordRequest(withdrawals, endpoint)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resp, &wResponse); err != nil {
		return nil, err
	}

	return wResponse, nil
}
