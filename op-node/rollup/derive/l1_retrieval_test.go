package derive

import (
	"context"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockDataSource struct {
	mock.Mock
}

func (m *MockDataSource) OpenData(ctx context.Context, id eth.BlockID) (DataIter, error) {
	out := m.Mock.MethodCalled("OpenData", id)
	return out[0].(DataIter), *out[1].(*error)
}

func (m *MockDataSource) ExpectOpenData(id eth.BlockID, iter DataIter, err error) {
	m.Mock.On("OpenData", id).Return(iter, &err)
}

var _ DataAvailabilitySource = (*MockDataSource)(nil)

type MockIngestData struct {
	MockOriginStage
}

func (im *MockIngestData) IngestData(data []byte) error {
	out := im.Mock.MethodCalled("IngestData", data)
	return *out[0].(*error)
}

func (im *MockIngestData) ExpectIngestData(data []byte, err error) {
	im.Mock.On("IngestData", data).Return(&err)
}

var _ L1SourceOutput = (*MockIngestData)(nil)

func TestL1Retrieval_Step(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))

	next := &MockIngestData{MockOriginStage{progress: Progress{Origin: testutils.RandomBlockRef(rng), Closed: true}}}
	dataSrc := &MockDataSource{}

	a := testutils.RandomData(rng, 10)
	b := testutils.RandomData(rng, 15)
	iter := &DataSlice{a, b}

	outer := Progress{Origin: testutils.NextRandomRef(rng, next.progress.Origin), Closed: false}

	// mock some L1 data to open for the origin that is opened by the outer stage
	dataSrc.ExpectOpenData(outer.Origin.ID(), iter, nil)

	next.ExpectIngestData(a, nil)
	next.ExpectIngestData(b, nil)

	defer dataSrc.AssertExpectations(t)
	defer next.AssertExpectations(t)

	l1r := NewL1Retrieval(testlog.Logger(t, log.LvlError), dataSrc, next)

	// first we expect the stage to reset to the origin of the inner stage
	require.NoError(t, RepeatResetStep(t, l1r.ResetStep, nil, 1))
	require.Equal(t, next.Progress(), l1r.Progress(), "stage needs to adopt the progress of next stage on reset")

	// and then start processing the data of the next stage
	require.NoError(t, RepeatStep(t, l1r.Step, outer, 10))
}
