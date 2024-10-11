package kvstore

import (
	"encoding/binary"
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-program/client"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum/go-ethereum/common"
)

type LocalPreimageSource struct {
	config *config.Config
}

func NewLocalPreimageSource(config *config.Config) *LocalPreimageSource {
	return &LocalPreimageSource{config}
}

var (
	l1HeadKey             = client.L1HeadLocalIndex.PreimageKey()
	l2OutputRootKey       = client.L2OutputRootLocalIndex.PreimageKey()
	l2ClaimKey            = client.L2ClaimLocalIndex.PreimageKey()
	l2ClaimBlockNumberKey = client.L2ClaimBlockNumberLocalIndex.PreimageKey()
	l2ChainIDKey          = client.L2ChainIDLocalIndex.PreimageKey()
	l2ChainConfigKey      = client.L2ChainConfigLocalIndex.PreimageKey()
	rollupKey             = client.RollupConfigLocalIndex.PreimageKey()
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
		// The CustomChainIDIndicator informs the client to rely on the L2ChainConfigKey to
		// read the chain config. Otherwise, it'll attempt to read a non-existent hardcoded chain config
		var chainID uint64
		if s.config.IsCustomChainConfig {
			chainID = client.CustomChainIDIndicator
		} else {
			chainID = s.config.L2ChainConfig.ChainID.Uint64()
		}
		return binary.BigEndian.AppendUint64(nil, chainID), nil
	case l2ChainConfigKey:
		if !s.config.IsCustomChainConfig {
			return nil, ErrNotFound
		}
		return json.Marshal(s.config.L2ChainConfig)
	case rollupKey:
		if !s.config.IsCustomChainConfig {
			return nil, ErrNotFound
		}
		return json.Marshal(s.config.Rollup)
	default:
		return nil, ErrNotFound
	}
}
