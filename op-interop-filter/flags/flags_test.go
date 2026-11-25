package flags

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestDefaultBackfillCoversMessageExpiryWindow(t *testing.T) {
	backfillFlag := *BackfillDurationFlag
	backfillFlag.EnvVars = nil
	messageExpiryFlag := *MessageExpiryWindowFlag
	messageExpiryFlag.EnvVars = nil

	app := cli.NewApp()
	app.Flags = []cli.Flag{&backfillFlag, &messageExpiryFlag}
	app.Action = func(ctx *cli.Context) error {
		backfillDuration := ctx.Duration(backfillFlag.Name)
		messageExpiryWindow := ctx.Duration(messageExpiryFlag.Name)
		require.Equal(t, 7*24*time.Hour, backfillDuration)
		require.Equal(t, messageExpiryWindow, backfillDuration)
		return nil
	}

	require.NoError(t, app.Run([]string{"op-interop-filter"}))
}
