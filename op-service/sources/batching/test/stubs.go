package test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

type expectedCall struct {
	block      batching.Block
	args       []interface{}
	packedArgs []byte
	outputs    []interface{}
}

func (e *expectedCall) String() string {
	return fmt.Sprintf("{block: %v, args: %v, outputs: %v}", e.block, e.args, e.outputs)
}

type AbiBasedRpc struct {
	t    *testing.T
	abi  *abi.ABI
	addr common.Address

	expectedCalls map[string][]*expectedCall
}

func NewAbiBasedRpc(t *testing.T, contractAbi *abi.ABI, addr common.Address) *AbiBasedRpc {
	return &AbiBasedRpc{
		t:             t,
		abi:           contractAbi,
		addr:          addr,
		expectedCalls: make(map[string][]*expectedCall),
	}
}

func (l *AbiBasedRpc) SetResponse(method string, block batching.Block, expected []interface{}, output []interface{}) {
	if expected == nil {
		expected = []interface{}{}
	}
	if output == nil {
		output = []interface{}{}
	}
	abiMethod, ok := l.abi.Methods[method]
	require.Truef(l.t, ok, "No method: %v", method)
	packedArgs, err := abiMethod.Inputs.Pack(expected...)
	require.NoErrorf(l.t, err, "Invalid expected arguments for method %v: %v", method, expected)
	l.expectedCalls[method] = append(l.expectedCalls[method], &expectedCall{
		block:      block,
		args:       expected,
		packedArgs: packedArgs,
		outputs:    output,
	})
}

func (l *AbiBasedRpc) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	var errs []error
	for _, elem := range b {
		elem.Error = l.CallContext(ctx, elem.Result, elem.Method, elem.Args...)
		errs = append(errs, elem.Error)
	}
	return errors.Join(errs...)
}

func (l *AbiBasedRpc) VerifyTxCandidate(candidate txmgr.TxCandidate) {
	require.EqualValues(l.t, &l.addr, candidate.To, "Incorrect To address")
	l.findExpectedCall(candidate.TxData, batching.BlockLatest.ArgValue())
}

func (l *AbiBasedRpc) CallContext(_ context.Context, out interface{}, method string, args ...interface{}) error {
	require.Equal(l.t, "eth_call", method)
	require.Len(l.t, args, 2)
	actualBlockRef := args[1]
	callOpts, ok := args[0].(map[string]any)
	require.True(l.t, ok)
	require.Equal(l.t, &l.addr, callOpts["to"])
	data, ok := callOpts["input"].(hexutil.Bytes)
	require.True(l.t, ok)

	call, abiMethod := l.findExpectedCall(data, actualBlockRef)

	output, err := abiMethod.Outputs.Pack(call.outputs...)
	require.NoErrorf(l.t, err, "Invalid outputs for method %v: %v", abiMethod.Name, call.outputs)

	// I admit I do not understand Go reflection.
	// So leverage json.Unmarshal to set the out value correctly.
	j, err := json.Marshal(hexutil.Bytes(output))
	require.NoError(l.t, err)
	require.NoError(l.t, json.Unmarshal(j, out))
	return nil
}

func (l *AbiBasedRpc) findExpectedCall(data []byte, actualBlockRef interface{}) (*expectedCall, *abi.Method) {

	abiMethod, err := l.abi.MethodById(data[0:4])
	require.NoError(l.t, err)

	argData := data[4:]
	args, err := abiMethod.Inputs.Unpack(argData)
	require.NoError(l.t, err)
	require.Len(l.t, args, len(abiMethod.Inputs))

	expectedCalls, ok := l.expectedCalls[abiMethod.Name]
	require.Truef(l.t, ok, "Unexpected call to %v", abiMethod.Name)
	var call *expectedCall
	for _, candidate := range expectedCalls {
		if slices.Equal(candidate.packedArgs, argData) && assert.ObjectsAreEqualValues(candidate.block.ArgValue(), actualBlockRef) {
			call = candidate
			break
		}
	}
	require.NotNilf(l.t, call, "No expected calls to %v at block %v with arguments: %v\nExpected calls: %v", abiMethod.Name, actualBlockRef, args, expectedCalls)
	return call, abiMethod
}
