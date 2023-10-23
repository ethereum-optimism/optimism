package test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

type AbiBasedRpc struct {
	t    *testing.T
	abi  *abi.ABI
	addr common.Address

	expectedArgs map[string][]interface{}
	outputs      map[string][]interface{}
}

func NewAbiBasedRpc(t *testing.T, contractAbi *abi.ABI, addr common.Address) *AbiBasedRpc {
	return &AbiBasedRpc{
		t:            t,
		abi:          contractAbi,
		addr:         addr,
		expectedArgs: make(map[string][]interface{}),
		outputs:      make(map[string][]interface{}),
	}
}

func (l *AbiBasedRpc) SetResponse(method string, expected []interface{}, output []interface{}) {
	if expected == nil {
		expected = []interface{}{}
	}
	if output == nil {
		output = []interface{}{}
	}
	l.expectedArgs[method] = expected
	l.outputs[method] = output
}

func (l *AbiBasedRpc) BatchCallContext(_ context.Context, b []rpc.BatchElem) error {
	panic("Not implemented")
}

func (l *AbiBasedRpc) CallContext(_ context.Context, out interface{}, method string, args ...interface{}) error {
	require.Equal(l.t, "eth_call", method)
	require.Len(l.t, args, 2)
	require.Equal(l.t, "latest", args[1])
	callOpts, ok := args[0].(map[string]any)
	require.True(l.t, ok)
	require.Equal(l.t, &l.addr, callOpts["to"])
	data, ok := callOpts["data"].(hexutil.Bytes)
	require.True(l.t, ok)
	abiMethod, err := l.abi.MethodById(data[0:4])
	require.NoError(l.t, err)

	args, err = abiMethod.Inputs.Unpack(data[4:])
	require.NoError(l.t, err)
	require.Len(l.t, args, len(abiMethod.Inputs))
	expectedArgs, ok := l.expectedArgs[abiMethod.Name]
	require.Truef(l.t, ok, "Unexpected call to %v", abiMethod.Name)
	require.EqualValues(l.t, expectedArgs, args, "Unexpected args")

	outputs, ok := l.outputs[abiMethod.Name]
	require.True(l.t, ok)
	output, err := abiMethod.Outputs.Pack(outputs...)
	require.NoError(l.t, err)

	// I admit I do not understand Go reflection.
	// So leverage json.Unmarshal to set the out value correctly.
	j, err := json.Marshal(hexutil.Bytes(output))
	require.NoError(l.t, err)
	require.NoError(l.t, json.Unmarshal(j, out))
	return nil
}
