package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type expectedCall struct {
	abiMethod  abi.Method
	to         common.Address
	block      rpcblock.Block
	args       []interface{}
	packedArgs []byte
	outputs    []interface{}
	err        error
}

func (c *expectedCall) Matches(rpcMethod string, args ...interface{}) error {
	if rpcMethod != "eth_call" {
		return fmt.Errorf("expected rpcMethod eth_call but was %v", rpcMethod)
	}
	if len(args) != 2 {
		return fmt.Errorf("expected arg count 2 but was %v", len(args))
	}
	callOpts, ok := args[0].(map[string]any)
	if !ok {
		return errors.New("arg 0 is not a map[string]any")
	}
	actualBlockRef := args[1]
	to, ok := callOpts["to"].(*common.Address)
	if !ok {
		return errors.New("to is not an address")
	}
	if to == nil {
		return errors.New("to is nil")
	}
	if *to != c.to {
		return fmt.Errorf("expected to %v but was %v", c.to, *to)
	}
	data, ok := callOpts["input"].(hexutil.Bytes)
	if !ok {
		return errors.New("input is not hexutil.Bytes")
	}
	if len(data) < 4 {
		return fmt.Errorf("expected input to have at least 4 bytes but was %v", len(data))
	}
	if !slices.Equal(c.abiMethod.ID, data[:4]) {
		return fmt.Errorf("expected abi method ID %x but was %v", c.abiMethod.ID, data[:4])
	}
	if !slices.Equal(c.packedArgs, data[4:]) {
		return fmt.Errorf("expected args %x but was %x", c.packedArgs, data[4:])
	}
	if !assert.ObjectsAreEqualValues(c.block.ArgValue(), actualBlockRef) {
		return fmt.Errorf("expected block ref %v but was %v", c.block.ArgValue(), actualBlockRef)
	}
	return nil
}

func (c *expectedCall) Execute(t *testing.T, out interface{}) error {
	output, err := c.abiMethod.Outputs.Pack(c.outputs...)
	require.NoErrorf(t, err, "Invalid outputs for method %v: %v", c.abiMethod.Name, c.outputs)

	// I admit I do not understand Go reflection.
	// So leverage json.Unmarshal to set the out value correctly.
	j, err := json.Marshal(hexutil.Bytes(output))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(j, out))
	return c.err
}

func (c *expectedCall) String() string {
	return fmt.Sprintf("{to: %v, block: %v, args: %v, outputs: %v}", c.to, c.block, c.args, c.outputs)
}

type AbiBasedRpc struct {
	RpcStub
	abis map[common.Address]*abi.ABI
}

func NewAbiBasedRpc(t *testing.T, to common.Address, contractAbi *abi.ABI) *AbiBasedRpc {
	abis := make(map[common.Address]*abi.ABI)
	abis[to] = contractAbi
	return &AbiBasedRpc{
		RpcStub: RpcStub{
			t: t,
		},
		abis: abis,
	}
}

func (l *AbiBasedRpc) AddContract(to common.Address, contractAbi *abi.ABI) {
	l.abis[to] = contractAbi
}

func (l *AbiBasedRpc) abi(to common.Address) *abi.ABI {
	abi, ok := l.abis[to]
	require.Truef(l.t, ok, "Missing ABI for %v", to)
	return abi
}

func (l *AbiBasedRpc) SetError(to common.Address, method string, block rpcblock.Block, expected []interface{}, callErr error) {
	if expected == nil {
		expected = []interface{}{}
	}
	abiMethod, ok := l.abi(to).Methods[method]
	require.Truef(l.t, ok, "No method: %v", method)
	packedArgs, err := abiMethod.Inputs.Pack(expected...)
	require.NoErrorf(l.t, err, "Invalid expected arguments for method %v: %v", method, expected)
	l.AddExpectedCall(&expectedCall{
		abiMethod:  abiMethod,
		to:         to,
		block:      block,
		args:       expected,
		packedArgs: packedArgs,
		outputs:    []interface{}{},
		err:        callErr,
	})
}
func (l *AbiBasedRpc) SetResponse(to common.Address, method string, block rpcblock.Block, expected []interface{}, output []interface{}) {
	if expected == nil {
		expected = []interface{}{}
	}
	if output == nil {
		output = []interface{}{}
	}
	abiMethod, ok := l.abi(to).Methods[method]
	require.Truef(l.t, ok, "No method: %v", method)
	packedArgs, err := abiMethod.Inputs.Pack(expected...)
	require.NoErrorf(l.t, err, "Invalid expected arguments for method %v: %v", method, expected)
	l.AddExpectedCall(&expectedCall{
		abiMethod:  abiMethod,
		to:         to,
		block:      block,
		args:       expected,
		packedArgs: packedArgs,
		outputs:    output,
	})
}

func (l *AbiBasedRpc) VerifyTxCandidate(candidate txmgr.TxCandidate) {
	require.NotNil(l.t, candidate.To)
	l.findExpectedCall("eth_call", map[string]any{
		"to":    candidate.To,
		"input": hexutil.Bytes(candidate.TxData),
		"value": candidate.Value,
	}, rpcblock.Latest.ArgValue())
}
