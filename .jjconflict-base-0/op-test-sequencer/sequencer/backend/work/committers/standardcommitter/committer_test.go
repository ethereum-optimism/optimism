package standardcommitter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type testAPI struct {
	v   *opsigner.SignedExecutionPayloadEnvelope
	err error
}

func (t *testAPI) CommitBlock(ctx context.Context, envelope *opsigner.SignedExecutionPayloadEnvelope) error {
	t.v = envelope
	return t.err
}

type dummySignedBlock struct {
}

func (s *dummySignedBlock) ID() eth.BlockID {
	return eth.BlockID{Number: 1000}
}

func (s *dummySignedBlock) String() string {
	return "test signed block"
}

func (s *dummySignedBlock) VerifySignature(authContext opsigner.Authenticator) error {
	return errors.New("not supported")
}

var _ work.SignedBlock = (*dummySignedBlock)(nil)

var _ apis.CommitAPI = (*testAPI)(nil)

func TestStandardCommitter(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	id := seqtypes.CommitterID("foo")
	api := &testAPI{}
	x := NewCommitter(id, logger, api)

	require.ErrorIs(t, x.Commit(context.Background(), &dummySignedBlock{}), seqtypes.ErrUnknownKind)

	signed := &opsigner.SignedExecutionPayloadEnvelope{
		Envelope: &eth.ExecutionPayloadEnvelope{
			ParentBeaconBlockRoot: nil,
			ExecutionPayload: &eth.ExecutionPayload{
				BlockHash: common.Hash{123},
			},
		},
		Signature: eth.Bytes65{42},
	}
	err := x.Commit(context.Background(), signed)
	require.NoError(t, err)
	require.Equal(t, signed, api.v)

	api.v = nil
	api.err = errors.New("test err")
	err = x.Commit(context.Background(), signed)
	require.ErrorIs(t, err, api.err)

	require.NoError(t, x.Close())
	require.Equal(t, "standard-committer-foo", x.String())
	require.Equal(t, id, x.ID())
}
