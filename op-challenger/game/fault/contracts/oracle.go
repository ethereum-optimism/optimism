package contracts

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type PreimageOracleAbi struct {
	abi *abi.ABI
}

func NewPreimageOracleAbi() (*PreimageOracleAbi, error) {
	oracleAbi, err := bindings.PreimageOracleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &PreimageOracleAbi{abi: oracleAbi}, nil
}

// GlobalOracleData takes the global preimage key and data
// and creates tx data to load the key, data pair into the
// PreimageOracle contract.
func (o *PreimageOracleAbi) GlobalOracleData(data *types.PreimageOracleData) ([]byte, error) {
	return o.abi.Pack(
		"loadKeccak256PreimagePart",
		big.NewInt(int64(data.OracleOffset)),
		data.GetPreimageWithoutSize(),
	)
}
