package keccak

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/fetcher"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type Oracle interface {
	fetcher.Oracle
	ChallengeTx(ident keccakTypes.LargePreimageIdent, challenge keccakTypes.Challenge) (txmgr.TxCandidate, error)
}

type Verifier interface {
	CreateChallenge(ctx context.Context, blockHash common.Hash, oracle fetcher.Oracle, preimage keccakTypes.LargePreimageMetaData) (keccakTypes.Challenge, error)
}

type Sender interface {
	SendAndWait(txPurpose string, txs ...txmgr.TxCandidate) ([]*types.Receipt, error)
}

type PreimageChallenger struct {
	log      log.Logger
	verifier Verifier
	sender   Sender
}

func NewPreimageChallenger(logger log.Logger, verifier Verifier, sender Sender) *PreimageChallenger {
	return &PreimageChallenger{
		log:      logger,
		verifier: verifier,
		sender:   sender,
	}
}

func (c *PreimageChallenger) Challenge(ctx context.Context, blockHash common.Hash, oracle Oracle, preimages []keccakTypes.LargePreimageMetaData) error {
	var txLock sync.Mutex
	var wg sync.WaitGroup
	var txs []txmgr.TxCandidate
	for _, preimage := range preimages {
		preimage := preimage
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger := c.log.New("oracle", oracle.Addr(), "claimant", preimage.Claimant, "uuid", preimage.UUID)
			challenge, err := c.verifier.CreateChallenge(ctx, blockHash, oracle, preimage)
			if errors.Is(err, matrix.ErrValid) {
				logger.Debug("Preimage is valid")
				return
			} else if err != nil {
				logger.Error("Failed to verify large preimage", "err", err)
				return
			}
			tx, err := oracle.ChallengeTx(preimage.LargePreimageIdent, challenge)
			if err != nil {
				logger.Error("Failed to create challenge transaction", "err", err)
				return
			}
			txLock.Lock()
			defer txLock.Unlock()
			txs = append(txs, tx)
		}()
	}
	wg.Wait()
	if len(txs) > 0 {
		_, err := c.sender.SendAndWait("challenge preimages", txs...)
		if err != nil {
			return fmt.Errorf("failed to send challenge txs: %w", err)
		}
	}
	return nil
}
