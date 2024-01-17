package preimages

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/log"
)

var _ PreimageUploader = (*LargePreimageUploader)(nil)

var errNotSupported = errors.New("not supported")

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
	// todo(proofs#467): track the challenge period starting once the full preimage is posted.
	// todo(proofs#467): once the challenge period is over, call `squeezeLPP` on the preimage oracle contract.
	return errNotSupported
}
