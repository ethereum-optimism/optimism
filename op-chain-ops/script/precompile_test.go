package script

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

type EmbeddedExample struct {
	Foo uint64
}

func (e *EmbeddedExample) TwoFoo() uint64 {
	e.Foo *= 2
	return e.Foo
}

type ExamplePrecompile struct {
	EmbeddedExample

	Bar       *big.Int
	hello     string
	helloFrom string
}

var testErr = errors.New("test err")

func (e *ExamplePrecompile) Greet(name string) (string, error) {
	if name == "mallory" {
		return "", testErr
	}
	e.helloFrom = name
	return e.hello + " " + name + "!", nil
}

func (e *ExamplePrecompile) Things() (bar *big.Int, hello string, seen string) {
	return e.Bar, e.hello, e.helloFrom
}

func (e *ExamplePrecompile) AddAndMul(a, b, c uint64, x uint8) uint64 {
	return (a + b + c) * uint64(x)
}

func TestPrecompile(t *testing.T) {
	e := &ExamplePrecompile{hello: "Hola", EmbeddedExample: EmbeddedExample{Foo: 42}, Bar: big.NewInt(123)}
	p, err := NewPrecompile[*ExamplePrecompile](e)
	require.NoError(t, err)

	for k, v := range p.abiMethods {
		t.Logf("4byte: %x  ABI: %s  Go: %s", k, v.abiSignature, v.goName)
	}

	// input/output
	input := crypto.Keccak256([]byte("greet(string)"))[:4]
	input = append(input, b32(0x20)...)                 // offset
	input = append(input, b32(uint64(len("alice")))...) // length
	input = append(input, "alice"...)
	out, err := p.Run(input)
	require.NoError(t, err)
	require.Equal(t, e.helloFrom, "alice")
	require.Equal(t, out[:32], b32(0x20))
	require.Equal(t, out[32:32*2], b32(uint64(len("Hola alice!"))))
	require.Equal(t, out[32*2:32*3], rightPad32([]byte("Hola alice!")))

	// error handling
	input = crypto.Keccak256([]byte("greet(string)"))[:4]
	input = append(input, b32(0x20)...)                   // offset
	input = append(input, b32(uint64(len("mallory")))...) // length
	input = append(input, "mallory"...)
	out, err = p.Run(input)
	require.Equal(t, err, vm.ErrExecutionReverted)
	msg, err := abi.UnpackRevert(out)
	require.NoError(t, err, "must unpack revert data")
	require.True(t, strings.HasSuffix(msg, testErr.Error()), "revert data must end with the inner error")

	// field reads
	input = crypto.Keccak256([]byte("foo()"))[:4]
	out, err = p.Run(input)
	require.NoError(t, err)
	require.Equal(t, out, b32(42))

	input = crypto.Keccak256([]byte("twoFoo()"))[:4]
	out, err = p.Run(input)
	require.NoError(t, err)
	require.Equal(t, out, b32(42*2))

	// persistent state changes
	input = crypto.Keccak256([]byte("twoFoo()"))[:4]
	out, err = p.Run(input)
	require.NoError(t, err)
	require.Equal(t, out, b32(42*2*2))

	// multi-output
	input = crypto.Keccak256([]byte("things()"))[:4]
	out, err = p.Run(input)
	require.NoError(t, err)
	require.Equal(t, b32(123), out[:32])
	require.Equal(t, b32(32*3), out[32*1:32*2])                   // offset of hello
	require.Equal(t, b32(32*5), out[32*2:32*3])                   // offset of seen
	require.Equal(t, b32(uint64(len("Hola"))), out[32*3:32*4])    // length of hello
	require.Equal(t, rightPad32([]byte("Hola")), out[32*4:32*5])  // hello content
	require.Equal(t, b32(uint64(len("alice"))), out[32*5:32*6])   // length of seen
	require.Equal(t, rightPad32([]byte("alice")), out[32*6:32*7]) // seen content

	// multi-input
	input = crypto.Keccak256([]byte("addAndMul(uint64,uint64,uint64,uint8)"))[:4]
	input = append(input, b32(42)...)
	input = append(input, b32(100)...)
	input = append(input, b32(7)...)
	input = append(input, b32(3)...)
	out, err = p.Run(input)
	require.NoError(t, err)
	require.Equal(t, b32((42+100+7)*3), out)
}

type DeploymentExample struct {
	FooBar common.Address
}

func TestDeploymentOutputPrecompile(t *testing.T) {
	e := &DeploymentExample{}
	p, err := NewPrecompile[*DeploymentExample](e, WithFieldSetter[*DeploymentExample])
	require.NoError(t, err)

	addr := common.Address{0: 0x42, 19: 0xaa}
	fooBarSelector := bytes4("fooBar()")
	var input []byte
	input = append(input, setterFnBytes4[:]...)
	input = append(input, rightPad32(fooBarSelector[:])...)
	input = append(input, leftPad32(addr[:])...)
	out, err := p.Run(input)
	require.NoError(t, err)
	require.Empty(t, out)
	require.Equal(t, addr, e.FooBar)
}
