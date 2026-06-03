package common

import (
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/kvstore"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-program/client/boot"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type LocalPreimageSource struct {
	config *config.Config
}

func NewLocalPreimageSource(config *config.Config) *LocalPreimageSource {
	return &LocalPreimageSource{config}
}

var (
	l1HeadKey             = preimage.L1HeadLocalIndex.PreimageKey()
	l2OutputRootKey       = preimage.L2OutputRootLocalIndex.PreimageKey()
	l2ClaimKey            = preimage.L2ClaimLocalIndex.PreimageKey()
	l2ClaimBlockNumberKey = preimage.L2ClaimBlockNumberLocalIndex.PreimageKey()
	l2ChainIDKey          = preimage.L2ChainIDLocalIndex.PreimageKey()
	l2ChainConfigKey      = preimage.L2ChainConfigLocalIndex.PreimageKey()
	rollupKey             = preimage.RollupConfigLocalIndex.PreimageKey()
	dependencySetKey      = preimage.DependencySetLocalIndex.PreimageKey()
	l1ChainConfigKey      = preimage.L1ChainConfigLocalIndex.PreimageKey()
)

func (s *LocalPreimageSource) Get(key common.Hash) ([]byte, error) {
	switch [32]byte(key) {
	case l1HeadKey:
		return s.config.L1Head.Bytes(), nil
	case l2OutputRootKey:
		return s.config.L2OutputRoot.Bytes(), nil
	case l2ClaimKey:
		return s.config.L2Claim.Bytes(), nil
	case l2ClaimBlockNumberKey:
		return binary.BigEndian.AppendUint64(nil, s.config.L2ClaimBlockNumber), nil
	case l2ChainIDKey:
		return binary.BigEndian.AppendUint64(nil, eth.EvilChainIDToUInt64(s.config.L2ChainID)), nil
	case l2ChainConfigKey:
		if s.config.L2ChainID != boot.CustomChainIDIndicator {
			return nil, kvstore.ErrNotFound
		}
		if s.config.InteropEnabled {
			return json.Marshal(s.config.L2ChainConfigs)
		}
		return json.Marshal(s.config.L2ChainConfigs[0])
	case rollupKey:
		if s.config.L2ChainID != boot.CustomChainIDIndicator {
			return nil, kvstore.ErrNotFound
		}
		if s.config.InteropEnabled {
			return json.Marshal(s.config.Rollups)
		}
		return json.Marshal(s.config.Rollups[0])
	case dependencySetKey:
		if !s.config.InteropEnabled {
			return nil, errors.New("host is not configured to serve dependencySet local keys")
		}
		return json.Marshal(s.config.DependencySet)
	case l1ChainConfigKey:
		// NOTE: We check the L2 chain ID again to determine if we are using custom configs
		if s.config.L2ChainID != boot.CustomChainIDIndicator {
			return nil, kvstore.ErrNotFound
		}
		return json.Marshal(s.config.L1ChainConfig)
	default:
		return nil, kvstore.ErrNotFound
	}
}
