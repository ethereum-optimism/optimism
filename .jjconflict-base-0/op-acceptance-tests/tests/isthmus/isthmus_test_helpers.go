package isthmus

import (
	"context"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm/program"
)

func DefaultTxSubmitOptions(w system.WalletV2) txplan.Option {
	return txplan.Combine(
		txplan.WithPrivateKey(w.PrivateKey()),
		txplan.WithChainID(w.Client()),
		txplan.WithAgainstLatestBlock(w.Client()),
		txplan.WithPendingNonce(w.Client()),
		txplan.WithEstimator(w.Client(), false),
		txplan.WithTransactionSubmitter(w.Client()),
	)
}

func DefaultTxInclusionOptions(w system.WalletV2) txplan.Option {
	return txplan.Combine(
		txplan.WithRetryInclusion(w.Client(), 10, retry.Exponential()),
		txplan.WithBlockInclusionInfo(w.Client()),
	)
}

func DefaultTxOpts(w system.WalletV2) txplan.Option {
	return txplan.Combine(
		DefaultTxSubmitOptions(w),
		DefaultTxInclusionOptions(w),
	)
}

func DeployProgram(ctx context.Context, wallet system.WalletV2, code []byte) (common.Address, error) {
	deployProgram := program.New().ReturnViaCodeCopy(code)

	opts := DefaultTxOpts(wallet)
	deployTx := txplan.NewPlannedTx(opts, txplan.WithData(deployProgram.Bytes()))

	res, err := deployTx.Included.Eval(ctx)
	if err != nil {
		return common.Address{}, err
	}
	ctrctAddr := res.ContractAddress
	return ctrctAddr, err
}
