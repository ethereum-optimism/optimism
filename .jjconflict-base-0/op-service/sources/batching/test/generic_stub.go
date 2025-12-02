package test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ExpectedRpcCall interface {
	fmt.Stringer
	Matches(rpcMethod string, args ...interface{}) error
	Execute(t *testing.T, out interface{}) error
}

type RpcStub struct {
	t             *testing.T
	expectedCalls []ExpectedRpcCall
}

func NewRpcStub(t *testing.T) *RpcStub {
	return &RpcStub{t: t}
}

func (r *RpcStub) ClearResponses() {
	r.expectedCalls = nil
}

func (r *RpcStub) AddExpectedCall(call ExpectedRpcCall) {
	r.expectedCalls = append(r.expectedCalls, call)
}

func (r *RpcStub) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	var errs []error
	for _, elem := range b {
		elem.Error = r.CallContext(ctx, elem.Result, elem.Method, elem.Args...)
		errs = append(errs, elem.Error)
	}
	return errors.Join(errs...)
}

func (r *RpcStub) CallContext(_ context.Context, out interface{}, method string, args ...interface{}) error {
	call := r.findExpectedCall(method, args...)
	return call.Execute(r.t, out)
}

func (r *RpcStub) findExpectedCall(rpcMethod string, args ...interface{}) ExpectedRpcCall {
	var matchResults string
	for _, call := range r.expectedCalls {
		if err := call.Matches(rpcMethod, args...); err == nil {
			return call
		} else {
			matchResults += fmt.Sprintf("%v: %v\n", call, err)
		}
	}
	require.Failf(r.t, "No matching expected calls.", matchResults)
	return nil
}

type GenericExpectedCall struct {
	method string
	args   []interface{}
	result interface{}
}

func NewGetBalanceCall(addr common.Address, block rpcblock.Block, balance *big.Int) ExpectedRpcCall {
	return &GenericExpectedCall{
		method: "eth_getBalance",
		args:   []interface{}{addr, block.ArgValue()},
		result: (*hexutil.Big)(balance),
	}
}

func (c *GenericExpectedCall) Matches(rpcMethod string, args ...interface{}) error {
	if rpcMethod != c.method {
		return fmt.Errorf("expected method %v but was %v", c.method, rpcMethod)
	}
	if !assert.ObjectsAreEqualValues(c.args, args) {
		return fmt.Errorf("expected args %v but was %v", c.args, args)
	}
	return nil
}

func (c *GenericExpectedCall) Execute(t *testing.T, out interface{}) error {
	// I admit I do not understand Go reflection.
	// So leverage json.Unmarshal to set the out value correctly.
	j, err := json.Marshal(c.result)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(j, out))
	return nil
}

func (c *GenericExpectedCall) String() string {
	return fmt.Sprintf("%v(%v)->%v", c.method, c.args, c.result)
}
