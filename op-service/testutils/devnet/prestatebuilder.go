package devnet

import (
	"context"

	"github.com/ethereum-optimism/optimism/devnet-sdk/proofs/prestate"
)

type mockPreStateBuilder struct {
	buildPrestate func(ctx context.Context, opts ...prestate.PrestateBuilderOption) (prestate.PrestateManifest, error)
	invocations   int
	lastOptsCount int
}

type mockPreStateBuilderOption func(*mockPreStateBuilder)

func WithBuildPrestate(buildPrestate func(ctx context.Context, opts ...prestate.PrestateBuilderOption) (prestate.PrestateManifest, error)) mockPreStateBuilderOption {
	return func(m *mockPreStateBuilder) {
		m.buildPrestate = buildPrestate
	}
}

func NewMockPreStateBuilder(opts ...mockPreStateBuilderOption) *mockPreStateBuilder {
	output := &mockPreStateBuilder{
		buildPrestate: func(ctx context.Context, opts ...prestate.PrestateBuilderOption) (prestate.PrestateManifest, error) {
			return prestate.PrestateManifest(map[string]string{
				"prestate":         "0x0374fe3399429aed8c34cb33608da7e15cdab7f7aba6d9e994a1fdb9dd04e1a3",
				"prestate_interop": "0x0329740c6b4f3441e11ee61a920d6a2ebca6cf6e246076eaf8b500d6a90bb6e2",
				"prestate_mt64":    "0x03e0fd1bc0e5fc6b77542f9834975b6bf55cd0008e64e8c50c47d157292f17cc",
			}), nil
		},
	}

	for _, opt := range opts {
		opt(output)
	}

	return output
}

func (m *mockPreStateBuilder) BuildPrestate(ctx context.Context, opts ...prestate.PrestateBuilderOption) (prestate.PrestateManifest, error) {
	m.invocations++
	m.lastOptsCount = len(opts)
	return m.buildPrestate(ctx, opts...)
}

func (m *mockPreStateBuilder) Invocations() int {
	return m.invocations
}

func (m *mockPreStateBuilder) LastOptsCount() int {
	return m.lastOptsCount
}
