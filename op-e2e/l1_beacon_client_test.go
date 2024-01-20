package op_e2e

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestGetVersion(t *testing.T) {
	l := testlog.Logger(t, log.LvlInfo)

	beaconApi := fakebeacon.NewBeacon(l, t.TempDir(), uint64(0), uint64(0))
	t.Cleanup(func() {
		_ = beaconApi.Close()
	})
	require.NoError(t, beaconApi.Start("127.0.0.1:0"))

	cl := sources.NewL1BeaconClient(client.NewBasicHTTPClient(beaconApi.BeaconAddr(), l))

	version, err := cl.GetVersion(context.Background())
	require.NoError(t, err)
	require.Equal(t, "fakebeacon 1.2.3", version)
}
