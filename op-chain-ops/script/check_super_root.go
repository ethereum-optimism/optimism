package script

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ChainSettings maintains chain-specific configuration and state
// required for super root calculation. Contains both static configuration
// from the rollup genesis and dynamic parameters derived during migration.
type ChainSettings struct {
	// ChainID is the Ethereum chain identifier for this L2 chain
	ChainID *big.Int
	// RPCURL is the endpoint used to connect to the chain's execution client
	RPCURL string
	// RollupGenesis contains the L1 and L2 genesis block info and system config
	RollupGenesis *rollup.Genesis
	// TargetBlock is the header of the L2 block found at or just before the anchor timestamp
	TargetBlock *types.Header
	// OutputRoot is the calculated L2 output root for the TargetBlock
	OutputRoot eth.Bytes32
	// BlockTime is the estimated time between L2 blocks in seconds,
	// derived from observed finalized block timestamps
	BlockTime uint64
	// EstimatedGenesisTimestamp is the estimated timestamp of the L2 genesis block,
	// derived from the finalized block and estimated block time
	EstimatedGenesisTimestamp uint64
	// TargetBlockNumber is the computed block number to look back for the anchor timestamp
	TargetBlockNumber *big.Int
}

// SuperRootMigrator orchestrates the process of calculating a super root
// based on the common finalized state of multiple L2 chains.
type SuperRootMigrator struct {
	// log provides structured logging capabilities
	log log.Logger
	// rpcEndpoints is the list of L2 EL RPC URLs provided as input
	rpcEndpoints []string
	// ethClients maps RPC URLs to their corresponding ethclient instances
	ethClients map[string]*ethclient.Client
	// chainSettings maps RPC URLs to their derived settings and state
	chainSettings map[string]*ChainSettings
	// anchorTimestamp is the latest timestamp common to all chains' finalized blocks
	// or the user-provided target timestamp.
	anchorTimestamp uint64
	// TargetTimestamp is the optional user-provided timestamp to use for the anchor.
	TargetTimestamp uint64
	// superRoot is the final calculated super root hash
	superRoot common.Hash
	// chainOutputs holds the calculated output root for each chain, ready for super root calculation
	chainOutputs []eth.ChainIDAndOutput
}

// NewSuperRootMigrator creates a new instance of the SuperRootMigrator.
// It requires a logger, a list of L2 execution client RPC endpoints,
// and an optional target timestamp.
func NewSuperRootMigrator(logger log.Logger, rpcEndpoints []string, targetTimestamp *uint64) (*SuperRootMigrator, error) {
	if logger == nil {
		logger = log.New("service", "super-root-migrator")
	}
	if len(rpcEndpoints) == 0 {
		return nil, errors.New("must provide at least one RPC endpoint")
	}
	if targetTimestamp == nil {
		return nil, errors.New("must provide a target timestamp")
	}

	return &SuperRootMigrator{
		log:             logger,
		rpcEndpoints:    rpcEndpoints,
		ethClients:      make(map[string]*ethclient.Client),
		chainSettings:   make(map[string]*ChainSettings),
		TargetTimestamp: *targetTimestamp, // Store the provided timestamp
	}, nil
}

// Run executes the main logic of the super root migrator within the given context.
func (m *SuperRootMigrator) Run(ctx context.Context) error {
	m.log.Info("Starting super root calculation process")

	// Use the provided context for all operations
	if err := m.initClientsAndFetchIDs(ctx); err != nil {
		return fmt.Errorf("failed to initialize clients: %w", err)
	}

	if err := m.calculateTargetBlockNumbers(ctx); err != nil {
		return fmt.Errorf("failed to calculate target block numbers: %w", err)
	}

	if err := m.findAnchorTimestamp(ctx); err != nil {
		return fmt.Errorf("failed to find anchor timestamp: %w", err)
	}
	m.log.Info("Found common anchor timestamp", "timestamp", m.anchorTimestamp)

	if err := m.calculateOutputRoots(ctx); err != nil {
		return fmt.Errorf("failed to calculate output roots: %w", err)
	}

	if err := m.calculateSuperRoot(); err != nil {
		return fmt.Errorf("failed to calculate super root: %w", err)
	}

	m.log.Info("Super root calculation process finished successfully")
	return nil
}

// initClientsAndFetchIDs establishes connections to all RPC endpoints
// and retrieves their chain IDs.
func (m *SuperRootMigrator) initClientsAndFetchIDs(ctx context.Context) error {
	m.log.Info("Initializing clients and fetching chain IDs...")

	for _, endpoint := range m.rpcEndpoints {
		m.log.Debug("Dialing RPC endpoint", "url", endpoint)
		client, err := ethclient.DialContext(ctx, endpoint)
		if err != nil {
			return fmt.Errorf("failed to connect to RPC endpoint %s: %w", endpoint, err)
		}
		m.ethClients[endpoint] = client
		m.log.Info("Connected to client", "url", endpoint)

		chainID, err := client.ChainID(ctx)
		if err != nil {
			// Clean up the client we just created before returning
			client.Close()
			delete(m.ethClients, endpoint)
			return fmt.Errorf("failed to get chain ID from %s: %w", endpoint, err)
		}
		m.log.Info("Fetched chain ID", "url", endpoint, "chain_id", chainID)

		m.chainSettings[endpoint] = &ChainSettings{
			ChainID: chainID,
			RPCURL:  endpoint,
		}
	}

	m.log.Info("Successfully initialized clients", "count", len(m.ethClients))
	return nil
}

func (m *SuperRootMigrator) calculateTargetBlockNumbers(ctx context.Context) error {
	m.log.Info("Calculating target block numbers...")

	for endpoint, client := range m.ethClients {
		// Get the latest block
		latestBlock, err := client.BlockByNumber(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to get latest block from %s: %w", endpoint, err)
		}

		// Get the parent block
		parentBlock, err := client.BlockByNumber(ctx, big.NewInt(latestBlock.Number().Int64()-1))
		if err != nil {
			return fmt.Errorf("failed to get parent block from %s: %w", endpoint, err)
		}

		// Calculate block time (difference in timestamps between latest and parent blocks)
		blockTime := latestBlock.Time() - parentBlock.Time()
		if blockTime == 0 {
			return fmt.Errorf("block time cannot be zero for chain %s", endpoint)
		}

		// Calculate how many blocks to look back to reach the target timestamp
		timeDiff := latestBlock.Time() - m.TargetTimestamp
		blocksToLookBack := (timeDiff / blockTime) + 1

		// Compute the target block number
		targetBlockNumber := big.NewInt(latestBlock.Number().Int64() - int64(blocksToLookBack))

		// Store the computed values in the chain settings
		m.chainSettings[endpoint].BlockTime = blockTime
		m.chainSettings[endpoint].TargetBlockNumber = targetBlockNumber

		m.log.Info("Calculated target block number",
			"url", endpoint,
			"blockTime", blockTime,
			"blocksToLookBack", blocksToLookBack,
			"targetBlockNumber", targetBlockNumber)
	}

	m.log.Info("Successfully calculated target block numbers for all chains")
	return nil
}

// / Find anchor timestamp and blocks associated. Update the chain settings with the actual anchor info
func (m *SuperRootMigrator) findAnchorTimestamp(ctx context.Context) error {
	m.log.Info("Finding common anchor timestamp...")

	// Initialize block numbers for each chain
	blockNumbers := make(map[string]*big.Int)
	for endpoint, settings := range m.chainSettings {
		blockNumbers[endpoint] = new(big.Int).Set(settings.TargetBlockNumber)
	}

	// Track the current timestamps for each chain
	timestamps := make(map[string]uint64)

	// Maximum number of iterations to prevent infinite loops
	maxIterations := 1000
	iteration := 0

	for iteration < maxIterations {
		iteration++

		// Fetch the current block for each chain
		for endpoint, blockNumber := range blockNumbers {
			client := m.ethClients[endpoint]
			block, err := client.BlockByNumber(ctx, blockNumber)
			if err != nil {
				return fmt.Errorf("failed to fetch block %v from %s: %w", blockNumber, endpoint, err)
			}
			timestamps[endpoint] = block.Time()
		}

		// Check if all timestamps match
		var commonTimestamp *uint64
		for _, timestamp := range timestamps {
			if commonTimestamp == nil {
				commonTimestamp = &timestamp
			} else if timestamp != *commonTimestamp {
				commonTimestamp = nil
				break
			}
		}

		if commonTimestamp != nil {
			m.anchorTimestamp = *commonTimestamp
			m.log.Info("Found common anchor timestamp", "timestamp", m.anchorTimestamp)

			// Record the blocks for each chain that match the anchor timestamp
			for endpoint, blockNumber := range blockNumbers {
				client := m.ethClients[endpoint]
				block, err := client.BlockByNumber(ctx, blockNumber)
				if err != nil {
					return fmt.Errorf("failed to fetch block %v from %s: %w", blockNumber, endpoint, err)
				}
				m.chainSettings[endpoint].TargetBlock = block.Header()
				m.log.Info("Recorded block for chain", "url", endpoint, "block_number", blockNumber, "block_hash", block.Hash())
			}

			return nil
		}

		// Find the chain with the highest timestamp and walk back one block
		var maxEndpoint string
		var maxTimestamp uint64
		for endpoint, timestamp := range timestamps {
			if timestamp > maxTimestamp {
				maxEndpoint = endpoint
				maxTimestamp = timestamp
			}
		}

		// Decrement the block number for the chain with the highest timestamp
		blockNumbers[maxEndpoint].Sub(blockNumbers[maxEndpoint], big.NewInt(1))
	}

	return fmt.Errorf("failed to find a common anchor timestamp after %d iterations", maxIterations)
}

// calculateOutputRoots computes the L2 output root for each chain's target block.
func (m *SuperRootMigrator) calculateOutputRoots(ctx context.Context) error {
	m.log.Info("Calculating output roots for target blocks...")
	// Initialize or clear the chainOutputs slice
	m.chainOutputs = make([]eth.ChainIDAndOutput, 0, len(m.chainSettings))

	for url, settings := range m.chainSettings {
		if settings.TargetBlock == nil {
			// This should ideally be caught earlier, but double-check
			return fmt.Errorf("missing target block for output root calculation on chain %s (ID: %s)", url, settings.ChainID)
		}
		targetHeader := settings.TargetBlock
		m.log.Debug("Calculating output root", "url", url, "chain_id", settings.ChainID, "block_num", targetHeader.Number, "block_hash", targetHeader.Hash())

		// Isthmus assumes WithdrawalsHash is present in the header.
		if targetHeader.WithdrawalsHash == nil {
			return fmt.Errorf("target block %d (%s) on chain %s (ID: %s) is missing withdrawals hash, required for Isthmus output root calculation",
				targetHeader.Number.Uint64(), targetHeader.Hash(), url, settings.ChainID)
		}

		// Construct OutputV0 using StateRoot, WithdrawalsHash (as MessagePasserStorageRoot), and BlockHash
		output := &eth.OutputV0{
			StateRoot:                eth.Bytes32(targetHeader.Root),
			MessagePasserStorageRoot: eth.Bytes32(*targetHeader.WithdrawalsHash),
			BlockHash:                targetHeader.Hash(),
		}

		// Calculate the output root hash
		settings.OutputRoot = eth.OutputRoot(output)
		m.log.Debug("Calculated output root", "url", url, "chain_id", settings.ChainID, "output_root", settings.OutputRoot)

		// Add the result to the list for final super root calculation
		m.chainOutputs = append(m.chainOutputs, eth.ChainIDAndOutput{
			ChainID: eth.ChainIDFromBig(settings.ChainID),
			Output:  settings.OutputRoot,
		})
	}

	m.log.Info("Calculated and sorted all chain output roots", "count", len(m.chainOutputs))

	return nil
}

// calculateSuperRoot computes the final super root hash from the sorted chain outputs.
func (m *SuperRootMigrator) calculateSuperRoot() error {
	m.log.Info("Calculating final super root...")
	if len(m.chainOutputs) == 0 {
		return errors.New("cannot compute super root: no chain outputs were generated")
	}

	// Create a SuperV1 structure with the anchor timestamp and chain outputs
	superV1 := eth.NewSuperV1(m.anchorTimestamp, m.chainOutputs...)

	// Calculate the super root hash
	m.superRoot = common.Hash(eth.SuperRoot(superV1))

	m.log.Info("Super root calculated successfully", "super_root", m.superRoot.Hex(), "timestamp", m.anchorTimestamp, "chain_count", len(m.chainOutputs))
	return nil
}
