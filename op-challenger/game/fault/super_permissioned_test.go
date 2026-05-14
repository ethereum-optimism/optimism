package fault

import (
	"context"
	"errors"
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var testGameL1Head = eth.BlockID{Number: 10}

func TestSuperPermissionedActorSkipsInvalidRoot(t *testing.T) {
	ctx := context.Background()
	contract := &stubSuperPermissionedContract{
		rootClaim: common.Hash{0xaa},
		sequence:  123,
	}
	sender := new(stubSuperPermissionedTxSender)
	actor := NewSuperPermissionedActor(log.New(), sender, contract, nil, nil, testGameL1Head)
	actor.expectedRootFn = func(context.Context, uint64) (common.Hash, bool, error) {
		return common.Hash{0xbb}, true, nil
	}

	require.NoError(t, actor.Act(ctx))

	require.Equal(t, 0, contract.callResolveCalls)
	require.Empty(t, sender.purposes)
}

func TestSuperPermissionedActorSkipsUnavailableRoot(t *testing.T) {
	ctx := context.Background()
	contract := &stubSuperPermissionedContract{
		rootClaim: common.Hash{0xaa},
		sequence:  123,
	}
	sender := new(stubSuperPermissionedTxSender)
	actor := NewSuperPermissionedActor(log.New(), sender, contract, nil, nil, testGameL1Head)
	actor.expectedRootFn = func(context.Context, uint64) (common.Hash, bool, error) {
		return common.Hash{}, false, nil
	}

	require.NoError(t, actor.Act(ctx))

	require.Equal(t, 0, contract.callResolveCalls)
	require.Empty(t, sender.purposes)
}

func TestSuperPermissionedActorWaitsWhenValidGameIsNotResolvable(t *testing.T) {
	ctx := context.Background()
	root := common.Hash{0xaa}
	contract := &stubSuperPermissionedContract{
		rootClaim:      root,
		sequence:       123,
		callResolveErr: errors.New("clock not expired"),
	}
	sender := new(stubSuperPermissionedTxSender)
	actor := NewSuperPermissionedActor(log.New(), sender, contract, nil, nil, testGameL1Head)
	actor.expectedRootFn = func(context.Context, uint64) (common.Hash, bool, error) {
		return root, true, nil
	}

	require.NoError(t, actor.Act(ctx))

	require.Equal(t, 1, contract.callResolveCalls)
	require.Empty(t, sender.purposes)
}

func TestSuperPermissionedActorResolvesValidResolvableGame(t *testing.T) {
	ctx := context.Background()
	root := common.Hash{0xaa}
	contract := &stubSuperPermissionedContract{
		rootClaim:         root,
		sequence:          123,
		callResolveStatus: gameTypes.GameStatusDefenderWon,
		resolveTx:         txmgr.TxCandidate{TxData: []byte{0x02}},
	}
	sender := new(stubSuperPermissionedTxSender)
	actor := NewSuperPermissionedActor(log.New(), sender, contract, nil, nil, testGameL1Head)
	actor.expectedRootFn = func(context.Context, uint64) (common.Hash, bool, error) {
		return root, true, nil
	}

	require.NoError(t, actor.Act(ctx))

	require.Equal(t, 1, contract.callResolveCalls)
	require.Equal(t, 1, contract.resolveTxCalls)
	require.Equal(t, []string{"resolve super permissioned game"}, sender.purposes)
	require.Equal(t, [][]byte{{0x02}}, sender.txData)
}

type stubSuperPermissionedContract struct {
	rootClaim         common.Hash
	sequence          uint64
	status            gameTypes.GameStatus
	callResolveStatus gameTypes.GameStatus
	callResolveErr    error
	resolveTx         txmgr.TxCandidate
	callResolveCalls  int
	resolveTxCalls    int
}

func (s *stubSuperPermissionedContract) GetL1Head(context.Context) (common.Hash, error) {
	return common.Hash{}, nil
}

func (s *stubSuperPermissionedContract) GetStatus(context.Context) (gameTypes.GameStatus, error) {
	return s.status, nil
}

func (s *stubSuperPermissionedContract) GetRootClaim(context.Context) (common.Hash, error) {
	return s.rootClaim, nil
}

func (s *stubSuperPermissionedContract) GetL2SequenceNumber(context.Context) (uint64, error) {
	return s.sequence, nil
}

func (s *stubSuperPermissionedContract) CallResolve(context.Context) (gameTypes.GameStatus, error) {
	s.callResolveCalls++
	return s.callResolveStatus, s.callResolveErr
}

func (s *stubSuperPermissionedContract) ResolveTx() (txmgr.TxCandidate, error) {
	s.resolveTxCalls++
	return s.resolveTx, nil
}

type stubSuperPermissionedTxSender struct {
	purposes []string
	txData   [][]byte
}

func (s *stubSuperPermissionedTxSender) From() common.Address {
	return common.Address{}
}

func (s *stubSuperPermissionedTxSender) SendAndWaitSimple(txPurpose string, txs ...txmgr.TxCandidate) error {
	s.purposes = append(s.purposes, txPurpose)
	for _, tx := range txs {
		s.txData = append(s.txData, tx.TxData)
	}
	return nil
}
