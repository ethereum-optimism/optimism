package preimages

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var errNotSupported = errors.New("not supported")

var _ PreimageUploader = (*LargePreimageUploader)(nil)

// LargePreimageUploader handles uploading large preimages by
// streaming the merkleized preimage to the PreimageOracle contract,
// tightly packed across multiple transactions.
type LargePreimageUploader struct {
	log log.Logger

	txMgr    txmgr.TxManager
	contract PreimageOracleContract
}

func NewLargePreimageUploader(logger log.Logger, txMgr txmgr.TxManager, contract PreimageOracleContract) *LargePreimageUploader {
	return &LargePreimageUploader{logger, txMgr, contract}
}

func (p *LargePreimageUploader) UploadPreimage(ctx context.Context, parent uint64, data *types.PreimageOracleData) error {
	// todo(proofs#467): generate the full preimage
	// todo(proofs#467): run the preimage through the keccak permutation, hashing
	//                   the intermediate state matrix after each block is applied.
	// todo(proofs#467): split up the preimage into chunks and submit the preimages
	//                   and state commitments to the preimage oracle contract using
	//                   `PreimageOracle.addLeavesLPP` (`_finalize` = false).

	// TODO(client-pod#473): The UUID must be deterministic so the challenger can resume uploads.
	uuid, err := p.newUUID()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}
	err = p.initLargePreimage(ctx, uuid, data.OracleOffset, uint32(len(data.OracleData)))
	if err != nil {
		return fmt.Errorf("failed to initialize large preimage with uuid: %s: %w", uuid, err)
	}

	// todo(proofs#467): track the challenge period starting once the full preimage is posted.
	// todo(proofs#467): once the challenge period is over, call `squeezeLPP` on the preimage oracle contract.

	return errNotSupported
}

func (p *LargePreimageUploader) newUUID() (*big.Int, error) {
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(max, big.NewInt(1))
	return rand.Int(rand.Reader, max)
}

// initLargePreimage initializes the large preimage proposal.
// This method *must* be called before adding any leaves.
func (p *LargePreimageUploader) initLargePreimage(ctx context.Context, uuid *big.Int, partOffset uint32, claimedSize uint32) error {
	candidate, err := p.contract.InitLargePreimage(uuid, partOffset, claimedSize)
	if err != nil {
		return fmt.Errorf("failed to create pre-image oracle tx: %w", err)
	}
	if err := p.sendTxAndWait(ctx, candidate); err != nil {
		return fmt.Errorf("failed to populate pre-image oracle: %w", err)
	}
	return nil
}

// sendTxAndWait sends a transaction through the [txmgr] and waits for a receipt.
// This sets the tx GasLimit to 0, performing gas estimation online through the [txmgr].
func (p *LargePreimageUploader) sendTxAndWait(ctx context.Context, candidate txmgr.TxCandidate) error {
	receipt, err := p.txMgr.Send(ctx, candidate)
	if err != nil {
		return err
	}
	if receipt.Status == ethtypes.ReceiptStatusFailed {
		p.log.Error("LargePreimageUploader tx successfully published but reverted", "tx_hash", receipt.TxHash)
	} else {
		p.log.Debug("LargePreimageUploader tx successfully published", "tx_hash", receipt.TxHash)
	}
	return nil
}
