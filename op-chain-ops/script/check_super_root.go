package script

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"

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
	anchorTimestamp uint64
	// superRoot is the final calculated super root hash
	superRoot common.Hash
	// chainOutputs holds the calculated output root for each chain, ready for super root calculation
	chainOutputs []eth.ChainIDAndOutput
}

// NewSuperRootMigrator creates a new instance of the SuperRootMigrator.
// It requires a logger and a list of L2 execution client RPC endpoints.
func NewSuperRootMigrator(logger log.Logger, rpcEndpoints []string) (*SuperRootMigrator, error) {
	if logger == nil {
		// Default logger if none provided
		logger = log.New("service", "super-root-migrator")
	}
	if len(rpcEndpoints) == 0 {
		return nil, errors.New("must provide at least one RPC endpoint")
	}

	return &SuperRootMigrator{
		log:           logger,
		rpcEndpoints:  rpcEndpoints,
		ethClients:    make(map[string]*ethclient.Client),
		chainSettings: make(map[string]*ChainSettings),
	}, nil
}

// Run executes the main logic of the super root migrator within the given context.
func (m *SuperRootMigrator) Run(ctx context.Context) error {
	m.log.Info("Starting super root calculation process")

	// Use the provided context for all operations
	if err := m.initClientsAndFetchIDs(ctx); err != nil {
		return fmt.Errorf("failed to initialize clients: %w", err)
	}

	if err := m.findAnchorTimestamp(ctx); err != nil {
		return fmt.Errorf("failed to find anchor timestamp: %w", err)
	}
	m.log.Info("Found common anchor timestamp", "timestamp", m.anchorTimestamp)

	if err := m.deriveParamsAndFindTargetBlocks(ctx); err != nil {
		return fmt.Errorf("failed to derive parameters and find target blocks: %w", err)
	}

	if err := m.calculateOutputRoots(ctx); err != nil {
		return fmt.Errorf("failed to calculate output roots: %w", err)
	}

	if err := m.calculateSuperRoot(); err != nil {
		return fmt.Errorf("failed to calculate super root: %w", err)
	}

	m.printResults()

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

// findAnchorTimestamp finds the minimum finalized block timestamp across all connected chains.
func (m *SuperRootMigrator) findAnchorTimestamp(ctx context.Context) error {
	m.log.Info("Finding common anchor timestamp across all chains...")
	var minTimestamp *uint64

	for url, client := range m.ethClients {
		m.log.Debug("Fetching finalized block header", "url", url)
		header, err := client.HeaderByNumber(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to get finalized header from %s: %w", url, err)
		}
		if header == nil {
			return fmt.Errorf("received nil finalized header from %s", url)
		}
		m.log.Debug("Got finalized header", "url", url, "number", header.Number, "timestamp", header.Time)

		if minTimestamp == nil || header.Time < *minTimestamp {
			timestamp := header.Time
			minTimestamp = &timestamp
			m.log.Debug("Updated minimum timestamp", "url", url, "new_min_timestamp", *minTimestamp)
		}
	}

	if minTimestamp == nil {
		return errors.New("no valid finalized timestamps found across connected chains")
	}
	m.anchorTimestamp = *minTimestamp
	m.log.Info("Determined common anchor timestamp", "timestamp", m.anchorTimestamp)
	return nil
}

// deriveParamsAndFindTargetBlocks derives block time, estimates genesis timestamp,
// and finds the specific block at or just before the anchor timestamp for each chain.
func (m *SuperRootMigrator) deriveParamsAndFindTargetBlocks(ctx context.Context) error {
	m.log.Info("Deriving chain parameters and finding target blocks...")

	for url, settings := range m.chainSettings {
		client := m.ethClients[url]
		m.log.Debug("Processing chain", "url", url, "chain_id", settings.ChainID)

		finalizedHeader, err := client.HeaderByNumber(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to get finalized block header for %s: %w", url, err)
		}
		if finalizedHeader == nil || finalizedHeader.Number == nil {
			return fmt.Errorf("received invalid finalized header for %s", url)
		}
		if finalizedHeader.Number.Sign() <= 0 {
			// Cannot derive block time if finalized is genesis or block 0
			m.log.Warn("Finalized block is genesis or block 0, cannot derive block time. Using fallback 2s.", "url", url)
			settings.BlockTime = 2
			settings.EstimatedGenesisTimestamp = finalizedHeader.Time // Genesis time is finalized time
		} else {
			finalizedNum := finalizedHeader.Number.Uint64()
			finalizedTime := finalizedHeader.Time

			// Fetch parent of finalized block
			parentNumBig := new(big.Int).Sub(finalizedHeader.Number, big.NewInt(1))
			parentHeader, err := client.HeaderByNumber(ctx, parentNumBig)
			if err != nil {
				// Fallback if parent fetch fails
				m.log.Warn("Failed to get parent of finalized block, using fallback block time 2s", "url", url, "err", err)
				settings.BlockTime = 2
			} else if parentHeader == nil {
				m.log.Warn("Parent of finalized block is nil, using fallback block time 2s", "url", url)
				settings.BlockTime = 2
			} else if finalizedTime <= parentHeader.Time {
				// Timestamps not increasing, indicates issue or maybe 0 block time? Use fallback.
				m.log.Warn("Finalized block timestamp not greater than parent, using fallback block time 2s", "url", url)
				settings.BlockTime = 2
			} else {
				settings.BlockTime = finalizedTime - parentHeader.Time
			}

			if settings.BlockTime == 0 { // Avoid division by zero if fallback wasn't used but time diff was 0
				m.log.Warn("Derived block time is zero, using fallback 2s", "url", url)
				settings.BlockTime = 2
			}

			// Estimate genesis timestamp (assuming genesis block number 0)
			// Clamp estimated genesis time to be <= finalized time
			estGenesisTime := int64(finalizedTime) - int64(finalizedNum*settings.BlockTime)
			if estGenesisTime < 0 {
				estGenesisTime = 0 // Genesis time cannot be negative
			}
			if uint64(estGenesisTime) > finalizedTime {
				settings.EstimatedGenesisTimestamp = finalizedTime
			} else {
				settings.EstimatedGenesisTimestamp = uint64(estGenesisTime)
			}
		}
		m.log.Debug("Derived parameters", "url", url, "block_time", settings.BlockTime, "est_genesis_time", settings.EstimatedGenesisTimestamp)

		// Start search from finalized block, walking backwards
		currentHeader := finalizedHeader
		for currentHeader.Time > m.anchorTimestamp {
			m.log.Debug("Searching for target block", "url", url, "current_num", currentHeader.Number, "current_time", currentHeader.Time, "anchor_time", m.anchorTimestamp)
			if currentHeader.Number == nil || currentHeader.Number.Sign() <= 0 {
				// If we reach block 0 and its time is still > anchor, something is wrong or anchor is before genesis
				return fmt.Errorf("reached genesis block 0 for %s, but its time %d is still after anchor %d", url, currentHeader.Time, m.anchorTimestamp)
			}

			parentNumBig := new(big.Int).Sub(currentHeader.Number, big.NewInt(1))
			parentHeader, err := client.HeaderByNumber(ctx, parentNumBig)
			if err != nil {
				return fmt.Errorf("failed to get parent block %d for %s during search: %w", parentNumBig.Uint64(), url, err)
			}
			if parentHeader == nil {
				return fmt.Errorf("received nil parent block %d for %s during search", parentNumBig.Uint64(), url)
			}

			// Basic check to prevent infinite loops if timestamps are weird (e.g., constant or decreasing going down)
			if parentHeader.Time >= currentHeader.Time && currentHeader.Number.Uint64() > 0 {
				return fmt.Errorf("block timestamps not decreasing during search for %s between block %d (%d) and %d (%d)",
					url, currentHeader.Number.Uint64(), currentHeader.Time, parentHeader.Number.Uint64(), parentHeader.Time)
			}
			currentHeader = parentHeader
		}

		// After loop, currentHeader.Time <= m.anchorTimestamp
		settings.TargetBlock = currentHeader
		m.log.Info("Found target block for chain", "url", url, "chain_id", settings.ChainID, "block_num", settings.TargetBlock.Number, "block_hash", settings.TargetBlock.Hash(), "block_time", settings.TargetBlock.Time)
	}
	return nil
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

	// Ensure chainOutputs are sorted by ChainID for deterministic super root calculation
	sort.SliceStable(m.chainOutputs, func(i, j int) bool {
		// Use ChainID's Cmp method for comparison since it's a wrapper around uint256.Int
		return m.chainOutputs[i].ChainID.Cmp(m.chainOutputs[j].ChainID) < 0
	})
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

// printResults prints the calculated super root and verification inputs.
func (m *SuperRootMigrator) printResults() {
	m.log.Info("Printing results...")
	fmt.Printf("Calculated Super Root: %s\n", m.superRoot.Hex())
	fmt.Printf("Anchor Timestamp: %d\n", m.anchorTimestamp)
	fmt.Println("Verification Inputs:")
	for rpcURL, settings := range m.chainSettings {
		fmt.Printf("  RPC URL: %s\n", rpcURL)
		if settings.ChainID != nil {
			fmt.Printf("    Chain ID: %s\n", settings.ChainID.String())
		} else {
			fmt.Println("    Chain ID: <not fetched>")
		}
		if settings.TargetBlock != nil {
			fmt.Printf("    Target Block Number: %d\n", settings.TargetBlock.Number)
			fmt.Printf("    Target Block Hash: %s\n", settings.TargetBlock.Hash().Hex())
		} else {
			fmt.Println("    Target Block: <not found>")
		}
		fmt.Printf("    Output Root: %s\n", common.Hash(settings.OutputRoot).Hex())
	}
}
