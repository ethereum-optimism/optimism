package op_challenger

import (
	"context"
	"testing"

	"github.com/HashKeyChain/verse/op-challenger/config"
	"github.com/HashKeyChain/verse/op-challenger/metrics"
	"github.com/HashKeyChain/verse/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestMainShouldReturnErrorWhenConfigInvalid(t *testing.T) {
	cfg := &config.Config{}
	app, err := Main(context.Background(), testlog.Logger(t, log.LevelInfo), cfg, metrics.NoopMetrics)
	require.ErrorIs(t, err, cfg.Check())
	require.Nil(t, app)
}
