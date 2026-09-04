package sysgo

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestLoadZKProgramVKey(t *testing.T) {
	t.Run("uses deterministic stub without built ELFs", func(t *testing.T) {
		vkey, err := loadZKProgramVKey("")
		require.NoError(t, err)
		require.Equal(t, crypto.Keccak256Hash([]byte("kona-sp1-stub-super-aggregation-vkey")), vkey)
	})

	t.Run("loads super aggregation vkey from ELF directory", func(t *testing.T) {
		elfDir := t.TempDir()
		want := common.HexToHash("0x1234")
		err := os.WriteFile(
			filepath.Join(elfDir, "vkeys.toml"),
			[]byte(`super-aggregation = "`+want.Hex()+`"`),
			0o600,
		)
		require.NoError(t, err)

		vkey, err := loadZKProgramVKey(elfDir)
		require.NoError(t, err)
		require.Equal(t, want, vkey)
	})

	t.Run("rejects missing super aggregation vkey", func(t *testing.T) {
		elfDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(elfDir, "vkeys.toml"), []byte(""), 0o600))

		_, err := loadZKProgramVKey(elfDir)
		require.EqualError(t, err, "vkeys.toml does not contain super-aggregation")
	})
}

func TestZKProposerOptions(t *testing.T) {
	cfg, err := newZKProposerConfig(
		WithZKProposalInterval(12*time.Second),
		WithZKFastFinality(),
		WithZKMetrics(),
	)
	require.NoError(t, err)
	require.Equal(t, 12*time.Second, *cfg.ProposalInterval)
	require.True(t, cfg.FastFinality)
	require.True(t, cfg.Metrics)
}

func TestZKProposerDisablesMetricsByDefault(t *testing.T) {
	cfg, err := newZKProposerConfig()
	require.NoError(t, err)
	require.False(t, cfg.Metrics)
}

func TestZKProposerConfigRejectsInvalidProposalInterval(t *testing.T) {
	_, err := newZKProposerConfig(WithZKProposalInterval(1500 * time.Millisecond))
	require.EqualError(t, err, "ZK proposer interval must use whole seconds")
}

func TestZKProposerRuntimeLifecycle(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		runtime := &MultiChainRuntime{}

		_, err := runtime.zkProposerRuntime()
		require.EqualError(t, err, "ZK proposer is not configured")
		_, err = runtime.startZKProposerRuntime()
		require.EqualError(t, err, "ZK proposer is not configured")
	})

	t.Run("configured for delayed start", func(t *testing.T) {
		handle := &ZKProposerRuntime{}
		starts := 0
		runtime := &MultiChainRuntime{
			startZKProposerFn: func() *ZKProposerRuntime {
				starts++
				return handle
			},
		}

		_, err := runtime.zkProposerRuntime()
		require.EqualError(t, err, "ZK proposer is configured but not started; call StartZKProposer")

		started, err := runtime.startZKProposerRuntime()
		require.NoError(t, err)
		require.Same(t, handle, started)
		require.Equal(t, 1, starts)

		running, err := runtime.zkProposerRuntime()
		require.NoError(t, err)
		require.Same(t, handle, running)

		_, err = runtime.startZKProposerRuntime()
		require.EqualError(t, err, "ZK proposer is already started")
		require.Equal(t, 1, starts)
	})

	t.Run("automatically started", func(t *testing.T) {
		handle := &ZKProposerRuntime{}
		runtime := &MultiChainRuntime{zkProposer: handle}

		running, err := runtime.zkProposerRuntime()
		require.NoError(t, err)
		require.Same(t, handle, running)
	})
}
