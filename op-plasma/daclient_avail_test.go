//go:build avail
// +build avail

package plasma

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-plasma/cmd/adapters"
	avail "github.com/ethereum-optimism/optimism/op-plasma/cmd/avail/service"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

const (
	RPC     = ""                // RPC URL
	SEED    = ""                // SEED PHRASE
	APPID   = 0                 // APPID
	TIMEOUT = 100 * time.Second // Timeout
)

func Check() error {
	if RPC == "" {
		return errors.New("no rpc url provided")
	}
	if APPID == 0 {
		return errors.New("no app id provided")
	}
	if SEED == "" {
		return errors.New("seedphrase not provided")
	}
	return nil
}

func TestAvailDAClientService(t *testing.T) {
	err := Check()
	if err != nil {
		panic(err)
	}
	store := avail.NewAvailService(RPC, SEED, APPID, TIMEOUT)
	logger := testlog.Logger(t, log.LevelDebug)

	ctx := context.Background()

	server := NewDAServer("127.0.0.1", 0, adapters.DAServiceAdapter{DAService: store}, logger, true)

	require.NoError(t, server.Start())

	cfg := CLIConfig{
		Enabled:      true,
		DAServerURL:  fmt.Sprintf("http://%s", server.Endpoint()),
		VerifyOnRead: false,
		GenericDA:    true,
	}
	require.NoError(t, cfg.Check())

	client := cfg.NewDAClient()

	rng := rand.New(rand.NewSource(1234))

	input := RandomData(rng, 2000)

	comm, err := client.SetInput(ctx, input)
	require.NoError(t, err)

	stored, err := client.GetInput(ctx, comm)
	require.NoError(t, err)
	require.Equal(t, stored, input)
}
