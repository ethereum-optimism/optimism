package e2e_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestPostieStorageRootUpdates(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	// wait for the first storage root of chain B to change
	var oldStorageRoot common.Hash
	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		oldStorageRoot = testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
		return oldStorageRoot != common.Hash{}, nil
	}))

	// initiate an message on chain B
	// NOTE: the destination chain does not matter for now as postie will update for any change
	outbox, err := bindings.NewCrossL2Outbox(predeploys.CrossL2OutboxAddr, testSuite.OpSysB.Clients["sequencer"])
	require.NoError(t, err)

	sender, senderAddr := testSuite.OpCfg.Secrets.Bob, testSuite.OpCfg.Secrets.Addresses().Bob
	senderOpts, err := bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdB)))
	require.NoError(t, err)
	senderOpts.Value = big.NewInt(params.Ether / 2)

	tx, err := outbox.InitiateMessage(senderOpts, common.BigToHash(big.NewInt(int64(testSuite.ChainIdA))), senderAddr, big.NewInt(25_000), []byte{})
	require.NoError(t, err)

	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)

	// wait for a changed root
	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		return testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB) != oldStorageRoot, nil
	}))
}
