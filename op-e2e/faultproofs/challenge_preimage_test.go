package faultproofs

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame/preimage"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	preimage2 "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/stretchr/testify/require"
)

func TestChallengeLargePreimages_ChallengeFirst(t *testing.T) {
	op_e2e.InitParallel(t)
	ctx := context.Background()
	sys, _ := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	disputeGameFactory.StartChallenger(ctx, "Challenger",
		challenger.WithAlphabet(sys.RollupEndpoint("sequencer")),
		challenger.WithPrivKey(sys.Cfg.Secrets.Alice))
	preimageHelper := disputeGameFactory.PreimageHelper(ctx)
	ident := preimageHelper.UploadLargePreimage(ctx, preimage.MinPreimageSize,
		preimage.WithReplacedCommitment(0, common.Hash{0xaa}))

	require.NotEqual(t, ident.Claimant, common.Address{})

	preimageHelper.WaitForChallenged(ctx, ident)
}

func TestChallengeLargePreimages_ChallengeMiddle(t *testing.T) {
	op_e2e.InitParallel(t)
	ctx := context.Background()
	sys, _ := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)
	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	disputeGameFactory.StartChallenger(ctx, "Challenger",
		challenger.WithAlphabet(sys.RollupEndpoint("sequencer")),
		challenger.WithPrivKey(sys.Cfg.Secrets.Mallory))
	preimageHelper := disputeGameFactory.PreimageHelper(ctx)
	ident := preimageHelper.UploadLargePreimage(ctx, preimage.MinPreimageSize,
		preimage.WithReplacedCommitment(10, common.Hash{0xaa}))

	require.NotEqual(t, ident.Claimant, common.Address{})

	preimageHelper.WaitForChallenged(ctx, ident)
}

func TestChallengeLargePreimages_ChallengeLast(t *testing.T) {
	op_e2e.InitParallel(t)
	ctx := context.Background()
	sys, _ := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)
	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	disputeGameFactory.StartChallenger(ctx, "Challenger",
		challenger.WithAlphabet(sys.RollupEndpoint("sequencer")),
		challenger.WithPrivKey(sys.Cfg.Secrets.Mallory))
	preimageHelper := disputeGameFactory.PreimageHelper(ctx)
	ident := preimageHelper.UploadLargePreimage(ctx, preimage.MinPreimageSize,
		preimage.WithLastCommitment(common.Hash{0xaa}))

	require.NotEqual(t, ident.Claimant, common.Address{})

	preimageHelper.WaitForChallenged(ctx, ident)
}

func TestUploadBlobPreimage(t *testing.T) {
	// Create some blob data to upload
	blob := testBlob()
	commitment, err := kzg4844.BlobToCommitment(kzg4844.Blob(blob))
	require.NoError(t, err)

	fieldIndex := uint64(24)
	elementData := blob[fieldIndex<<5 : (fieldIndex+1)<<5]
	//kzgProof, claim, err := kzg4844.ComputeProof(kzg4844.Blob(blob), kzg4844.Point(elementData))
	//require.NoError(t, err)
	blobValue := make([]byte, len(elementData)+8)
	binary.BigEndian.PutUint64(blobValue[:8], uint64(len(elementData)))
	copy(blobValue[8:], elementData[:])

	keyBuf := make([]byte, 80)
	copy(keyBuf[:48], commitment[:])
	binary.BigEndian.PutUint64(keyBuf[72:], fieldIndex)
	key := preimage2.BlobKey(crypto.Keccak256Hash(keyBuf)).PreimageKey()

	blobData, err := cannon.LoadBlobPreimageFromParts(key[:], blobValue, 0, eth.Blob(blob), commitment[:], fieldIndex)
	require.NoError(t, err)

	// Start the system
	op_e2e.InitParallel(t)
	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)
	caller := batching.NewMultiCaller(l1Client.Client(), 100)
	factory, err := contracts.NewDisputeGameFactoryContract(sys.Cfg.L1Deployments.DisputeGameFactoryProxy, caller)
	require.NoError(t, err)
	gameAddr, err := factory.GetGameImpl(ctx, 0)
	require.NoError(t, err)
	gameContract, err := contracts.NewFaultDisputeGameContract(gameAddr, caller)
	require.NoError(t, err)
	oracle, err := gameContract.GetOracle(ctx)
	require.NoError(t, err)

	// Upload the data, first try just with a call to see if it reverts
	err = oracle.CallAddGlobalData(ctx, blobData)
	require.NoError(t, err)

	// Then actually send it as a tx
	tx, err := oracle.AddGlobalDataTx(blobData)
	require.NoError(t, err)
	_, rcpt, err := transactions.SendTx(ctx, sys.Cfg.Secrets.Mallory, tx, l1Client)
	require.NoError(t, err)

	// Check the preimage it uploaded is what we expected
	require.Len(t, rcpt.Logs, 1)
	addedKey, err := oracle.ParseLog(rcpt.Logs[0])
	require.NoError(t, err)
	require.EqualValues(t, common.Hash(blobData.OracleKey), addedKey)

	// Check its now available in the oracle
	exists, err := oracle.GlobalDataExists(ctx, blobData)
	require.NoError(t, err)
	require.True(t, exists, "Should have uploaded blob data to preimage oracle")
}

// Returns a serialized random field element in big-endian
func fieldElement(val uint64) [32]byte {
	var r fr.Element
	_, _ = r.SetRandom()
	return gokzg4844.SerializeScalar(r)
}

func testBlob() gokzg4844.Blob {
	var blob gokzg4844.Blob
	bytesPerBlob := gokzg4844.ScalarsPerBlob * gokzg4844.SerializedScalarSize
	for i := 0; i < bytesPerBlob; i += gokzg4844.SerializedScalarSize {
		fieldElementBytes := fieldElement(uint64(i))
		copy(blob[i:i+gokzg4844.SerializedScalarSize], fieldElementBytes[:])
	}
	return blob
}
