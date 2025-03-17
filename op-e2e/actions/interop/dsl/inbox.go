package dsl

import (
	"math/big"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/interop/contracts/bindings/inbox"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	stypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type InboxContract struct {
	t helpers.Testing

	Transactions []*GeneratedTransaction
}

func NewInboxContract(t helpers.Testing) *InboxContract {
	return &InboxContract{
		t: t,
	}
}

type ExecuteOpts struct {
	Identifier *inbox.Identifier
	Payload    *[]byte
}

func WithIdentifier(ident inbox.Identifier) func(opts *ExecuteOpts) {
	return func(opts *ExecuteOpts) {
		opts.Identifier = &ident
	}
}

func WithPayload(payload []byte) func(opts *ExecuteOpts) {
	return func(opts *ExecuteOpts) {
		opts.Payload = &payload
	}
}

func WithPendingMessage(emitter *EmitterContract, chain *Chain, number uint64, logIndex int, msg string) func(opts *ExecuteOpts) {
	return func(opts *ExecuteOpts) {
		blockTime := chain.RollupCfg.TimestampForBlock(number)
		id := inbox.Identifier{
			Origin:      emitter.Address(chain),
			BlockNumber: big.NewInt(int64(number)),
			LogIndex:    big.NewInt(int64(logIndex)),
			Timestamp:   big.NewInt(int64(blockTime)),
			ChainId:     chain.RollupCfg.L2ChainID,
		}
		opts.Identifier = &id

		topic := crypto.Keccak256Hash([]byte("DataEmitted(bytes)"))
		var payload []byte
		payload = append(payload, topic.Bytes()...)
		msgHash := crypto.Keccak256Hash([]byte(msg))
		payload = append(payload, msgHash.Bytes()...)
		opts.Payload = &payload
	}
}

func (i *InboxContract) Execute(user *DSLUser, initTx *GeneratedTransaction, args ...func(opts *ExecuteOpts)) TransactionCreator {
	opts := ExecuteOpts{}
	for _, arg := range args {
		arg(&opts)
	}
	return func(chain *Chain) *GeneratedTransaction {
		// Wait until we're actually creating this transaction to call initTx methods.
		// This allows the init tx to be in the same block as the exec tx as the actual initTx is only
		// created when it gets included in the block.
		var ident inbox.Identifier
		if opts.Identifier != nil {
			ident = *opts.Identifier
		} else {
			ident = initTx.Identifier()
		}
		var payload []byte
		if opts.Payload != nil {
			payload = *opts.Payload
		} else {
			payload = initTx.MessagePayload()
		}
		txOpts, from := user.TransactOpts(chain.ChainID.ToBig())
		contract, err := inbox.NewInbox(predeploys.CrossL2InboxAddr, chain.SequencerEngine.EthClient())
		require.NoError(i.t, err)
		id := stypes.Identifier{
			Origin:      ident.Origin,
			BlockNumber: ident.BlockNumber.Uint64(),
			LogIndex:    uint32(ident.LogIndex.Uint64()),
			Timestamp:   ident.Timestamp.Uint64(),
			ChainID:     eth.ChainIDFromBig(ident.ChainId),
		}
		msgHash := crypto.Keccak256Hash(payload)
		access := id.ChecksumArgs(msgHash).Access()
		inboxAccessList := stypes.EncodeAccessList([]stypes.Access{access})
		txOpts.AccessList = types.AccessList{types.AccessTuple{
			Address:     predeploys.CrossL2InboxAddr,
			StorageKeys: inboxAccessList,
		}}
		tx, err := contract.ValidateMessage(txOpts, ident, msgHash)
		require.NoError(i.t, err)
		genTx := NewGeneratedTransaction(i.t, chain, tx, from)
		i.Transactions = append(i.Transactions, genTx)
		return genTx
	}
}

func (i *InboxContract) LastTransaction() *GeneratedTransaction {
	require.NotZero(i.t, i.Transactions, "no transactions created")
	return i.Transactions[len(i.Transactions)-1]
}
