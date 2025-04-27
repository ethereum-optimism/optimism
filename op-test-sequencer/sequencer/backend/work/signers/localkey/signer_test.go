package localkey

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

func TestSigner(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey)
	chainID := eth.ChainIDFromUInt64(123)
	id := seqtypes.SignerID("foobar")

	signer := NewSigner(id, logger, chainID, key)
	testSigner(t, signer, chainID, addr)
}
