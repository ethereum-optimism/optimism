package nooppublisher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/HashKeyChain/verse/op-service/testlog"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/seqtypes"
)

func TestNoopPublisher(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	id := seqtypes.PublisherID("foo")
	x := NewPublisher(id, logger)

	err := x.Publish(context.Background(), nil)
	require.NoError(t, err)

	require.NoError(t, x.Close())
	require.Equal(t, "noop-publisher-foo", x.String())
	require.Equal(t, id, x.ID())
}
