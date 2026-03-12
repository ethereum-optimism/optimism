package script

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/dial"
)

const latestFinalizedLookupTimestamp = ^uint64(0)

// SuperRootMigrator fetches a verified super root from SuperNode.
type SuperRootMigrator struct {
	log             log.Logger
	superNodeRPC    string
	client          apis.SupernodeQueryAPI
	TargetTimestamp *uint64
}

// NewSuperRootMigrator creates a new instance of the SuperRootMigrator for CLI use.
func NewSuperRootMigrator(logger log.Logger, superNodeRPC string, targetTimestamp *uint64) (*SuperRootMigrator, error) {
	if superNodeRPC == "" {
		return nil, errors.New("must provide a SuperNode RPC endpoint")
	}

	return &SuperRootMigrator{
		log:             logger,
		superNodeRPC:    superNodeRPC,
		TargetTimestamp: targetTimestamp,
	}, nil
}

// NewSuperRootMigratorWithClient creates a new migrator with an injected SuperNode client for testing.
func NewSuperRootMigratorWithClient(logger log.Logger, client apis.SupernodeQueryAPI, targetTimestamp *uint64) (*SuperRootMigrator, error) {
	if client == nil {
		return nil, errors.New("must provide a SuperNode client")
	}

	return &SuperRootMigrator{
		log:             logger,
		client:          client,
		TargetTimestamp: targetTimestamp,
	}, nil
}

// Run fetches the requested verified super root.
func (m *SuperRootMigrator) Run(ctx context.Context) (common.Hash, error) {
	client := m.client
	var closeFn func()
	if client == nil {
		superNodeClient, err := dial.DialSuperNodeClientWithTimeout(ctx, m.log, m.superNodeRPC)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to dial SuperNode RPC %s: %w", m.superNodeRPC, err)
		}
		client = superNodeClient
		closeFn = superNodeClient.Close
	}
	if closeFn != nil {
		defer closeFn()
	}

	targetTimestamp := m.TargetTimestamp
	if targetTimestamp == nil {
		initialResp, err := client.SuperRootAtTimestamp(ctx, latestFinalizedLookupTimestamp)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to resolve finalized timestamp from SuperNode: %w", err)
		}
		resolvedTimestamp := initialResp.CurrentFinalizedTimestamp
		targetTimestamp = &resolvedTimestamp
		m.log.Info("Resolved latest finalized timestamp from SuperNode",
			"timestamp", resolvedTimestamp,
			"safeTimestamp", initialResp.CurrentSafeTimestamp,
			"currentL1", initialResp.CurrentL1)
	} else {
		m.log.Info("Using user-provided timestamp", "timestamp", *targetTimestamp)
	}

	resp, err := client.SuperRootAtTimestamp(ctx, *targetTimestamp)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch super root at timestamp %d: %w", *targetTimestamp, err)
	}
	if resp.Data == nil {
		return common.Hash{}, fmt.Errorf("no verified super root data at timestamp %d (safe=%d finalized=%d)", *targetTimestamp, resp.CurrentSafeTimestamp, resp.CurrentFinalizedTimestamp)
	}

	superRoot := common.Hash(resp.Data.SuperRoot)
	m.log.Info("Super root fetched successfully",
		"superRoot", superRoot.Hex(),
		"timestamp", *targetTimestamp,
		"chains", len(resp.ChainIDs),
		"currentL1", resp.CurrentL1)
	return superRoot, nil
}
