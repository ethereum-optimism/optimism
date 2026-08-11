package e2esys

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/services"
	"github.com/stretchr/testify/require"
)

func TestSystemStartWaitsForL1BlockPause(t *testing.T) {
	cfg := DefaultSystemConfig(t, WithL2ELKind(services.ELKindOpGeth))
	cfg.Nodes = nil

	realAcknowledgement := make(chan (<-chan struct{}), 1)
	gatedAcknowledgement := make(chan struct{})
	pause := WithL1BlockPauseAtBlock2()
	pauseWait := func(reached <-chan struct{}) <-chan struct{} {
		realAcknowledgement <- reached
		return gatedAcknowledgement
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	handshakeComplete := make(chan struct{})
	defer func() {
		cancel()
		<-handshakeComplete
	}()
	sendCompleted := make(chan struct{})
	go func() {
		defer close(handshakeComplete)
		var reached <-chan struct{}
		select {
		case reached = <-realAcknowledgement:
		case <-ctx.Done():
			return
		}
		select {
		case <-reached:
		case <-ctx.Done():
			return
		}
		select {
		case gatedAcknowledgement <- struct{}{}:
			close(sendCompleted)
		case <-ctx.Done():
		}
	}()

	sys, err := cfg.start(t, pauseWait, pause)
	require.NoError(t, err)
	select {
	case <-sendCompleted:
	case <-ctx.Done():
		require.NoError(t, ctx.Err(), "completing the real and gated L1 pause acknowledgements")
	}
	block, err := sys.NodeClient(RoleL1).BlockNumber(t.Context())
	require.NoError(t, err)
	require.Equal(t, uint64(2), block)
	require.NoError(t, sys.ResumeL1())
}
