package keccak

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/fetcher"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestChallenge(t *testing.T) {
	preimages := []keccakTypes.LargePreimageMetaData{
		{
			LargePreimageIdent: keccakTypes.LargePreimageIdent{
				Claimant: common.Address{0xff, 0x00},
				UUID:     big.NewInt(0),
			},
		},
		{
			LargePreimageIdent: keccakTypes.LargePreimageIdent{
				Claimant: common.Address{0xff, 0x01},
				UUID:     big.NewInt(1),
			},
		},
		{
			LargePreimageIdent: keccakTypes.LargePreimageIdent{
				Claimant: common.Address{0xff, 0x02},
				UUID:     big.NewInt(2),
			},
		},
	}

	verifier := &stubVerifier{
		challenges: map[keccakTypes.LargePreimageIdent]keccakTypes.Challenge{
			preimages[1].LargePreimageIdent: {StateMatrix: []byte{0x01}},
			preimages[2].LargePreimageIdent: {StateMatrix: []byte{0x02}},
		},
	}
	sender := &stubSender{}
	oracle := &stubChallengerOracle{}
	challenger := NewPreimageChallenger(testlog.Logger(t, log.LvlInfo), verifier, sender)
	err := challenger.Challenge(context.Background(), common.Hash{0xaa}, oracle, preimages)
	require.NoError(t, err)

	// Should send the two challenges before returning
	require.Len(t, sender.sent, 1, "Should send a single batch of transactions")
	for ident, challenge := range verifier.challenges {
		tx, err := oracle.ChallengeTx(ident, challenge)
		require.NoError(t, err)
		require.Contains(t, sender.sent[0], tx)
	}
}

type stubVerifier struct {
	challenges map[keccakTypes.LargePreimageIdent]keccakTypes.Challenge
}

func (s *stubVerifier) CreateChallenge(_ context.Context, _ common.Hash, _ fetcher.Oracle, preimage keccakTypes.LargePreimageMetaData) (keccakTypes.Challenge, error) {
	challenge, ok := s.challenges[preimage.LargePreimageIdent]
	if !ok {
		return keccakTypes.Challenge{}, matrix.ErrValid
	}
	return challenge, nil
}

type stubSender struct {
	sent [][]txmgr.TxCandidate
}

func (s *stubSender) SendAndWait(_ string, txs ...txmgr.TxCandidate) ([]*types.Receipt, error) {
	s.sent = append(s.sent, txs)
	return nil, nil
}

type stubChallengerOracle struct {
	stubOracle
}

func (s *stubChallengerOracle) ChallengeTx(ident keccakTypes.LargePreimageIdent, challenge keccakTypes.Challenge) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{
		To:     &ident.Claimant,
		TxData: append(ident.UUID.Bytes(), challenge.StateMatrix...),
	}, nil
}
