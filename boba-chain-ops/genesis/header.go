package genesis

import (
	"math/big"

	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/boba-chain-ops/chain"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
)

func CreateHeader(g *types.Genesis) (*types.Header, error) {
	head := &types.Header{
		Number:        new(big.Int).SetUint64(g.Number),
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
		ExcessDataGas: g.ExcessDataGas,
		AuRaStep:      g.AuRaStep,
		AuRaSeal:      g.AuRaSeal,
	}

	if g.GasLimit == 0 {
		head.GasLimit = params.GenesisGasLimit
	}
	if g.Difficulty == nil {
		head.Difficulty = params.GenesisDifficulty
	}
	if g.Config != nil && (g.Config.IsLondon(0)) {
		if g.BaseFee != nil {
			head.BaseFee = g.BaseFee
		} else {
			head.BaseFee = new(big.Int).SetUint64(params.InitialBaseFee)
		}
	}

	// update header for legacy genesis
	if !chain.IsBobaValidChainId(g.Config.ChainID) {
		return nil, chain.ErrInvalidChainID
	}

	head.Time = 0
	head.Difficulty = big.NewInt(1)
	head.Extra = common.Hex2Bytes(chain.GetBobaGenesisExtraData(g.Config.ChainID))
	head.Coinbase = libcommon.HexToAddress(chain.GetBobaGenesisCoinbase(g.Config.ChainID))
	head.Root = libcommon.HexToHash(chain.GetBobaGenesisRoot(g.Config.ChainID))

	return head, nil
}
