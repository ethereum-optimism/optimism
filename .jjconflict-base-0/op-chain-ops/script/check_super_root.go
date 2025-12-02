package script

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

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
	// TargetTimestamp is the timestamp to calculate the super root for.
	// If not provided by the user it will be set to the latest timestamp that is finalized on all chains.
	TargetTimestamp *uint64
	// superRoot is the final calculated super root hash
	superRoot common.Hash
	// chainOutputs holds the calculated output root for each chain, ready for super root calculation
	chainOutputs []eth.ChainIDAndOutput
}

// NewSuperRootMigrator creates a new instance of the SuperRootMigrator.
// It requires a logger, a list of L2 execution client RPC endpoints,
// and an optional target timestamp.
func NewSuperRootMigrator(logger log.Logger, rpcEndpoints []string, targetTimestamp *uint64) (*SuperRootMigrator, error) {
	if len(rpcEndpoints) == 0 {
		return nil, errors.New("must provide at least one RPC endpoint")
	}

	migrator := &SuperRootMigrator{
		log:             logger,
		rpcEndpoints:    rpcEndpoints,
		chainSettings:   make(map[string]*ChainSettings),
		TargetTimestamp: targetTimestamp,
	}
	return migrator, nil
}

func NewSuperRootMigratorWithClients(logger log.Logger, clients map[string]*ethclient.Client, targetTimestamp *uint64) (*SuperRootMigrator, error) {
	if len(clients) == 0 {
		return nil, errors.New("must provide at least one client")
	}
	migrator := &SuperRootMigrator{
		log:             logger,
		ethClients:      clients,
		chainSettings:   make(map[string]*ChainSettings),
		TargetTimestamp: targetTimestamp,
	}
	return migrator, nil
}

// Run executes the main logic of the super root migrator within the given context.
func (m *SuperRootMigrator) Run(ctx context.Context) (common.Hash, error) {
	if m.ethClients == nil {
		clients, err := dialClients(ctx, m.rpcEndpoints)
		if err != nil {
			return common.Hash{}, err
		}
		m.ethClients = clients
	}
	// Use the provided context for all operations
	if err := m.initClientsAndFetchIDs(ctx); err != nil {
		return common.Hash{}, fmt.Errorf("failed to initialize clients: %w", err)
	}

	if err := m.findAnchorTimestamp(ctx); err != nil {
		return common.Hash{}, fmt.Errorf("failed to find anchor timestamp: %w", err)
	}

	if err := m.calculateTargetBlockNumbers(ctx); err != nil {
		return common.Hash{}, fmt.Errorf("failed to calculate target block numbers: %w", err)
	}

	if err := m.calculateOutputRoots(ctx); err != nil {
		return common.Hash{}, fmt.Errorf("failed to calculate output roots: %w", err)
	}

	if err := m.calculateSuperRoot(); err != nil {
		return common.Hash{}, fmt.Errorf("failed to calculate super root: %w", err)
	}

	return m.superRoot, nil
}

func dialClients(ctx context.Context, urls []string) (map[string]*ethclient.Client, error) {
	clients := make(map[string]*ethclient.Client)
	for _, endpoint := range urls {
		client, err := ethclient.DialContext(ctx, endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to RPC endpoint %s: %w", endpoint, err)
		}
		clients[endpoint] = client
	}
	return clients, nil
}

// initClientsAndFetchIDs establishes connections to all RPC endpoints
// and retrieves their chain IDs.
func (m *SuperRootMigrator) initClientsAndFetchIDs(ctx context.Context) error {
	for endpoint, client := range m.ethClients {
		chainID, err := client.ChainID(ctx)
		if err != nil {
			// Clean up the client we just created before returning
			client.Close()
			delete(m.ethClients, endpoint)
			return fmt.Errorf("failed to get chain ID from %s: %w", endpoint, err)
		}
		m.log.Info("Connected to client", "url", endpoint, "chainID", chainID)

		m.chainSettings[endpoint] = &ChainSettings{
			ChainID: chainID,
			RPCURL:  endpoint,
		}
	}
	return nil
}

// findAnchorTimestamp finds the minimum finalized block timestamp across all connected chains
// if no target timestamp is provided, otherwise uses the target timestamp.
func (m *SuperRootMigrator) findAnchorTimestamp(ctx context.Context) error {
	// Check if a target timestamp was provided by the user
	if m.TargetTimestamp != nil {
		m.log.Info("Using user-provided timestamp", "timestamp", *m.TargetTimestamp)
		return nil
	}

	var minTimestamp *uint64

	for url, client := range m.ethClients {
		m.log.Debug("Fetching finalized block header", "url", url)
		header, err := client.HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
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
			m.log.Debug("Updated minimum timestamp", "url", url, "minTimestamp", *minTimestamp)
		}
	}

	if minTimestamp == nil {
		return errors.New("no valid finalized timestamps found across connected chains")
	}
	m.TargetTimestamp = minTimestamp
	m.log.Info("Using finalized timestamp", "timestamp", *m.TargetTimestamp)
	return nil
}

func (m *SuperRootMigrator) calculateTargetBlockNumbers(ctx context.Context) error {
	for endpoint, client := range m.ethClients {
		// Get the latest block
		latestBlock, err := client.BlockByNumber(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to get latest block from %s: %w", endpoint, err)
		}

		// Get the parent block
		parentBlock, err := client.BlockByHash(ctx, latestBlock.ParentHash())
		if err != nil {
			return fmt.Errorf("failed to get parent block from %s: %w", endpoint, err)
		}

		// Calculate block time (difference in timestamps between latest and parent blocks)
		blockTime := latestBlock.Time() - parentBlock.Time()
		if blockTime == 0 {
			return fmt.Errorf("block time cannot be zero for chain %s", endpoint)
		}

		// Calculate how many blocks to look back to reach the target timestamp
		timeDiff := latestBlock.Time() - *m.TargetTimestamp
		blocksToLookBack := timeDiff / blockTime
		if timeDiff%blockTime != 0 {
			// Round up the number of blocks to look back to ensure that we get the latest block at or before the timestamp
			blocksToLookBack++
		}

		if blocksToLookBack > latestBlock.NumberU64() {
			return fmt.Errorf("target timestamp is prior to genesis for endpoint %v", endpoint)
		}
		// Compute the target block number
		targetBlockNumber := new(big.Int).SetUint64(latestBlock.Number().Uint64() - blocksToLookBack)

		// Store the computed values in the chain settings
		m.chainSettings[endpoint].BlockTime = blockTime
		m.chainSettings[endpoint].TargetBlockNumber = targetBlockNumber
	}

	return nil
}

// calculateOutputRoots computes the L2 output root for each chain's target block.
func (m *SuperRootMigrator) calculateOutputRoots(ctx context.Context) error {
	// Initialize or clear the chainOutputs slice
	m.chainOutputs = make([]eth.ChainIDAndOutput, 0, len(m.chainSettings))

	for url, settings := range m.chainSettings {
		targetHeader, err := m.ethClients[url].HeaderByNumber(ctx, settings.TargetBlockNumber)
		if err != nil {
			return fmt.Errorf("failed to get header by number %s from %s: %w", settings.TargetBlockNumber, url, err)
		}

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
		m.log.Info("Calculated output root", "url", url, "chainID", settings.ChainID, "block", eth.HeaderBlockID(targetHeader), "timestamp", targetHeader.Time, "outputRoot", settings.OutputRoot)

		// Add the result to the list for final super root calculation
		m.chainOutputs = append(m.chainOutputs, eth.ChainIDAndOutput{
			ChainID: eth.ChainIDFromBig(settings.ChainID),
			Output:  settings.OutputRoot,
		})
	}

	return nil
}

// calculateSuperRoot computes the final super root hash from the sorted chain outputs.
func (m *SuperRootMigrator) calculateSuperRoot() error {
	if len(m.chainOutputs) == 0 {
		return errors.New("cannot compute super root: no chain outputs were generated")
	}

	// Create a SuperV1 structure with the anchor timestamp and chain outputs
	superV1 := eth.NewSuperV1(*m.TargetTimestamp, m.chainOutputs...)

	// Calculate the super root hash
	m.superRoot = common.Hash(eth.SuperRoot(superV1))

	m.log.Info("Super root calculated successfully", "superRoot", m.superRoot.Hex(), "timestamp", *m.TargetTimestamp, "chains", len(m.chainOutputs))
	return nil
}
