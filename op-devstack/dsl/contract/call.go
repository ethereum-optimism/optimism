package contract

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	// readTimeout bounds individual contract read calls to prevent tests from
	// hanging when an in-memory geth node stalls under CI resource contention.
	readTimeout = 60 * time.Second
	// writeTimeout bounds contract write calls (transaction submission + mining).
	writeTimeout = 5 * time.Minute
)

// TestCallView is used in devstack for wrapping errors
type TestCallView[O any] interface {
	Test() bindings.BaseTest
}

// checkTestable checks whether the TypedCall can be used as a DSL using the testing context
func checkTestable[O any](call bindings.TypedCall[O]) {
	callTest, ok := any(call).(TestCallView[O])
	if !ok || callTest.Test() == nil {
		panic(fmt.Sprintf("call of type %T does not support testing", call))
	}
}

// Read executes a new message call without creating a transaction on the blockchain.
// Each call is bounded by readTimeout to prevent hangs under CI resource contention.
func Read[O any](call bindings.TypedCall[O], opts ...txplan.Option) O {
	checkTestable(call)
	ctx, cancel := context.WithTimeout(call.Test().Ctx(), readTimeout)
	defer cancel()
	o, err := contractio.Read(call, ctx, opts...)
	call.Test().Require().NoError(err)
	return o
}

// ReadArray retrieves all data from an array in batches.
// Each call is bounded by readTimeout to prevent hangs under CI resource contention.
func ReadArray[T any](countCall bindings.TypedCall[*big.Int], elemCall func(i *big.Int) bindings.TypedCall[T]) []T {
	checkTestable(countCall)
	test := countCall.Test()
	ctx, cancel := context.WithTimeout(countCall.Test().Ctx(), readTimeout)
	defer cancel()

	caller := countCall.Client().NewMultiCaller(batching.DefaultBatchSize)

	o, err := contractio.ReadArray(ctx, caller, countCall, elemCall)
	test.Require().NoError(err)
	return o
}

// Write makes a user to write a tx by using the planned contract bindings.
// Each call is bounded by writeTimeout to prevent hangs under CI resource contention.
func Write[O any](user *dsl.EOA, call bindings.TypedCall[O], opts ...txplan.Option) *types.Receipt {
	checkTestable(call)
	ctx, cancel := context.WithTimeout(call.Test().Ctx(), writeTimeout)
	defer cancel()
	finalOpts := txplan.Combine(user.Plan(), txplan.Combine(opts...))
	o, err := contractio.Write(call, ctx, finalOpts)
	call.Test().Require().NoError(err, "contract write failed: %v", errutil.TryAddRevertReason(err))
	return o
}

var _ TestCallView[any] = (*bindings.TypedCall[any])(nil)
