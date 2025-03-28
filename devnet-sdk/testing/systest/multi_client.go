package systest

import (
	"context"
	"errors"
	"fmt"

	"math/big"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// RequireNoChainFork checks that the L2 chain has not forked now, and returns a
// function that check again (to be called at the end of the test).
func RequireNoChainFork(t T, chain system.L2Chain, logger log.Logger) func() {
	ctx := t.Context()

	clients := make([]*ethclient.Client, 0, len(chain.Nodes()))
	for _, node := range chain.Nodes() {
		client, err := node.GethClient()
		require.NoError(t, err)
		clients = append(clients, client)
	}

	// We use a multiclient where feasible to automatically check for consistency
	// between the nodes.
	l2MultiClient := NewMultiClient(clients)

	// Setup chain fork detection
	logger.Info("Setting up chain fork detection")
	l2StartHeader, err := l2MultiClient.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	logger.Debug("Got L2 head block", "number", l2StartHeader.Number)
	return func() {
		l2EndHeader, err := l2MultiClient.HeaderByNumber(ctx, nil)
		require.NoError(t, err)
		require.True(t, l2EndHeader.Number.Cmp(l2StartHeader.Number) > 0, "L2 chain should have progressed")
		logger.Debug("Got L2 end block", "number", l2EndHeader.Number)
	}
}

// MultiClient is a simple client that checks hash consistency between underlying clients
type MultiClient struct {
	clients    []*ethclient.Client
	maxRetries int
	retryDelay time.Duration
}

// NewMultiClient creates a new MultiClient with the specified underlying clients
func NewMultiClient(clients []*ethclient.Client) *MultiClient {
	return &MultiClient{
		clients:    clients,
		maxRetries: 3,
		retryDelay: 500 * time.Millisecond,
	}
}

// Close closes all underlying client connections
func (mc *MultiClient) Close() {
	for _, client := range mc.clients {
		client.Close()
	}
}

// BlockByNumber returns a block from the first client while verifying hash consistency
func (mc *MultiClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	if len(mc.clients) == 0 {
		return nil, errors.New("no clients configured")
	}

	// Single client optimization
	if len(mc.clients) == 1 {
		return mc.clients[0].BlockByNumber(ctx, number)
	}

	// Define the query function
	queryFn := func(client *ethclient.Client, num *big.Int) (interface{}, *big.Int, common.Hash, error) {
		block, err := client.BlockByNumber(ctx, num)
		if err != nil {
			return nil, nil, common.Hash{}, fmt.Errorf("failed to get block: %w", err)
		}
		if block == nil {
			return nil, nil, common.Hash{}, fmt.Errorf("returned nil block for number %v", num)
		}
		return block, block.Number(), block.Hash(), nil
	}

	// Fetch with consistency check
	result, err := mc.fetchWithConsistencyCheck(ctx, number, queryFn)
	if err != nil {
		return nil, err
	}

	return result.(*types.Block), nil
}

// HeaderByNumber returns a header from the first client while verifying hash consistency
func (mc *MultiClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if len(mc.clients) == 0 {
		return nil, errors.New("no clients configured")
	}

	// Single client optimization
	if len(mc.clients) == 1 {
		return mc.clients[0].HeaderByNumber(ctx, number)
	}

	// Define the query function
	queryFn := func(client *ethclient.Client, num *big.Int) (interface{}, *big.Int, common.Hash, error) {
		header, err := client.HeaderByNumber(ctx, num)
		if err != nil {
			return nil, nil, common.Hash{}, fmt.Errorf("failed to get header for block number %v: %w", num, err)
		}
		if header == nil {
			return nil, nil, common.Hash{}, fmt.Errorf("returned nil header for number %v", num)
		}
		return header, header.Number, header.Hash(), nil
	}

	// Fetch with consistency check
	result, err := mc.fetchWithConsistencyCheck(ctx, number, queryFn)
	if err != nil {
		return nil, err
	}

	return result.(*types.Header), nil
}

// fetchWithConsistencyCheck implements generic fetching with consistency verification
func (mc *MultiClient) fetchWithConsistencyCheck(
	ctx context.Context,
	number *big.Int,
	queryFn func(*ethclient.Client, *big.Int) (interface{}, *big.Int, common.Hash, error),
) (interface{}, error) {
	// Get from primary client
	// print whether mc.clients[0] is nil
	primaryItem, blockNum, primaryHash, err := queryFn(mc.clients[0], number)
	if err != nil {
		return nil, err
	}

	// Create a hash-only getter for followers
	getFollowerHash := func(client *ethclient.Client, num *big.Int) (common.Hash, error) {
		_, _, hash, err := queryFn(client, num)
		return hash, err
	}

	// Verify consistency with retry for followers
	mismatches, err := mc.verifyFollowersWithRetry(ctx, blockNum, primaryHash, getFollowerHash)
	if err != nil {
		// If err is a chain split error, pass it through
		if strings.Contains(err.Error(), "chain split detected") {
			return nil, err
		}
		return nil, err
	}

	// This should no longer occur with the updated verifyFollowersWithRetry implementation
	// that returns immediately when chain splits are detected
	if mismatches.Len() > 0 {
		return nil, formatHashMismatchError(blockNum, primaryHash, mismatches.clientIndices, mismatches.hashes)
	}

	return primaryItem, nil
}

// mismatches holds information about hash mismatches
type mismatches struct {
	clientIndices []int
	hashes        []common.Hash
}

// Len returns the number of mismatches
func (m mismatches) Len() int {
	return len(m.clientIndices)
}

// verifyFollowersWithRetry checks the hash consistency with retries in case of temporary sync issues
func (mc *MultiClient) verifyFollowersWithRetry(
	ctx context.Context,
	blockNum *big.Int,
	primaryHash common.Hash,
	getHash func(*ethclient.Client, *big.Int) (common.Hash, error),
) (mismatches, error) {
	var result mismatches

	// Track which clients still need verification
	pendingClients := make(map[int]bool)
	for i := 1; i < len(mc.clients); i++ {
		pendingClients[i] = true
	}

	// Try up to maxRetries times
	for attempt := 0; attempt <= mc.maxRetries; attempt++ {
		// If no pending clients, we're done
		if len(pendingClients) == 0 {
			return result, nil
		}

		// If not first attempt, wait before retry
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(mc.retryDelay):
				// Continue after delay
			}
		}

		// Check each pending client
		for clientIdx := range pendingClients {
			hash, err := getHash(mc.clients[clientIdx], blockNum)

			if err != nil {
				// Check if error indicates "not found" - these errors should be retried
				if errors.Is(err, ethereum.NotFound) ||
					strings.Contains(strings.ToLower(err.Error()), "not found") ||
					strings.Contains(strings.ToLower(err.Error()), "nil") {
					// If this is our last attempt, return the error
					if attempt == mc.maxRetries {
						return result, fmt.Errorf("client %d failed after %d attempts: %w", clientIdx, attempt+1, err)
					}
					// Otherwise, try again in the next iteration
					continue
				} else {
					// For other errors, also retry
					if attempt == mc.maxRetries {
						return result, fmt.Errorf("client %d failed after %d attempts: %w", clientIdx, attempt+1, err)
					}
					continue
				}
			}

			// If hash matches, remove from pending
			if hash == primaryHash {
				delete(pendingClients, clientIdx)
			} else {
				// Detected chain split - return error immediately without further retries
				result.clientIndices = append(result.clientIndices, clientIdx)
				result.hashes = append(result.hashes, hash)
				// Format and return chain split error
				return result, formatChainSplitError(blockNum, primaryHash, clientIdx, hash)
			}
		}
	}

	return result, nil
}

// formatChainSplitError creates a descriptive error when a chain split is detected
func formatChainSplitError(blockNum *big.Int, primaryHash common.Hash, clientIdx int, hash common.Hash) error {
	return fmt.Errorf("chain split detected at block #%s: primary=%s, client%d=%s",
		blockNum, primaryHash.Hex()[:10], clientIdx, hash.Hex()[:10])
}

// formatHashMismatchError creates a descriptive error when hash mismatch occurs
func formatHashMismatchError(blockNum *big.Int, primaryHash common.Hash, clientIndices []int, hashes []common.Hash) error {
	msg := fmt.Sprintf("block #%s hash mismatch after retries: primary=%s", blockNum, primaryHash.Hex()[:10])
	for i, idx := range clientIndices {
		msg += fmt.Sprintf(", client%d=%s", idx, hashes[i].Hex()[:10])
	}
	return errors.New(msg)
}
