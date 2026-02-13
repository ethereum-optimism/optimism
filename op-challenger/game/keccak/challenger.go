package keccak

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type Oracle interface {
	VerifierPreimageOracle
	ChallengeTx(ident keccakTypes.LargePreimageIdent, challenge keccakTypes.Challenge) (txmgr.TxCandidate, error)
}

type ChallengeMetrics interface {
	RecordPreimageChallenged()
	RecordPreimageChallengeFailed()
}

type Verifier interface {
	CreateChallenge(ctx context.Context, blockHash common.Hash, oracle VerifierPreimageOracle, preimage keccakTypes.LargePreimageMetaData) (keccakTypes.Challenge, error)
}

type Sender interface {
	SendAndWaitSimple(txPurpose string, txs ...txmgr.TxCandidate) error
}

type PreimageChallenger struct {
	log      log.Logger
	metrics  ChallengeMetrics
	verifier Verifier
	sender   Sender
}

func NewPreimageChallenger(logger log.Logger, metrics ChallengeMetrics, verifier Verifier, sender Sender) *PreimageChallenger {
	return &PreimageChallenger{
		log:      logger,
		metrics:  metrics,
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
			logger.Info("Challenging preimage", "block", challenge.Poststate.Index)
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
	c.log.Debug("Created preimage challenge transactions", "count", len(txs))
	if len(txs) > 0 {
		err := c.sender.SendAndWaitSimple("challenge preimages", txs...)
		if err != nil {
			c.metrics.RecordPreimageChallengeFailed()
			return fmt.Errorf("failed to send challenge txs: %w", err)
		}
		c.metrics.RecordPreimageChallenged()
	}
	return nil
}
