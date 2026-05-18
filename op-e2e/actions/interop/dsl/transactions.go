package dsl

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/bindingspreview"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/contracts/bindings/inbox"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	stypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type TxIncluder interface {
	IncludeTx(transaction *types.Transaction, from common.Address) (*types.Receipt, error)
}
type GeneratedTransaction struct {
	t     helpers.Testing
	chain *Chain
	tx    *types.Transaction
	from  common.Address

	// rcpt is only available after inclusion
	rcpt *types.Receipt
}

func NewGeneratedTransaction(t helpers.Testing, chain *Chain, tx *types.Transaction, from common.Address) *GeneratedTransaction {
	return &GeneratedTransaction{
		t:     t,
		chain: chain,
		tx:    tx,
		from:  from,
	}
}

func (m *GeneratedTransaction) Include() {
	rcpt, err := m.chain.SequencerEngine.EngineApi.IncludeTx(m.tx, m.from)
	require.NoError(m.t, err)
	m.rcpt = rcpt
}

// IncludeDepositOK includes the GeneratedTransaction via a user deposit transaction.
func (m *GeneratedTransaction) IncludeDepositOK(l1User *DSLUser, depositTxOpts *bind.TransactOpts, l1Miner *helpers.L1Miner) {
	optimismPortal2, err := bindingspreview.NewOptimismPortal2(m.chain.RollupCfg.DepositContractAddress, l1Miner.EthClient())
	require.NoError(m.t, err)

	l1Opts, _ := l1User.TransactOpts(l1Miner.L1Chain().Config().ChainID)
	l1Opts.Value = depositTxOpts.Value

	to := m.tx.To()
	min, err := optimismPortal2.MinimumGasLimit(&bind.CallOpts{}, uint64(len(m.tx.Data())))
	require.NoError(m.t, err)
	gas := max(m.tx.Gas(), min)
	tx, err := optimismPortal2.DepositTransaction(l1Opts, *to, m.tx.Value(), gas, to == nil, m.tx.Data())
	require.NoError(m.t, err, "failed to create deposit tx")
	rcpt := l1Miner.IncludeTx(m.t, tx)
	require.Equal(m.t, types.ReceiptStatusSuccessful, rcpt.Status, "deposit tx failed")
}

func (m *GeneratedTransaction) Identifier() inbox.Identifier {
	require.NotZero(m.t, len(m.rcpt.Logs), "Transaction did not include any logs to reference")

	return Identifier(m.chain, m.tx, m.rcpt)
}

func Identifier(chain *Chain, tx *types.Transaction, rcpt *types.Receipt) inbox.Identifier {
	blockTime := chain.RollupCfg.TimestampForBlock(bigs.Uint64Strict(rcpt.BlockNumber))
	return inbox.Identifier{
		Origin:      *tx.To(),
		BlockNumber: rcpt.BlockNumber,
		LogIndex:    new(big.Int).SetUint64(uint64(rcpt.Logs[0].Index)),
		Timestamp:   new(big.Int).SetUint64(blockTime),
		ChainId:     chain.RollupCfg.L2ChainID,
	}
}

func (m *GeneratedTransaction) MessagePayload() []byte {
	require.NotZero(m.t, len(m.rcpt.Logs), "Transaction did not include any logs to reference")
	return stypes.LogToMessagePayload(m.rcpt.Logs[0])
}

