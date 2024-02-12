package l1

import (
	"math/big"

	test "github.com/ethereum-optimism/optimism/op-test"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type Name string

type Request struct {
	ActiveFork L1Fork
}

type Option interface {
	Apply(req *Request) error
}

type OptionFn func(req *Request) error

func (fn OptionFn) Apply(req *Request) error {
	return fn(req)
}

type L1Fork string

const (
	Shapella L1Fork = "shapella"
	Dencun   L1Fork = "dencun"
)

func (f L1Fork) String() string {
	return string(f)
}

var Forks = []L1Fork{
	Shapella,
	Dencun,
}

func ActiveFork(fork L1Fork) Option {
	return OptionFn(func(req *Request) error {
		req.ActiveFork = fork
		return nil
	})
}

type L1 interface {
	ChainID() *big.Int
	ChainConfig() *params.ChainConfig
	Signer() *types.Signer
}

type Backend interface {
	RequestL1(Name, ...Option) L1
}

func NewBackend(t test.Testing, kind test.BackendKind) Backend {
	switch kind {
	case test.Live:
		return &LiveBackend{T: t}
	case test.Managed:
		return &ManagedBackend{T: t}
	case test.Instant:
		return &InstantBackend{T: t}
	default:
		t.Fatalf("unrecognized L1 backend type: %q", kind)
		return nil
	}
}
