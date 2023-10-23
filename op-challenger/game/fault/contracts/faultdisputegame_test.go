package contracts

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestGetStatus(t *testing.T) {
	stubRpc, game := setup(t)
	stubRpc.SetResponse("status", nil, []interface{}{types.GameStatusChallengerWon})
	status, err := game.GetStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, types.GameStatusChallengerWon, status)
}

func TestGetClaim(t *testing.T) {
	stubRpc, game := setup(t)
	idx := big.NewInt(2)
	parentIndex := uint32(1)
	countered := true
	value := common.Hash{0xab}
	position := big.NewInt(2)
	clock := big.NewInt(1234)
	stubRpc.SetResponse("claimData", []interface{}{idx}, []interface{}{parentIndex, countered, value, position, clock})
	status, err := game.GetClaim(context.Background(), idx.Uint64())
	require.NoError(t, err)
	require.Equal(t, faultTypes.Claim{
		ClaimData: faultTypes.ClaimData{
			Value:    value,
			Position: faultTypes.NewPositionFromGIndex(position),
		},
		Countered:           true,
		Clock:               1234,
		ContractIndex:       int(idx.Uint64()),
		ParentContractIndex: 1,
	}, status)
}

type abiBasedRpc struct {
	t    *testing.T
	abi  *abi.ABI
	addr common.Address

	expectedArgs map[string][]interface{}
	outputs      map[string][]interface{}
}

func setup(t *testing.T) (*abiBasedRpc, *FaultDisputeGameContract) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	require.NoError(t, err)
	address := common.HexToAddress("0x24112842371dFC380576ebb09Ae16Cb6B6caD7CB")

	stubRpc := &abiBasedRpc{
		t:            t,
		abi:          fdgAbi,
		addr:         address,
		expectedArgs: make(map[string][]interface{}),
		outputs:      make(map[string][]interface{}),
	}
	caller := batching.NewMultiCaller(stubRpc, 1)
	game, err := NewFaultDisputeGameContract(address, caller)
	require.NoError(t, err)
	return stubRpc, game
}

func (l *abiBasedRpc) SetResponse(method string, expected []interface{}, output []interface{}) {
	if expected == nil {
		expected = []interface{}{}
	}
	if output == nil {
		output = []interface{}{}
	}
	l.expectedArgs[method] = expected
	l.outputs[method] = output
}

func (l *abiBasedRpc) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	panic("Not implemented")
}

func (l *abiBasedRpc) CallContext(ctx context.Context, out interface{}, method string, args ...interface{}) error {
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
	require.True(l.t, ok)
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
