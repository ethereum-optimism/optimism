package boot

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-program/chainconfig"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// CustomChainIDIndicator is used to detect when the program should load custom chain configuration
var CustomChainIDIndicator = eth.ChainIDFromUInt64(uint64(math.MaxUint64))

type BootInfo struct {
	L1Head             common.Hash
	L2OutputRoot       common.Hash
	L2Claim            common.Hash
	L2ClaimBlockNumber uint64
	L2ChainID          eth.ChainID

	L2ChainConfig *params.ChainConfig
	RollupConfig  *rollup.Config
	L1ChainConfig *params.ChainConfig
}

type BootstrapClient struct {
	r oracleClient
}

func NewBootstrapClient(r oracleClient) *BootstrapClient {
	return &BootstrapClient{r: r}
}

func (br *BootstrapClient) BootInfo() *BootInfo {
	l1Head := common.BytesToHash(br.r.Get(L1HeadLocalIndex))
	l2OutputRoot := common.BytesToHash(br.r.Get(L2OutputRootLocalIndex))
	l2Claim := common.BytesToHash(br.r.Get(L2ClaimLocalIndex))
	l2ClaimBlockNumber := binary.BigEndian.Uint64(br.r.Get(L2ClaimBlockNumberLocalIndex))
	l2ChainID := eth.ChainIDFromUInt64(binary.BigEndian.Uint64(br.r.Get(L2ChainIDLocalIndex)))

	var l1ChainConfig *params.ChainConfig
	var l2ChainConfig *params.ChainConfig
	var rollupConfig *rollup.Config
	if l2ChainID == CustomChainIDIndicator {
		l2ChainConfig = new(params.ChainConfig)
		err := json.Unmarshal(br.r.Get(L2ChainConfigLocalIndex), &l2ChainConfig)
		if err != nil {
			panic("failed to bootstrap l2ChainConfig")
		}
		rollupConfig = new(rollup.Config)
		err = json.Unmarshal(br.r.Get(RollupConfigLocalIndex), rollupConfig)
		if err != nil {
			panic("failed to bootstrap rollup config")
		}
		l1ChainConfig = new(params.ChainConfig)
		err = json.Unmarshal(br.r.Get(L1ChainConfigLocalIndex), l1ChainConfig)
		if err != nil {
			panic("failed to bootstrap l1ChainConfig: " + fmt.Sprintf("%v", err))
		}
		if l1ChainConfig.ChainID.Cmp(rollupConfig.L1ChainID) != 0 {
			panic(fmt.Sprintf("l1ChainConfig chain ID does not match rollup config L1 chain ID: %v != %v",
				l1ChainConfig.ChainID, rollupConfig.L1ChainID))
		}
	} else {
		var err error
		rollupConfig, err = chainconfig.RollupConfigByChainID(l2ChainID)
		if err != nil {
			panic(err)
		}
		l1ChainID := eth.ChainIDFromBig(rollupConfig.L1ChainID)
		l1ChainConfig, err = chainconfig.L1ChainConfigByChainID(l1ChainID)
		if err != nil {
			panic(err)
		}
		l2ChainConfig, err = chainconfig.L2ChainConfigByChainID(l2ChainID)
		if err != nil {
			panic(err)
		}
	}

	return &BootInfo{
		L1Head:             l1Head,
		L2OutputRoot:       l2OutputRoot,
		L2Claim:            l2Claim,
		L2ClaimBlockNumber: l2ClaimBlockNumber,
		L2ChainID:          l2ChainID,
		L2ChainConfig:      l2ChainConfig,
		RollupConfig:       rollupConfig,
		L1ChainConfig:      l1ChainConfig,
	}
}
