package client

import (
	"fmt"
	"io"
	"net/http"

	"encoding/json"

	"github.com/ethereum-optimism/optimism/indexer/api"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
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

// IndexerClient ... Indexer client struct
type IndexerClient struct {
	cfg *Config
	c   *http.Client
	m   node.Metricer
}

// NewClient ... Construct a new indexer client
func NewClient(cfg *Config, m node.Metricer) (*IndexerClient, error) {
	if cfg.PaginationLimit == 0 {
		cfg.PaginationLimit = defaultPagingLimit
	}

	c := &http.Client{}
	return &IndexerClient{cfg: cfg, c: c, m: m}, nil
}

// GetAllWithdrawalsByAddress ... Gets all withdrawals by address
func (ic *IndexerClient) GetAllWithdrawalsByAddress(l2Address string) ([]database.L2BridgeWithdrawalWithTransactionHashes, error) {
	var withdrawals []database.L2BridgeWithdrawalWithTransactionHashes

	cursor := ""
	for {
		wResponse, err := ic.GetWithdrawalsByAddress(l2Address, cursor)
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

// GetWithdrawalsByAddress ... Gets a withdrawal response object provided an L2 address
func (ic *IndexerClient) GetWithdrawalsByAddress(l2Address string, cursor string) (*database.L2BridgeWithdrawalsResponse, error) {
	var wResponse *database.L2BridgeWithdrawalsResponse

	endpoint := fmt.Sprintf(ic.cfg.URL+api.WithdrawalsPath+l2Address+urlParams, cursor, ic.cfg.PaginationLimit)
	resp, err := ic.c.Get(endpoint)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, &wResponse); err != nil {
		return nil, err
	}

	return wResponse, nil
}
