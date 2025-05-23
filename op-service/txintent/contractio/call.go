package contractio

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Write receives a Call and uses to plan transaction, and attempts to write.
func Write(call txintent.Call, ctx context.Context, opts ...txplan.Option) (*types.Receipt, error) {
	plan, err := Plan(call)
	if err != nil {
		return nil, err
	}
	tx := txplan.NewPlannedTx(plan, txplan.Combine(opts...))
	receipt, err := tx.Included.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

// Read receives a CallView and uses to plan transaction, and attempts to read.
func Read[O any](view txintent.CallView[O], ctx context.Context, opts ...txplan.Option) (O, error) {
	plan, err := Plan(view)
	if err != nil {
		return *new(O), err
	}
	client := view.Client()
	tx := txplan.NewPlannedTx(
		plan,
		txplan.WithAgainstLatestBlock(client),
		txplan.WithReader(client),
		// use default sender as null
		txplan.WithSender(common.Address{}),
		txplan.Combine(opts...),
	)
	res, err := tx.Read.Eval(ctx)
	if err != nil {
		return *new(O), err
	}
	decoded, err := view.DecodeOutput(res)
	if err != nil {
		return *new(O), err
	}
	return decoded, nil
}

func Plan(call txintent.Call) (txplan.Option, error) {
	target, err := call.To()
	if err != nil {
		return nil, err
	}
	calldata, err := call.EncodeInput()
	if err != nil {
		return nil, err
	}
	tx := txplan.Combine(
		txplan.WithData(calldata),
		txplan.WithTo(target),
	)
	return tx, nil
}
