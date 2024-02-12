package l1el

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	test "github.com/ethereum-optimism/optimism/op-test"
	"github.com/ethereum-optimism/optimism/op-test/components/l1"
)

type Name string

type Request struct {
	// active block-building, peering, etc. config
	BlockBuilding bool
}

func RequestFromOpts(t test.Testing, opts []Option) *Request {
	var req Request
	for i, opt := range opts {
		require.NoError(t, opt.Apply(&req), "must apply option %d", i)
	}
	return &req
}

type Option interface {
	Apply(req *Request) error
}

type OptionFn func(req *Request) error

func (fn OptionFn) Apply(req *Request) error {
	return fn(req)
}

type L1EL interface {
	RPC() client.RPC
	L1Client() *sources.L1Client
}

func BlockBuilding(v bool) Option {
	return OptionFn(func(req *Request) error {
		req.BlockBuilding = v
		return nil
	})
}

type Backend interface {
	RequestL1EL(Name, ...Option) L1EL
}

func NewBackend(t test.Testing, l1Ch l1.L1, backendKind test.BackendKind) Backend {
	switch backendKind {
	case test.Live:
		return &LiveBackend{T: t, L1: l1Ch}
	case test.Managed:
		return &ManagedBackend{T: t, L1: l1Ch}
	case test.Instant:
		return &InstantBackend{T: t, L1: l1Ch}
	default:
		t.Fatalf("unknown backend type %q", backendKind)
		panic("no backend")
	}
}
