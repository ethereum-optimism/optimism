package dsl

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// HeaderProvider interface for multi-client operations
type HeaderProvider interface {
	InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error)
	InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
}

// CheckForChainFork checks that the L2 chain has not forked now, and returns a
// function that check again (to be called at the end of the test). An error is
// returned from this function (and the returned function) if a chain fork has
// been detected.
func CheckForChainFork(ctx context.Context, networks []*L2Network, logger log.Logger) (func(bool) error, error) {
	var allClients []HeaderProvider
	for _, network := range networks {
		clients, err := getEthClientsFromL2Network(network)
		if err != nil {
			return nil, fmt.Errorf("failed to get eth clients from network %s: %w", network.String(), err)
		}
		allClients = append(allClients, clients...)
	}

	return checkForChainFork(ctx, allClients, logger)
}

// getEthClientsFromL2Network extracts HeaderProvider clients from an L2Network
func getEthClientsFromL2Network(network *L2Network) ([]HeaderProvider, error) {
	stackNetwork := network.Escape()
	hps := make([]HeaderProvider, 0, len(stackNetwork.L2ELNodes()))
	for _, n := range stackNetwork.L2ELNodes() {
		ethClient := n.L2EthClient()
		if !regexp.MustCompile(`snapsync-\d+$`).MatchString(n.ID().Key()) {
			hps = append(hps, ethClient)
		}
	}
	return hps, nil
}

func checkForChainFork(ctx context.Context, clients []HeaderProvider, logger log.Logger) (func(bool) error, error) {
	l2MultiClient := NewMultiClient(clients)

	// Setup chain fork detection
	logger.Info("Running fork detection precheck")
	l2StartInfo, err := l2MultiClient.InfoByLabel(ctx, eth.Unsafe)
	if err != nil {
		return nil, fmt.Errorf("fork detection precheck failed: %w", err)
	}

	return func(failed bool) error {
		logger.Info("Running fork detection postcheck")
		l2EndInfo, err := l2MultiClient.InfoByLabel(ctx, eth.Unsafe)
		if err != nil {
			return fmt.Errorf("fork detection postcheck failed: %w", err)
		}
		if l2EndInfo.NumberU64() <= l2StartInfo.NumberU64() {
			if !failed {
				return fmt.Errorf("L2 chain has not progressed: start=%d, end=%d", l2StartInfo.NumberU64(), l2EndInfo.NumberU64())
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

// InfoByNumber returns block info from the first client while verifying hash consistency
func (mc *MultiClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	if len(mc.clients) == 0 {
		return nil, errors.New("no clients configured")
	}

	// Single client optimization
	info, err := mc.clients[0].InfoByNumber(ctx, number)
	if err != nil || len(mc.clients) == 1 {
		return info, err
	}

	// Fetch with consistency check
	err = mc.verifyFollowersWithRetry(ctx, big.NewInt(int64(number)), info.Hash())
	return info, err
}

// InfoByLabel returns block info from the first client while verifying hash consistency
func (mc *MultiClient) InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error) {
	if len(mc.clients) == 0 {
		return nil, errors.New("no clients configured")
	}

	info, err := mc.clients[0].InfoByLabel(ctx, label)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("no block info found for label %v", label)
	}
	if len(mc.clients) == 1 {
		return info, nil
	}

	// Verify consistency with retry for followers
	err = mc.verifyFollowersWithRetry(ctx, big.NewInt(int64(info.NumberU64())), info.Hash())

	return info, err
}

// InfoByHash returns block info from the first client while verifying hash consistency
func (mc *MultiClient) InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error) {
	if len(mc.clients) == 0 {
		return nil, errors.New("no clients configured")
	}

	info, err := mc.clients[0].InfoByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("no block info found for hash %v", hash)
	}
	if len(mc.clients) == 1 {
		return info, nil
	}

	// Verify consistency with retry for followers
	err = mc.verifyFollowersWithRetry(ctx, big.NewInt(int64(info.NumberU64())), info.Hash())

	return info, err
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
				info, err := client.InfoByNumber(ctx, blockNum.Uint64())
				if err != nil {
					return common.Hash{}, err
				}
				return info.Hash(), nil
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

// MultiClientForL2Network creates a MultiClient from an L2Network
func MultiClientForL2Network(network *L2Network) (*MultiClient, error) {
	clients := make([]HeaderProvider, 0)
	for _, node := range network.Escape().L2ELNodes() {
		clients = append(clients, node.EthClient())
	}
	return NewMultiClient(clients), nil
}

// MultiClientForL1Network creates a MultiClient from an L1Network
func MultiClientForL1Network(network *L1Network) (*MultiClient, error) {
	clients := make([]HeaderProvider, 0)
	for _, node := range network.Escape().L1ELNodes() {
		clients = append(clients, node.EthClient())
	}
	return NewMultiClient(clients), nil
}
