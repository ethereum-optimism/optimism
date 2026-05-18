package dsl

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/contracts/bindings/inbox"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
	GasLimit   uint64
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
		txOpts.GasLimit = opts.GasLimit
		contract, err := inbox.NewInbox(predeploys.CrossL2InboxAddr, chain.SequencerEngine.EthClient())
		require.NoError(i.t, err)
		id := stypes.Identifier{
			Origin:      ident.Origin,
			BlockNumber: bigs.Uint64Strict(ident.BlockNumber),
			LogIndex:    uint32(bigs.Uint64Strict(ident.LogIndex)),
			Timestamp:   bigs.Uint64Strict(ident.Timestamp),
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
