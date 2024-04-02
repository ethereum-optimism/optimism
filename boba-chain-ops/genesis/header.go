package genesis

import (
	"math/big"

	"github.com/bobanetwork/boba/boba-chain-ops/chain"
	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
)

func CreateHeader(g *types.Genesis, parentHeader *types.Header, config *DeployConfig) (*types.Header, error) {
	head := &types.Header{
		Number:        big.NewInt(int64(config.L2OutputOracleStartingBlockNumber)),
		Nonce:         types.EncodeNonce(g.Nonce),
		Time:          g.Timestamp,
		ParentHash:    g.ParentHash,
		Extra:         g.ExtraData,
		GasLimit:      g.GasLimit,
		GasUsed:       g.GasUsed,
		Difficulty:    g.Difficulty,
		MixDigest:     g.Mixhash,
		Coinbase:      g.Coinbase,
		BaseFee:       g.BaseFee,
		ExcessBlobGas: g.ExcessBlobGas,
		AuRaStep:      g.AuRaStep,
		AuRaSeal:      g.AuRaSeal,
	}

	// update header for legacy genesis
	if !chain.IsBobaValidChainId(g.Config.ChainID) {
		return nil, chain.ErrInvalidChainID
	}

	head.Extra = []byte{}
	head.Time = uint64(config.L2OutputOracleStartingTimestamp)
	head.Difficulty = big.NewInt(0)
	head.BaseFee = libcommon.Big0
	head.ParentHash = parentHeader.Hash()

	return head, nil
}
