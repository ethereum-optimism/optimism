package crypto

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-service/testlog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// should be run with CGO_ENABLED=0

func TestSignerFactoryFromMnemonic(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"
	hdPath := "m/44'/60'/0'/0/1"
	testSigner(t, "", mnemonic, hdPath, signer.CLIConfig{})
}

func TestSignerFactoryFromKey(t *testing.T) {
	priv := "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
	testSigner(t, priv, "", "", signer.CLIConfig{})
}

func testSigner(t *testing.T, priv, mnemonic, hdPath string, cfg signer.CLIConfig) {
	logger := testlog.Logger(t, log.LevelDebug)

	factoryFn, addr, err := SignerFactoryFromConfig(logger, priv, mnemonic, hdPath, cfg)
	require.NoError(t, err)
	expectedAddr := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	require.Equal(t, expectedAddr, addr)
	chainID := big.NewInt(10)
	signerFn := factoryFn(chainID) // for chain ID 10
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     0,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		Gas:       21000,
		To:        nil,
		Value:     big.NewInt(0),
		Data:      []byte("test"),
	})
	signedTx, err := signerFn(context.Background(), addr, tx)
	require.NoError(t, err)
	gethSigner := types.LatestSignerForChainID(chainID)
	sender, err := gethSigner.Sender(signedTx)
	require.NoError(t, err)
	require.Equal(t, expectedAddr, sender)
}
