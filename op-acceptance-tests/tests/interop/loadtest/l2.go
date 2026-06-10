package loadtest

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-service/txspam"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type L2 struct {
	Config      *params.ChainConfig
	BlockTime   time.Duration
	EL          *dsl.L2ELNode
	EOAs        *txspam.RoundRobin[*txspam.SyncEOA]
	EventLogger common.Address
	Wallet      *dsl.HDWallet
}

func (l2 *L2) DeployEventLogger(t devtest.T) {
	tx, err := l2.Include(t, txplan.WithData(common.FromHex(bindings.EventloggerBin)))
	t.Require().NoError(err)
	l2.EventLogger = tx.Receipt.ContractAddress
}

// Include includes the transaction on l2. It guarantees that the returned transaction was executed
// successfully when the error is non-nil.
func (l2 *L2) Include(t devtest.T, opts ...txplan.Option) (*txinclude.IncludedTx, error) {
	includedTx, err := l2.EOAs.Get().Include(t.Ctx(), opts...)
	if err != nil {
		return nil, err
	}
	t.Require().Equal(ethtypes.ReceiptStatusSuccessful, includedTx.Receipt.Status)
	return includedTx, nil
}
