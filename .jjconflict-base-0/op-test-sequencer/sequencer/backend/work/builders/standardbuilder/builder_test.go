package standardbuilder

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/metrics"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type testAPI struct {
	parent eth.BlockID
	attrs  *eth.PayloadAttributes
	info   eth.PayloadInfo
	v      *eth.ExecutionPayloadEnvelope
}

func (t *testAPI) OpenBlock(ctx context.Context, parent eth.BlockID, attrs *eth.PayloadAttributes) (eth.PayloadInfo, error) {
	t.parent = parent
	t.attrs = attrs
	return t.info, nil
}

func (t *testAPI) CancelBlock(ctx context.Context, id eth.PayloadInfo) error {
	t.info = eth.PayloadInfo{}
	return nil
}

func (t *testAPI) SealBlock(ctx context.Context, id eth.PayloadInfo) (*eth.ExecutionPayloadEnvelope, error) {
	if t.info != id {
		return nil, &rpc.JsonError{Code: apis.BuildErrCodeUnknownPayload, Message: "unknown payload"}
	}
	return t.v, nil
}

var _ apis.BuildAPI = (*testAPI)(nil)

type fakeAttributesBuilder struct {
}

func (f *fakeAttributesBuilder) PreparePayloadAttributes(ctx context.Context, l2Parent eth.L2BlockRef, epoch eth.BlockID) (attrs *eth.PayloadAttributes, err error) {
	return &eth.PayloadAttributes{
		Timestamp: eth.Uint64Quantity(l2Parent.Time) + 2,
	}, nil
}

var _ derive.AttributesBuilder = (*fakeAttributesBuilder)(nil)

func TestStandardBuilder(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	id := seqtypes.BuilderID("foo")
	api := &testAPI{}
	m := &metrics.NoopMetrics{}
	l1 := &testutils.MockL1Source{}
	l2 := &testutils.MockL2Client{}
	fb := &fakeAttributesBuilder{}
	reg := work.NewJobRegistry()

	x := NewBuilder(id, logger, m, l1, l2, fb, api, reg)

	ctx := context.Background()
	rng := rand.New(rand.NewSource(123))

	l2Parent := testutils.RandomL2BlockRef(rng)
	l1Origin := testutils.RandomBlockRef(rng)
	opts := &seqtypes.BuildOpts{
		Parent:   l2Parent.Hash,
		L1Origin: &l1Origin.Hash,
	}
	l1.ExpectL1BlockRefByHash(l1Origin.Hash, l1Origin, nil)
	l2.ExpectL2BlockRefByHash(l2Parent.Hash, l2Parent, nil)
	job, err := x.NewJob(ctx, *opts)
	require.NoError(t, err)
	require.NotNil(t, reg.GetJob(job.ID()), "job should be there now")

	err = job.Open(ctx)
	require.NoError(t, err)

	require.Equal(t, l2Parent.Time+2, uint64(api.attrs.Timestamp), "must be building the right block")
	l1.AssertExpectations(t)
	l2.AssertExpectations(t)

	// prepare block-sealing result
	api.v = &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		BlockHash:   testutils.RandomHash(rng),
		BlockNumber: eth.Uint64Quantity(l2Parent.Number + 1),
		Timestamp:   api.attrs.Timestamp,
	}}

	result, err := job.Seal(ctx)
	require.NoError(t, err)
	require.Equal(t, api.v.ID(), result.ID())
	require.NotNil(t, reg.GetJob(job.ID()), "job is not removed upon sealing but upon closing")

	job.Close()
	require.Nil(t, reg.GetJob(job.ID()), "job should be cleaned up")

	l1.ExpectL1BlockRefByHash(l1Origin.Hash, l1Origin, nil)
	l2.ExpectL2BlockRefByHash(l2Parent.Hash, l2Parent, nil)
	job2, err := x.NewJob(ctx, *opts)
	require.NoError(t, err)
	l1.AssertExpectations(t)
	l2.AssertExpectations(t)
	require.NotNil(t, reg.GetJob(job2.ID()), "job 2 should be there now")

	require.NoError(t, job2.Cancel(ctx))
	require.NotNil(t, reg.GetJob(job2.ID()), "job is not removed upon canceling but upon closing")
	job2.Close()
	require.Nil(t, reg.GetJob(job2.ID()), "job 2 should be cleaned up")
}
