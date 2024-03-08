package client

import (
	"encoding/binary"
	"encoding/json"
	"math"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-program/chainconfig"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// LocalIndexKey constants define keys for local storage.
const (
	L1HeadLocalIndex preimage.LocalIndexKey = iota + 1
	L2OutputRootLocalIndex
	L2ClaimLocalIndex
	L2ClaimBlockNumberLocalIndex
	L2ChainIDLocalIndex
	// Local keys for custom chains.
	L2ChainConfigLocalIndex
	RollupConfigLocalIndex
)

// CustomChainIDIndicator is a sentinel value indicating the presence of custom chain config.
const CustomChainIDIndicator = uint64(math.MaxUint64)

// BootInfo holds bootstrap information for the client.
type BootInfo struct {
	L1Head             common.Hash
	L2OutputRoot       common.Hash
	L2Claim            common.Hash
	L2ClaimBlockNumber uint64
	L2ChainID          uint64

	// Configuration for L2 chain and rollup.
	L2ChainConfig *params.ChainConfig
	RollupConfig *rollup.Config
}

// oracleClient defines the interface for fetching data from an oracle.
type oracleClient interface {
	Get(key preimage.Key) []byte
}

// BootstrapClient is responsible for bootstrapping the client with necessary information.
type BootstrapClient struct {
	r oracleClient
}

// NewBootstrapClient initializes a new BootstrapClient with the given oracleClient.
func NewBootstrapClient(r oracleClient) *BootstrapClient {
	return &BootstrapClient{r: r}
}

// BootInfo retrieves bootstrap information from the oracle.
func (br *BootstrapClient) BootInfo() *BootInfo {
	// Fetching L1 head, L2 output root, L2 claim, and L2 claim block number.
	l1Head := common.BytesToHash(br.r.Get(L1HeadLocalIndex))
	l2OutputRoot := common.BytesToHash(br.r.Get(L2OutputRootLocalIndex))
	l2Claim := common.BytesToHash(br.r.Get(L2ClaimLocalIndex))
	l2ClaimBlockNumber := binary.BigEndian.Uint64(br.r.Get(L2ClaimBlockNumberLocalIndex))
	l2ChainID := binary.BigEndian.Uint64(br.r.Get(L2ChainIDLocalIndex))

	// Handling custom chain configurations.
	var l2ChainConfig *params.ChainConfig
	var rollupConfig *rollup.Config
	if l2ChainID == CustomChainIDIndicator {
		// Load custom chain and rollup configurations.
		l2ChainConfig = new(params.ChainConfig)
		if err := json.Unmarshal(br.r.Get(L2ChainConfigLocalIndex), l2ChainConfig); err != nil {
			panic("failed to bootstrap l2ChainConfig")
		}
		rollupConfig = new(rollup.Config)
		if err := json.Unmarshal(br.r.Get(RollupConfigLocalIndex), rollupConfig); err != nil {
			panic("failed to bootstrap rollup config")
		}
	} else {
		// Load default chain and rollup configurations.
		var err error
		if rollupConfig, err = chainconfig.RollupConfigByChainID(l2ChainID); err != nil {
			panic(err)
		}
		if l2ChainConfig, err = chainconfig.ChainConfigByChainID(l2ChainID); err != nil {
			panic(err)
		}
	}

	// Return boot info struct.
	return &BootInfo{
		L1Head:             l1Head,
		L2OutputRoot:       l2OutputRoot,
		L2Claim:            l2Claim,
		L2ClaimBlockNumber: l2ClaimBlockNumber,
		L2ChainID:          l2ChainID,
		L2ChainConfig:      l2ChainConfig,
		RollupConfig:       rollupConfig,
	}
}
