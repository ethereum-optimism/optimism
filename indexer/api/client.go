package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
)

// Client defines the methods for interfacing with the indexer API.
type Client struct {
	url string
	c   *http.Client
}

// NewClient creates a client that uses the given RPC client.
func NewClient(c *http.Client, url string) *Client {
	return &Client{
		url: url,
		c:   c,
	}
}

// Close closes the underlying RPC connection.
func (ec *Client) Close() {
	ec.c.CloseIdleConnections()
}

// GetDepositsByAddress returns all associated (L1->L2) deposits for a provided L1 address.
func (ic *Client) GetDepositsByAddress(addr common.Address) ([]*database.L1BridgeDepositWithTransactionHashes, error) {
	var deposits []*database.L1BridgeDepositWithTransactionHashes
	resp, err := ic.c.Get(ic.url + depositPath + addr.Hex())
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&deposits); err != nil {
		return nil, err
	}

	return deposits, nil
}

// GetWithdrawalsByAddress returns all associated (L2->L1) withdrawals for a provided L2 address.
func (ic *Client) GetWithdrawalsByAddress(addr common.Address) ([]*database.L2BridgeWithdrawalWithTransactionHashes, error) {
	var withdrawals []*database.L2BridgeWithdrawalWithTransactionHashes
	resp, err := ic.c.Get(ic.url + depositPath + addr.Hex())
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&withdrawals); err != nil {
		return nil, err
	}

	return withdrawals, nil
}
