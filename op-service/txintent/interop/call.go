package txintent

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"

	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var _ txintent.Call = (*InitTrigger)(nil)
var _ txintent.Call = (*ExecTrigger)(nil)
var _ txintent.Call = (*RelayTrigger)(nil)

type InitTrigger struct {
	Emitter    common.Address // address of the EventLogger contract
	Topics     []common.Hash
	OpaqueData []byte
}

func (v *InitTrigger) To() (*common.Address, error) {
	return &v.Emitter, nil
}

func (v *InitTrigger) Data() ([]byte, error) {
	// TODO format call
	// This OpaqueData was for using the EventLogger contract to test
	// This is a temp bypass for setting calldata for basic testing
	return v.OpaqueData, nil
}

func (v *InitTrigger) AccessList() (types.AccessList, error) {
	return nil, nil
}

type ExecTrigger struct {
	Executor common.Address // address of the EventLogger contract
	Msg      suptypes.Message
}

func (v *ExecTrigger) To() (*common.Address, error) {
	return &v.Executor, nil
}

func (v *ExecTrigger) Data() ([]byte, error) {
	// TODO: Need to do better construct call input than this
	validateMessage := w3.MustNewFunc("validateMessage((address Origin, uint256 BlockNumber, uint256 LogIndex, uint256 Timestamp, uint256 ChainId), bytes32)", "")
	type Identifier struct {
		Origin      common.Address
		BlockNumber *big.Int
		LogIndex    *big.Int
		Timestamp   *big.Int
		ChainId     *big.Int
	}
	identifier := &Identifier{
		v.Msg.Identifier.Origin,
		big.NewInt(int64(v.Msg.Identifier.BlockNumber)),
		big.NewInt(int64(v.Msg.Identifier.LogIndex)),
		big.NewInt(int64(v.Msg.Identifier.Timestamp)),
		v.Msg.Identifier.ChainID.ToBig(),
	}
	validateMessageCalldata, err := validateMessage.EncodeArgs(
		identifier,
		v.Msg.PayloadHash,
	)
	if err != nil {
		return nil, err
	}
	return validateMessageCalldata, nil
}

func (v *ExecTrigger) AccessList() (types.AccessList, error) {
	access := v.Msg.Access()
	accessList := types.AccessList{{
		Address:     constants.CrossL2Inbox,
		StorageKeys: suptypes.EncodeAccessList([]suptypes.Access{access}),
	}}
	return accessList, nil
}

type RelayTrigger struct {
	ExecTrigger
	Payload []byte
}

func (v *RelayTrigger) Data() ([]byte, error) {
	// TODO: Need to do better construct call input than this
	relayMessage := w3.MustNewFunc("relayMessage((address Origin, uint256 BlockNumber, uint256 LogIndex, uint256 Timestamp, uint256 ChainId), bytes sentMessage)", "bytes returnData")
	type Identifier struct {
		Origin      common.Address
		BlockNumber *big.Int
		LogIndex    *big.Int
		Timestamp   *big.Int
		ChainId     *big.Int
	}
	identifier := &Identifier{
		v.Msg.Identifier.Origin,
		big.NewInt(int64(v.Msg.Identifier.BlockNumber)),
		big.NewInt(int64(v.Msg.Identifier.LogIndex)),
		big.NewInt(int64(v.Msg.Identifier.Timestamp)),
		v.Msg.Identifier.ChainID.ToBig(),
	}
	relayMessageCalldata, err := relayMessage.EncodeArgs(
		identifier,
		v.Payload,
	)
	if err != nil {
		return nil, err
	}
	return relayMessageCalldata, nil
}
