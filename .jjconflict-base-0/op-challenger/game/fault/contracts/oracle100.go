package contracts

import (
	_ "embed"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

//go:embed abis/PreimageOracle-1.0.0.json
var preimageOracleAbi100 []byte

type PreimageOracleContract100 struct {
	PreimageOracleContractLatest
}

func (c *PreimageOracleContract100) AddGlobalDataTx(data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	if len(data.OracleKey) == 0 || preimage.KeyType(data.OracleKey[0]) != preimage.PrecompileKeyType {
		return c.PreimageOracleContractLatest.AddGlobalDataTx(data)
	}
	inputs := data.GetPreimageWithoutSize()
	call := c.contract.Call(methodLoadPrecompilePreimagePart,
		new(big.Int).SetUint64(uint64(data.OracleOffset)),
		common.BytesToAddress(inputs[0:20]),
		inputs[20:])
	return call.ToTxCandidate()
}
