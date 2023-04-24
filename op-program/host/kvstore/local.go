package kvstore

import (
	"encoding/binary"
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	"github.com/ethereum/go-ethereum/common"
)

type LocalPreimageSource struct {
	config *config.Config
}

func NewLocalPreimageSource(config *config.Config) *LocalPreimageSource {
	return &LocalPreimageSource{config}
}

func localKey(num int64) common.Hash {
	return preimage.LocalIndexKey(num).PreimageKey()
}

var (
	L1HeadKey             = localKey(1)
	L2HeadKey             = localKey(2)
	L2ClaimKey            = localKey(3)
	L2ClaimBlockNumberKey = localKey(4)
	L2ChainConfigKey      = localKey(5)
	RollupKey             = localKey(6)
)

func (s *LocalPreimageSource) Get(key common.Hash) ([]byte, error) {
	switch key {
	case L1HeadKey:
		return s.config.L1Head.Bytes(), nil
	case L2HeadKey:
		return s.config.L2Head.Bytes(), nil
	case L2ClaimKey:
		return s.config.L2Claim.Bytes(), nil
	case L2ClaimBlockNumberKey:
		return binary.BigEndian.AppendUint64(nil, s.config.L2ClaimBlockNumber), nil
	case L2ChainConfigKey:
		return json.Marshal(s.config.L2ChainConfig)
	case RollupKey:
		return json.Marshal(s.config.Rollup)
	default:
		return nil, ErrNotFound
	}
}
