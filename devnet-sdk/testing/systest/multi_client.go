package systest

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"

	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type HeaderProvider interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	Close()
}

var _ HeaderProvider = (*ethclient.Client)(nil)

func getEthClients(chain system.Chain) ([]HeaderProvider, error) {
	hps := make([]HeaderProvider, 0, len(chain.Nodes()))
	for _, n := range chain.Nodes() {
		gethCl, err := n.GethClient()

		if err != nil {
			return nil, fmt.Errorf("failed to get geth client: %w", err)
		}
		if !regexp.MustCompile(`snapsync-\d+$`).MatchString(n.Name()) {
			hps = append(hps, gethCl)
		}
	}
	return hps, nil
}

// CheckForChainFork checks that the L2 chain has not forked now, and returns a
// function that check again (to be called at the end of the test). An error is
// returned from this function (and the returned function) if a chain fork has
// been detected.
func CheckForChainFork(ctx context.Context, chain system.L2Chain, logger log.Logger) (func(bool) error, error) {
	clients, err := getEthClients(chain)
	if err != nil {
		return nil, fmt.Errorf("failed to get eth clients: %w", err)
	}

	return checkForChainFork(ctx, clients, logger)
}

func checkForChainFork(ctx context.Context, clients []HeaderProvider, logger log.Logger) (func(bool) error, error) {
	l2MultiClient := NewMultiClient(clients)

	// Setup chain fork detection
	logger.Info("Running fork detection precheck")
	l2StartHeader, err := l2MultiClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("fork detection precheck failed: %w", err)
	}

	return func(failed bool) error {
		logger.Info("Running fork detection postcheck")
		l2EndHeader, err := l2MultiClient.HeaderByNumber(ctx, nil)
		if err != nil {
			return fmt.Errorf("fork detection postcheck failed: %w", err)
		}
		if l2EndHeader.Number.Cmp(l2StartHeader.Number) <= 0 {
			if !failed {
				return fmt.Errorf("L2 chain has not progressed: start=%s, end=%s", l2StartHeader.Number, l2EndHeader.Number)
			} else {
				logger.Debug("L2 chain has not progressed, but the test failed so we will not error again")
			}
		}
		return nil
	}, nil
}

// MultiClient is a simple client that checks hash consistency between underlying clients
type MultiClient struct {
	clients       []HeaderProvider
	retryStrategy retry.Strategy
	maxAttempts   int
}

// NewMultiClient creates a new MultiClient with the specified underlying clients
func NewMultiClient(clients []HeaderProvider) *MultiClient {
	return &MultiClient{
		clients:       clients,
		maxAttempts:   3,
		retryStrategy: retry.Fixed(500 * time.Millisecond),
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
	block, err := mc.clients[0].BlockByNumber(ctx, number)
	if err != nil || len(mc.clients) == 1 {
		return block, err
	}

	// Fetch with consistency check
	err = mc.verifyFollowersWithRetry(ctx, number, block.Hash())
	return block, err
}

// HeaderByNumber returns a header from the first client while verifying hash consistency
func (mc *MultiClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if len(mc.clients) == 0 {
		return nil, errors.New("no clients configured")
	}

	header, err := mc.clients[0].HeaderByNumber(ctx, number)
	if err != nil || len(mc.clients) == 1 {
		return header, err
	}

	// Verify consistency with retry for followers
	err = mc.verifyFollowersWithRetry(ctx, header.Number, header.Hash())

	return header, err
}

// verifyFollowersWithRetry checks hash consistency with retries in case of temporary sync issues
func (mc *MultiClient) verifyFollowersWithRetry(
	ctx context.Context,
	blockNum *big.Int,
	primaryHash common.Hash,
) error {
	var wg sync.WaitGroup
	errs := make(chan error)

	// Track which clients still need verification
	for clientIndex, c := range mc.clients[1:] {
		actualIndex := clientIndex + 1
		client := c // copy so the goroutine closure has a stable reference
		wg.Add(1)
		go func() {
			defer wg.Done()
			hash, err := retry.Do(ctx, mc.maxAttempts, mc.retryStrategy, func() (common.Hash, error) {
				header, err := client.HeaderByNumber(ctx, blockNum)
				if err != nil {
					return common.Hash{}, err
				}
				return header.Hash(), nil
			})
			if err != nil {
				errs <- err
				return
			}
			// Detect chain split
			if hash != primaryHash {
				errs <- formatChainSplitError(blockNum, primaryHash, actualIndex, hash)
				return
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errs)
	}()

	allErrs := []error{}
	for err := range errs {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return errors.Join(allErrs...)
	}

	return nil
}

// formatChainSplitError creates a descriptive error when a chain split is detected
func formatChainSplitError(blockNum *big.Int, primaryHash common.Hash, clientIdx int, hash common.Hash) error {
	return fmt.Errorf("chain split detected at block #%s: primary=%s, client%d=%s",
		blockNum, primaryHash.Hex()[:10], clientIdx, hash.Hex()[:10])
}
