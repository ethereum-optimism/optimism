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
		WithZKSyncL1Confirmations(2),
	)
	require.NoError(t, err)
	require.Equal(t, 12*time.Second, *cfg.ProposalInterval)
	require.Equal(t, uint64(2), *cfg.SyncL1Confirmations)
}

func TestZKProposerConfigRejectsInvalidProposalInterval(t *testing.T) {
	_, err := newZKProposerConfig(WithZKProposalInterval(1500 * time.Millisecond))
	require.EqualError(t, err, "ZK proposer interval must use whole seconds")
}
