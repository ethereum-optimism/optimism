package contracts

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

const (
	methodLoadKeccak256PreimagePart = "loadKeccak256PreimagePart"
)

// PreimageOracleContract is a binding that works with contracts implementing the IPreimageOracle interface
type PreimageOracleContract struct {
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

func NewPreimageOracleContract(addr common.Address, caller *batching.MultiCaller) (*PreimageOracleContract, error) {
	mipsAbi, err := bindings.PreimageOracleMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to load preimage oracle ABI: %w", err)
	}

	return &PreimageOracleContract{
		multiCaller: caller,
		contract:    batching.NewBoundContract(mipsAbi, addr),
	}, nil
}

func (c PreimageOracleContract) AddGlobalDataTx(data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	call := c.contract.Call(methodLoadKeccak256PreimagePart, new(big.Int).SetUint64(uint64(data.OracleOffset)), data.GetPreimageWithoutSize())
	return call.ToTxCandidate()
}
