package txintent

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"

	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var _ Call = (*InitTrigger)(nil)
var _ Call = (*ExecTrigger)(nil)

var _ Call = (*SendTrigger)(nil)
var _ Call = (*RelayTrigger)(nil)

// Trigger for using the EventLogger to trigger emitLog
type InitTrigger struct {
	Emitter    common.Address // address of the EventLogger
	Topics     [][32]byte
	OpaqueData []byte
}

func (v *InitTrigger) To() (*common.Address, error) {
	return &v.Emitter, nil
}

func (v *InitTrigger) Data() ([]byte, error) {
	// TODO(15005): Need to do better construct call input than this
	emitLog := w3.MustNewFunc("emitLog(bytes32[] topics, bytes data)", "")
	calldata, err := emitLog.EncodeArgs(v.Topics, v.OpaqueData)
	if err != nil {
		return nil, fmt.Errorf("failed to construct calldata: %w", err)
	}
	return calldata, nil
}

func (v *InitTrigger) AccessList() (types.AccessList, error) {
	return nil, nil
}

// Trigger for using the L2ToL2CrossDomainMessenger to trigger sendMessage
type SendTrigger struct {
	Emitter         common.Address // address of the L2ToL2CrossDomainMessenger
	DestChainID     eth.ChainID
	Target          common.Address
	RelayedCalldata []byte
}

func (v *SendTrigger) To() (*common.Address, error) {
	return &v.Emitter, nil
}

func (v *SendTrigger) Data() ([]byte, error) {
	// TODO(15005): Need to do better construct call input than this
	sendMessage := w3.MustNewFunc("sendMessage(uint256, address, bytes calldata)", "bytes32")
	calldata, err := sendMessage.EncodeArgs(v.DestChainID.ToBig(), v.Target, v.RelayedCalldata)
	if err != nil {
		return nil, fmt.Errorf("failed to construct calldata: %w", err)
	}
	return calldata, nil
}

func (v *SendTrigger) AccessList() (types.AccessList, error) {
	return nil, nil
}

// Trigger for using the CrossL2Inbox to trigger validateMessage
// This Trigger may be embedded to other triggers for preparing access lists
type ExecTrigger struct {
	Executor common.Address // address of the EventLogger or CrossL2Inbox
	Msg      suptypes.Message
}

func (v *ExecTrigger) To() (*common.Address, error) {
	return &v.Executor, nil
}

func (v *ExecTrigger) Data() ([]byte, error) {
	// TODO(15005): Need to do better construct call input than this
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
	calldata, err := validateMessage.EncodeArgs(identifier, v.Msg.PayloadHash)
	if err != nil {
		return nil, fmt.Errorf("failed to construct calldata: %w", err)
	}
	return calldata, nil
}

func (v *ExecTrigger) AccessList() (types.AccessList, error) {
	access := v.Msg.Access()
	accessList := types.AccessList{{
		Address:     constants.CrossL2Inbox,
		StorageKeys: suptypes.EncodeAccessList([]suptypes.Access{access}),
	}}
	return accessList, nil
}

// Trigger for using the L2ToL2CrossDomainMessenger to trigger relayMessage
type RelayTrigger struct {
	ExecTrigger
	Payload []byte
}

func (v *RelayTrigger) Data() ([]byte, error) {
	// TODO(15005): Need to do better construct call input than this
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
	calldata, err := relayMessage.EncodeArgs(identifier, v.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to construct calldata: %w", err)
	}
	return calldata, nil
}
