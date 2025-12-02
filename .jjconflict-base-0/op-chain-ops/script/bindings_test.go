package script

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

type EmbeddedBindings struct {
	Adder func(a uint8, b uint64) *big.Int
}

type ExampleBindings struct {
	DoThing func() error
	EmbeddedBindings
	Hello func(greeting string, target common.Address) string
}

func TestBindings(t *testing.T) {
	var testFn CallBackendFn
	backendFn := func(data []byte) ([]byte, error) {
		return testFn(data) // indirect call, so we can swap it per test case.
	}
	bindings, err := MakeBindings[ExampleBindings](backendFn, nil)
	require.NoError(t, err)

	testFn = func(data []byte) ([]byte, error) {
		require.Len(t, data, 4)
		require.Equal(t, bytes4("doThing()"), [4]byte(data))
		return encodeRevert(errors.New("example revert"))
	}
	err = bindings.DoThing()
	require.ErrorContains(t, err, "example revert")

	testFn = func(data []byte) ([]byte, error) {
		require.Len(t, data, 4)
		require.Equal(t, bytes4("doThing()"), [4]byte(data))
		return []byte{}, nil
	}
	err = bindings.DoThing()
	require.NoError(t, err, "no revert")

	testFn = func(data []byte) ([]byte, error) {
		require.Len(t, data, 4+32+32, "selector and two ABI args")
		require.Equal(t, bytes4("adder(uint8,uint64)"), [4]byte(data[:4]))
		a := new(big.Int).SetBytes(data[4 : 4+32])
		b := new(big.Int).SetBytes(data[4+32:])
		return leftPad32(new(big.Int).Add(a, b).Bytes()), nil
	}
	result := bindings.Adder(42, 0x1337)
	require.NoError(t, err)
	require.True(t, result.IsUint64())
	require.Equal(t, uint64(42+0x1337), result.Uint64())
}

type TestContract struct{}

func (e *TestContract) Hello(greeting string, target common.Address) string {
	return fmt.Sprintf("Test says: %s %s!", greeting, target)
}

func TestPrecompileBindings(t *testing.T) {
	contract, err := NewPrecompile[*TestContract](&TestContract{})
	require.NoError(t, err)

	bindings, err := MakeBindings[ExampleBindings](contract.Run, nil)
	require.NoError(t, err)

	addr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	response := bindings.Hello("Hola", addr)
	require.Equal(t, fmt.Sprintf("Test says: Hola %s!", addr), response)
}

func TestBindingsABICheck(t *testing.T) {
	fn := CallBackendFn(func(data []byte) ([]byte, error) {
		panic("should not run")
	})
	needABI := map[string]struct{}{
		"doThing()":             {},
		"adder(uint8,uint64)":   {},
		"hello(string,address)": {},
	}
	gotABI := make(map[string]struct{})
	abiCheckFn := func(abiDef string) bool {
		_, ok := needABI[abiDef]
		gotABI[abiDef] = struct{}{}
		return ok
	}
	_, err := MakeBindings[ExampleBindings](fn, abiCheckFn)
	require.NoError(t, err)
	require.Equal(t, needABI, gotABI, "checked all ABI methods")

	delete(needABI, "doThing()")
	_, err = MakeBindings[ExampleBindings](fn, abiCheckFn)
	require.ErrorIs(t, err, ErrABICheck)
}
