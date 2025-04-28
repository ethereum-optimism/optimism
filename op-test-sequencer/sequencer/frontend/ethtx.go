package frontend

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
)

type EthTxFrontend struct {
	Sequencer work.Sequencer
	Logger    log.Logger
}

func (etf *EthTxFrontend) SendRawTransaction(ctx context.Context, tx hexutil.Bytes) error {
	etf.Logger.Debug("EthTxFrontend SendRawTransaaction request", "tx", tx)

	incl, ok := etf.Sequencer.(IncludeTxSupport)
	if !ok {
		return &rpc.JsonError{Code: -39990, Message: "no tx inclusion supported"}
	}
	incl.IncludeTx(ctx, tx)

	return nil
}
