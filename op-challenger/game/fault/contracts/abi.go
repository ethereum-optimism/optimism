package contracts

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type FaultDisputeGameAbi struct {
	fdgAbi *abi.ABI
}

func NewFaultDisputeGameAbi() (*FaultDisputeGameAbi, error) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGameAbi{fdgAbi: fdgAbi}, nil
}

func (f *FaultDisputeGameAbi) ResolveCallData() ([]byte, error) {
	return f.fdgAbi.Pack("resolve")
}

func (f *FaultDisputeGameAbi) ParseResolveResult(res []byte) (gameTypes.GameStatus, error) {
	var status uint8
	if err := f.fdgAbi.UnpackIntoInterface(&status, "resolve", res); err != nil {
		return gameTypes.GameStatusInProgress, err
	}
	return gameTypes.GameStatusFromUint8(status)
}

// ResolveClaimData creates the transaction data for the ResolveClaim function.
func (f *FaultDisputeGameAbi) ResolveClaimData(claimIdx uint64) ([]byte, error) {
	return f.fdgAbi.Pack("resolveClaim", big.NewInt(int64(claimIdx)))
}

// FaultDefendData creates the transaction data for the Defend function.
func (f *FaultDisputeGameAbi) FaultDefendData(parentContractIndex int, pivot common.Hash) ([]byte, error) {
	return f.fdgAbi.Pack(
		"defend",
		big.NewInt(int64(parentContractIndex)),
		pivot,
	)
}

// FaultAttackData creates the transaction data for the Attack function.
func (f *FaultDisputeGameAbi) FaultAttackData(parentContractIndex int, pivot common.Hash) ([]byte, error) {
	return f.fdgAbi.Pack(
		"attack",
		big.NewInt(int64(parentContractIndex)),
		pivot,
	)
}

// StepTxData creates the transaction data for the step function.
func (f *FaultDisputeGameAbi) StepTxData(claimIdx uint64, isAttack bool, stateData []byte, proof []byte) ([]byte, error) {
	return f.fdgAbi.Pack(
		"step",
		big.NewInt(int64(claimIdx)),
		isAttack,
		stateData,
		proof,
	)
}

// AddLocalData takes the local preimage key and data
// and creates tx data to load the key, data pair into the
// PreimageOracle contract from the FaultDisputeGame contract call.
func (f *FaultDisputeGameAbi) AddLocalData(data *types.PreimageOracleData) ([]byte, error) {
	return f.fdgAbi.Pack(
		"addLocalData",
		data.GetIdent(),
		big.NewInt(int64(data.OracleOffset)),
	)
}
