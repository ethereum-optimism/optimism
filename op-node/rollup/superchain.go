package rollup

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/superchain-registry/superchain"
)

var OPStackSupport = params.ProtocolVersionV0{Build: [8]byte{}, Major: 6, Minor: 0, Patch: 0, PreRelease: 0}.Encode()

const (
	pgnSepolia = 58008
)

// LoadOPStackRollupConfig loads the rollup configuration of the requested chain ID from the superchain-registry.
// Some chains may require a SystemConfigProvider to retrieve any values not part of the registry.
func LoadOPStackRollupConfig(chainID uint64) (*Config, error) {
	chConfig, ok := superchain.OPChains[chainID]
	if !ok {
		return nil, fmt.Errorf("unknown chain ID: %d", chainID)
	}

	superChain, ok := superchain.Superchains[chConfig.Superchain]
	if !ok {
		return nil, fmt.Errorf("chain %d specifies unknown superchain: %q", chainID, chConfig.Superchain)
	}

	var genesisSysConfig eth.SystemConfig
	if sysCfg, ok := superchain.GenesisSystemConfigs[chainID]; ok {
		genesisSysConfig = eth.SystemConfig{
			BatcherAddr: common.Address(sysCfg.BatcherAddr),
			Overhead:    eth.Bytes32(sysCfg.Overhead),
			Scalar:      eth.Bytes32(sysCfg.Scalar),
			GasLimit:    sysCfg.GasLimit,
		}
	} else {
		return nil, fmt.Errorf("unable to retrieve genesis SystemConfig of chain %d", chainID)
	}

	addrs, ok := superchain.Addresses[chainID]
	if !ok {
		return nil, fmt.Errorf("unable to retrieve deposit contract address")
	}

	var plasma *PlasmaConfig
	if chConfig.Plasma != nil {
		plasma = &PlasmaConfig{
			DAChallengeAddress: common.Address(*chConfig.Plasma.DAChallengeAddress),
			DAChallengeWindow:  *chConfig.Plasma.DAChallengeWindow,
			DAResolveWindow:    *chConfig.Plasma.DAResolveWindow,
		}
	}

	regolithTime := uint64(0)
	cfg := &Config{
		Genesis: Genesis{
			L1: eth.BlockID{
				Hash:   common.Hash(chConfig.Genesis.L1.Hash),
				Number: chConfig.Genesis.L1.Number,
			},
			L2: eth.BlockID{
				Hash:   common.Hash(chConfig.Genesis.L2.Hash),
				Number: chConfig.Genesis.L2.Number,
			},
			L2Time:       chConfig.Genesis.L2Time,
			SystemConfig: genesisSysConfig,
		},
		// The below chain parameters can be different per OP-Stack chain,
		// but since none of the superchain chains differ, it's not represented in the superchain-registry yet.
		// This restriction on superchain-chains may change in the future.
		// Test/Alt configurations can still load custom rollup-configs when necessary.
		BlockTime:              2,
		MaxSequencerDrift:      600,
		SeqWindowSize:          3600,
		ChannelTimeout:         300,
		L1ChainID:              new(big.Int).SetUint64(superChain.Config.L1.ChainID),
		L2ChainID:              new(big.Int).SetUint64(chConfig.ChainID),
		RegolithTime:           &regolithTime,
		CanyonTime:             chConfig.CanyonTime,
		DeltaTime:              chConfig.DeltaTime,
		EcotoneTime:            chConfig.EcotoneTime,
		FjordTime:              chConfig.FjordTime,
		BatchInboxAddress:      common.Address(chConfig.BatchInboxAddr),
		DepositContractAddress: common.Address(addrs.OptimismPortalProxy),
		L1SystemConfigAddress:  common.Address(addrs.SystemConfigProxy),
		PlasmaConfig:           plasma,
	}

	if superChain.Config.ProtocolVersionsAddr != nil { // Set optional protocol versions address
		cfg.ProtocolVersionsAddress = common.Address(*superChain.Config.ProtocolVersionsAddr)
	}
	if chainID == pgnSepolia {
		cfg.MaxSequencerDrift = 1000
		cfg.SeqWindowSize = 7200
	}
	return cfg, nil
}
