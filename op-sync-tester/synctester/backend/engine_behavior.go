package backend

import (
	"context"
	"fmt"

	rollupengine "github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

// EngineBehavior defines the interface for different EL implementation behaviors
type EngineBehavior interface {
	// GetEngineKind returns the engine kind this behavior represents
	GetEngineKind() rollupengine.Kind

	// SupportsPostFinalizationELSync returns whether this engine supports post-finalization EL sync
	SupportsPostFinalizationELSync() bool

	// HandleForkchoiceUpdate processes forkchoice updates with engine-specific behavior
	HandleForkchoiceUpdate(ctx context.Context, session *eth.SyncTesterSession, logger log.Logger,
		state *eth.ForkchoiceState, attr *eth.PayloadAttributes, payloadVersion engine.PayloadVersion,
		isCanyon, isEcotone bool) (*eth.ForkchoiceUpdatedResult, error)

	// HandleNewPayload processes new payloads with engine-specific behavior
	HandleNewPayload(ctx context.Context, session *eth.SyncTesterSession, logger log.Logger,
		payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash,
		executionRequests []hexutil.Bytes, isEcotone, isIsthmus bool) (*eth.PayloadStatusV1, error)

	// GetSyncMode returns the sync mode this engine uses
	GetSyncMode() string

	// GetNetworkCharacteristics returns network-specific characteristics
	GetNetworkCharacteristics() NetworkCharacteristics
}

// NetworkCharacteristics defines network-specific behavior
type NetworkCharacteristics struct {
	// SupportsRegenesis indicates if this network supports regenesis
	SupportsRegenesis bool

	// DefaultSyncMode is the default sync mode for this network
	DefaultSyncMode string

	// BlockTime is the expected block time for this network
	BlockTime uint64
}

// GethBehavior implements Geth-specific engine behavior
type GethBehavior struct {
	syncMode string
	network  NetworkCharacteristics
}

func NewGethBehavior(syncMode string, networkType string) *GethBehavior {
	network := getNetworkCharacteristics(networkType)
	if syncMode == "" {
		syncMode = network.DefaultSyncMode
	}
	return &GethBehavior{
		syncMode: syncMode,
		network:  network,
	}
}

func (g *GethBehavior) GetEngineKind() rollupengine.Kind {
	return rollupengine.Geth
}

func (g *GethBehavior) SupportsPostFinalizationELSync() bool {
	return false // Geth does not support post-finalization EL sync
}

func (g *GethBehavior) HandleForkchoiceUpdate(ctx context.Context, session *eth.SyncTesterSession, logger log.Logger,
	state *eth.ForkchoiceState, attr *eth.PayloadAttributes, payloadVersion engine.PayloadVersion,
	isCanyon, isEcotone bool) (*eth.ForkchoiceUpdatedResult, error) {

	logger.Debug("Geth behavior: handling forkchoice update", "engine", "geth", "syncMode", g.syncMode)

	// Geth-specific behavior: more conservative sync approach
	// Geth typically has stricter validation and slower sync
	return nil, nil // Delegate to default implementation
}

func (g *GethBehavior) HandleNewPayload(ctx context.Context, session *eth.SyncTesterSession, logger log.Logger,
	payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash,
	executionRequests []hexutil.Bytes, isEcotone, isIsthmus bool) (*eth.PayloadStatusV1, error) {

	logger.Debug("Geth behavior: handling new payload", "engine", "geth", "syncMode", g.syncMode)

	// Geth-specific behavior: more conservative payload validation
	return nil, nil // Delegate to default implementation
}

func (g *GethBehavior) GetSyncMode() string {
	return g.syncMode
}

func (g *GethBehavior) GetNetworkCharacteristics() NetworkCharacteristics {
	return g.network
}

// RethBehavior implements Reth-specific engine behavior
type RethBehavior struct {
	syncMode string
	network  NetworkCharacteristics
}

func NewRethBehavior(syncMode string, networkType string) *RethBehavior {
	network := getNetworkCharacteristics(networkType)
	if syncMode == "" {
		syncMode = network.DefaultSyncMode
	}
	return &RethBehavior{
		syncMode: syncMode,
		network:  network,
	}
}

func (r *RethBehavior) GetEngineKind() rollupengine.Kind {
	return rollupengine.Reth
}

func (r *RethBehavior) SupportsPostFinalizationELSync() bool {
	return true // Reth supports post-finalization EL sync
}

func (r *RethBehavior) HandleForkchoiceUpdate(ctx context.Context, session *eth.SyncTesterSession, logger log.Logger,
	state *eth.ForkchoiceState, attr *eth.PayloadAttributes, payloadVersion engine.PayloadVersion,
	isCanyon, isEcotone bool) (*eth.ForkchoiceUpdatedResult, error) {

	logger.Debug("Reth behavior: handling forkchoice update", "engine", "reth", "syncMode", r.syncMode)

	// Reth-specific behavior: more aggressive sync, better performance
	// Reth can handle post-finalization sync scenarios
	return nil, nil // Delegate to default implementation
}

func (r *RethBehavior) HandleNewPayload(ctx context.Context, session *eth.SyncTesterSession, logger log.Logger,
	payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash,
	executionRequests []hexutil.Bytes, isEcotone, isIsthmus bool) (*eth.PayloadStatusV1, error) {

	logger.Debug("Reth behavior: handling new payload", "engine", "reth", "syncMode", r.syncMode)

	// Reth-specific behavior: faster payload processing
	return nil, nil // Delegate to default implementation
}

func (r *RethBehavior) GetSyncMode() string {
	return r.syncMode
}

func (r *RethBehavior) GetNetworkCharacteristics() NetworkCharacteristics {
	return r.network
}

// ErigonBehavior implements Erigon-specific engine behavior
type ErigonBehavior struct {
	syncMode string
	network  NetworkCharacteristics
}

func NewErigonBehavior(syncMode string, networkType string) *ErigonBehavior {
	network := getNetworkCharacteristics(networkType)
	if syncMode == "" {
		syncMode = network.DefaultSyncMode
	}
	return &ErigonBehavior{
		syncMode: syncMode,
		network:  network,
	}
}

func (e *ErigonBehavior) GetEngineKind() rollupengine.Kind {
	return rollupengine.Erigon
}

func (e *ErigonBehavior) SupportsPostFinalizationELSync() bool {
	return true // Erigon supports post-finalization EL sync
}

func (e *ErigonBehavior) HandleForkchoiceUpdate(ctx context.Context, session *eth.SyncTesterSession, logger log.Logger,
	state *eth.ForkchoiceState, attr *eth.PayloadAttributes, payloadVersion engine.PayloadVersion,
	isCanyon, isEcotone bool) (*eth.ForkchoiceUpdatedResult, error) {

	logger.Debug("Erigon behavior: handling forkchoice update", "engine", "erigon", "syncMode", e.syncMode)

	// Erigon-specific behavior: optimized for archival data, different sync patterns
	return nil, nil // Delegate to default implementation
}

func (e *ErigonBehavior) HandleNewPayload(ctx context.Context, session *eth.SyncTesterSession, logger log.Logger,
	payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash,
	executionRequests []hexutil.Bytes, isEcotone, isIsthmus bool) (*eth.PayloadStatusV1, error) {

	logger.Debug("Erigon behavior: handling new payload", "engine", "erigon", "syncMode", e.syncMode)

	// Erigon-specific behavior: optimized for large datasets
	return nil, nil // Delegate to default implementation
}

func (e *ErigonBehavior) GetSyncMode() string {
	return e.syncMode
}

func (e *ErigonBehavior) GetNetworkCharacteristics() NetworkCharacteristics {
	return e.network
}

// CreateEngineBehavior creates an engine behavior based on the engine kind
func CreateEngineBehavior(engineKind rollupengine.Kind, syncMode, networkType string) (EngineBehavior, error) {
	switch engineKind {
	case rollupengine.Geth:
		return NewGethBehavior(syncMode, networkType), nil
	case rollupengine.Reth:
		return NewRethBehavior(syncMode, networkType), nil
	case rollupengine.Erigon:
		return NewErigonBehavior(syncMode, networkType), nil
	default:
		return nil, fmt.Errorf("unsupported engine kind: %s", engineKind)
	}
}

// getNetworkCharacteristics returns network-specific characteristics
func getNetworkCharacteristics(networkType string) NetworkCharacteristics {
	switch networkType {
	case "mainnet":
		return NetworkCharacteristics{
			SupportsRegenesis: true,
			DefaultSyncMode:   "full",
			BlockTime:         2, // 2 seconds for Optimism mainnet
		}
	case "sepolia":
		return NetworkCharacteristics{
			SupportsRegenesis: false,
			DefaultSyncMode:   "snap",
			BlockTime:         2,
		}
	case "goerli":
		return NetworkCharacteristics{
			SupportsRegenesis: true,
			DefaultSyncMode:   "snap",
			BlockTime:         2,
		}
	default:
		// Default characteristics for unknown networks
		return NetworkCharacteristics{
			SupportsRegenesis: false,
			DefaultSyncMode:   "full",
			BlockTime:         2,
		}
	}
}
