package challenger

import (
	"context"
	"errors"
	"testing"
	"time"

	op_challenger "github.com/ethereum-optimism/optimism/op-challenger"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type Helper struct {
	log    log.Logger
	cancel func()
	errors chan error
}

type Option func(config2 *config.Config)

func NewChallenger(t *testing.T, ctx context.Context, l1Endpoint string, name string, options ...Option) *Helper {
	log := testlog.Logger(t, log.LvlInfo).New("role", name)
	log.Info("Creating challenger", "l1", l1Endpoint)
	txmgrCfg := txmgr.NewCLIConfig(l1Endpoint)
	txmgrCfg.NumConfirmations = 1
	txmgrCfg.ReceiptQueryInterval = 1 * time.Second
	cfg := &config.Config{
		L1EthRpc:                l1Endpoint,
		AlphabetTrace:           "",
		AgreeWithProposedOutput: true,
		TxMgrConfig:             txmgrCfg,
	}
	for _, option := range options {
		option(cfg)
	}
	require.NotEmpty(t, cfg.TxMgrConfig.PrivateKey, "Missing private key for TxMgrConfig")
	require.NoError(t, cfg.Check(), "op-challenger config should be valid")

	errCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer close(errCh)
		errCh <- op_challenger.Main(ctx, log, cfg)
	}()
	return &Helper{
		log:    log,
		cancel: cancel,
		errors: errCh,
	}
}

func (h *Helper) Close() error {
	h.cancel()
	select {
	case <-time.After(1 * time.Minute):
		return errors.New("timed out while stopping challenger")
	case err := <-h.errors:
		if !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	}
}
