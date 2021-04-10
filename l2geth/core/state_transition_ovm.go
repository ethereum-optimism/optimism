package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rollup/dump"
)

var ZeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")

type ovmTransaction struct {
	Timestamp     *big.Int       `json:"timestamp"`
	BlockNumber   *big.Int       `json:"blockNumber"`
	L1QueueOrigin uint8          `json:"l1QueueOrigin"`
	L1TxOrigin    common.Address `json:"l1TxOrigin"`
	Entrypoint    common.Address `json:"entrypoint"`
	GasLimit      *big.Int       `json:"gasLimit"`
	Data          []uint8        `json:"data"`
}

func toExecutionManagerRun(evm *vm.EVM, msg Message) (Message, error) {
	tx := ovmTransaction{
		evm.Context.Time,
		msg.L1BlockNumber(),
		uint8(msg.QueueOrigin().Uint64()),
		*msg.L1MessageSender(),
		*msg.To(),
		big.NewInt(int64(msg.Gas())),
		msg.Data(),
	}

	var abi = evm.Context.OvmExecutionManager.ABI
	var args = []interface{}{
		tx,
		evm.Context.OvmStateManager.Address,
	}

	ret, err := abi.Pack("run", args...)
	if err != nil {
		return nil, err
	}

	outputmsg, err := modMessage(
		msg,
		msg.From(),
		&evm.Context.OvmExecutionManager.Address,
		ret,
		evm.Context.GasLimit,
	)
	if err != nil {
		return nil, err
	}

	return outputmsg, nil
}

func AsOvmMessage(tx *types.Transaction, signer types.Signer, decompressor common.Address, gasLimit uint64) (Message, error) {
	msg, err := tx.AsMessage(signer)
	if err != nil {
		// This should only be allowed to pass if the transaction is in the ctc
		// already. The presence of `Index` should specify this.
		index := tx.GetMeta().Index
		if index == nil {
			return msg, fmt.Errorf("Cannot convert tx to message in asOvmMessage: %w", err)
		}
	}

	// Queue origin L1ToL2 transactions do not go through the
	// sequencer entrypoint. The calldata is expected to be in the
	// correct format when deserialized from the EVM events, see
	// rollup/sync_service.go.
	qo := msg.QueueOrigin()
	if qo != nil && qo.Uint64() == uint64(types.QueueOriginL1ToL2) {
		return msg, nil
	}

	// Sequencer transactions get sent to the "sequencer entrypoint," a contract that decompresses
	// the incoming transaction data.
	outmsg, err := modMessage(
		msg,
		msg.From(),
		&decompressor,
		tx.GetMeta().RawTransaction,
		gasLimit,
	)

	if err != nil {
		return msg, fmt.Errorf("Cannot mod message: %w", err)
	}

	return outmsg, nil
}

func EncodeSimulatedMessage(msg Message, timestamp, blockNumber *big.Int, executionManager, stateManager dump.OvmDumpAccount) (Message, error) {
	to := msg.To()
	if to == nil {
		to = &common.Address{0}
	}

	tx := ovmTransaction{
		timestamp,
		blockNumber, // TODO (what's the correct block number?)
		uint8(msg.QueueOrigin().Uint64()),
		*msg.L1MessageSender(),
		*to,
		big.NewInt(int64(msg.Gas())),
		msg.Data(),
	}

	from := msg.From()
	var args = []interface{}{
		tx,
		from,
		stateManager.Address,
	}

	output, err := executionManager.ABI.Pack("simulateMessage", args...)
	if err != nil {
		return nil, fmt.Errorf("Cannot pack simulateMessage: %w", err)
	}

	return modMessage(
		msg,
		common.Address{},
		&executionManager.Address,
		output,
		msg.Gas(),
	)
}

func modMessage(
	msg Message,
	from common.Address,
	to *common.Address,
	data []byte,
	gasLimit uint64,
) (Message, error) {
	queueOrigin, err := getQueueOrigin(msg.QueueOrigin())
	if err != nil {
		return nil, err
	}

	outmsg := types.NewMessage(
		from,
		to,
		msg.Nonce(),
		msg.Value(),
		gasLimit,
		msg.GasPrice(),
		data,
		false,
		msg.L1MessageSender(),
		msg.L1BlockNumber(),
		queueOrigin,
		msg.SignatureHashType(),
	)

	return outmsg, nil
}

func getQueueOrigin(
	queueOrigin *big.Int,
) (types.QueueOrigin, error) {
	if queueOrigin.Cmp(big.NewInt(0)) == 0 {
		return types.QueueOriginSequencer, nil
	} else if queueOrigin.Cmp(big.NewInt(1)) == 0 {
		return types.QueueOriginL1ToL2, nil
	} else if queueOrigin.Cmp(big.NewInt(2)) == 0 {
		return types.QueueOriginL1ToL2, nil
	} else {
		return types.QueueOriginSequencer, fmt.Errorf("invalid queue origin: %d", queueOrigin)
	}
}
