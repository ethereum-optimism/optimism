package derive

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	OutboxMessagePassedABI     = "MessagePassed(uint256,address,address,bytes32,uint256,uint256,bytes,bytes32)"
	OutboxMessagePassedABIHash = crypto.Keccak256Hash([]byte(OutboxMessagePassedABI))

	CrossL2InboxAddr          = predeploys.CrossL2InboxAddr
	CrossL2InboxFuncSignature = "deliverMessages((bytes32,bytes32,uint256,bytes32[])[])"
	CrossL2InboxFuncBytes4    = crypto.Keccak256([]byte(CrossL2InboxFuncSignature))[:4]

	CrossL2InboxDepositerAddress = common.HexToAddress("0xdeaddeaddeaddeaddeaddeaddeaddeaddead0002")
)

type InteropMessage struct {
	Nonce *big.Int

	SourceChain common.Hash
	TargetChain common.Hash

	From common.Address
	To   common.Address

	Value    *big.Int
	GasLimit *big.Int

	Data hexutil.Bytes

	MessageRoot common.Hash
}

type InteropMessageSourceInfo struct {
	RemoteChain common.Hash

	FromBlockNumber *big.Int
	ToBlockNumber   *big.Int
}

type InteropMessages struct {
	SourceInfo InteropMessageSourceInfo
	Messages   []InteropMessage
}

func UnmarshalInteropMessageLog(remoteChain common.Hash, ev *types.Log) (*InteropMessage, error) {
	if len(ev.Topics) != 4 {
		return nil, fmt.Errorf("expected 4 event topics for outbox message event. got %d", len(ev.Topics))
	}
	if ev.Topics[0] != OutboxMessagePassedABIHash {
		return nil, fmt.Errorf("invalid message event selector: %s, expected: %s", ev.Topics[0], OutboxMessagePassedABIHash)
	}

	// indexed fields
	nonce := new(big.Int).SetBytes(ev.Topics[1][:])
	from := common.BytesToAddress(ev.Topics[2][12:])
	to := common.BytesToAddress(ev.Topics[3][12:])

	// TODO: dont rely on bindings as & manually marshal/unmarshal structs
	var outboxMessagePassed bindings.CrossL2OutboxMessagePassed
	abi, err := bindings.CrossL2OutboxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	if err := abi.UnpackIntoInterface(&outboxMessagePassed, "MessagePassed", ev.Data); err != nil {
		return nil, err
	}

	msg := InteropMessage{
		Nonce: nonce,
		From:  from,
		To:    to,

		SourceChain: remoteChain,
		TargetChain: outboxMessagePassed.TargetChain,
		Value:       outboxMessagePassed.Value,
		GasLimit:    outboxMessagePassed.GasLimit,
		Data:        outboxMessagePassed.Data,
		MessageRoot: outboxMessagePassed.MessageRoot,
	}

	return &msg, nil
}

// InteropMessagesDeposit will marshal the supplied batch of incoming interop messages
// into a system deposit transaction. When constructng this batch, the outbox of each
// remote chain should have been filtered to only include messages targeted for the local
// chain.
func InteropMessagesDeposit(interopMsgs []InteropMessages) (*types.DepositTx, error) {
	// TODO: dont rely on bindings & manually marshal/unmarshal structs
	abi, err := bindings.CrossL2InboxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	// TODO: derive from the remote source info. Since the calldata includes the
	// latest remote block number, the transaction hash will be unique anyways
	sourceHash := common.Hash{}

	type inboxMessage struct {
		Chain        common.Hash
		Output       common.Hash
		BlockNumber  *big.Int
		MessageRoots []common.Hash
	}

	inboxMessages := make([]inboxMessage, len(interopMsgs))

	mint := big.NewInt(0)
	for i, chainMsgs := range interopMsgs {
		msgRoots := make([]common.Hash, len(chainMsgs.Messages))
		for j, msg := range chainMsgs.Messages {
			msgRoots[j] = msg.MessageRoot // verify message root
			mint = mint.Add(mint, msg.Value)
		}

		inboxMessages[i] = inboxMessage{
			Chain:        chainMsgs.SourceInfo.RemoteChain,
			BlockNumber:  chainMsgs.SourceInfo.ToBlockNumber,
			MessageRoots: msgRoots,
			// TODO: output information
		}
	}

	// encode function call
	deliverMsgsFn := abi.Methods["deliverMessages"]
	data, err := deliverMsgsFn.Inputs.Pack(inboxMessages)
	if err != nil {
		return nil, err
	}

	return &types.DepositTx{
		SourceHash:          sourceHash,
		From:                CrossL2InboxDepositerAddress,
		To:                  &CrossL2InboxAddr,
		Mint:                mint,
		Value:               mint,
		Gas:                 RegolithSystemTxGas,
		IsSystemTransaction: false,
		Data:                data,
	}, nil
}
