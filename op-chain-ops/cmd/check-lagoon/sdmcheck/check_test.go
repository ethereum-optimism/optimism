package sdmcheck

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestConfigApplyDefaults(t *testing.T) {
	var cfg Config
	cfg.ApplyDefaults()
	require.Equal(t, 12, cfg.BatchSize)
	require.Equal(t, uint64(20), cfg.SlotCount)
	require.Equal(t, 2, cfg.MinUserTxs)
	require.Equal(t, 3, cfg.Attempts)
	require.Equal(t, uint64(1_000_000), cfg.GasLimit)
	require.Equal(t, uint64(2_000_000), cfg.DeployLimit)
	require.Equal(t, uint64(200_000), cfg.FundGasLimit)
	require.Equal(t, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), cfg.FundAmount)
}

func TestDeriveSDMAccountKey(t *testing.T) {
	funder, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)
	first, err := DeriveSDMAccountKey(funder)
	require.NoError(t, err)
	second, err := DeriveSDMAccountKey(funder)
	require.NoError(t, err)
	require.Equal(t, crypto.FromECDSA(first), crypto.FromECDSA(second))
	require.NotEqual(t, crypto.PubkeyToAddress(funder.PublicKey), crypto.PubkeyToAddress(first.PublicKey))
}

func TestFundingDeficit(t *testing.T) {
	minimum := big.NewInt(20)
	require.Equal(t, big.NewInt(20), fundingDeficit(big.NewInt(0), minimum))
	require.Equal(t, big.NewInt(5), fundingDeficit(big.NewInt(15), minimum))
	require.Zero(t, fundingDeficit(big.NewInt(20), minimum).Sign())
	require.Zero(t, fundingDeficit(big.NewInt(25), minimum).Sign())
	// The helper must not mutate caller-owned amounts.
	require.Equal(t, big.NewInt(20), minimum)
}

func TestValidateConfig(t *testing.T) {
	key := new(ecdsa.PrivateKey)
	valid := Config{RPCURL: "http://l2", Key: key, BatchSize: 12, MinUserTxs: 2, Attempts: 3, FundAmount: big.NewInt(1)}
	require.NoError(t, validateConfig(valid))

	invalid := valid
	invalid.MinUserTxs = 13
	require.ErrorContains(t, validateConfig(invalid), "exceeds batch size")

	invalid = valid
	invalid.FundL2 = true
	require.ErrorContains(t, validateConfig(invalid), "requires the L1 RPC")

	invalid.L1RPCURL = "http://l1"
	invalid.L1Key = key
	invalid.Portal = common.HexToAddress("0x1234")
	require.NoError(t, validateConfig(invalid))
}
