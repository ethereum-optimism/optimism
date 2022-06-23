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

func TestL1Source_Step(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))

	next := &MockIngestData{MockOriginStage{originOpen: false, currentOrigin: testutils.RandomBlockRef(rng)}}
	dataSrc := &MockDataSource{}

	a := testutils.RandomData(rng, 10)
	b := testutils.RandomData(rng, 15)
	iter := &DataSlice{a, b}

	ref := testutils.RandomBlockRef(rng)

	// origin of the next stage
	dataSrc.ExpectOpenData(ref.ID(), iter, nil)

	next.ExpectOpenOrigin(ref, nil)
	next.ExpectIngestData(a, nil)
	next.ExpectIngestData(b, nil)
	next.ExpectCloseOrigin()

	defer dataSrc.AssertExpectations(t)
	defer next.AssertExpectations(t)

	l1Src := NewL1Source(testlog.Logger(t, log.LvlError), dataSrc, next)

	require.NoError(t, RepeatStep(t, l1Src.Step, 10))
}

func TestL1Source_ResetStep(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	ref := testutils.RandomBlockRef(rng)

	next := &MockIngestData{MockOriginStage{originOpen: true, currentOrigin: ref}}
	dataSrc := &MockDataSource{}

	l1Src := NewL1Source(testlog.Logger(t, log.LvlError), dataSrc, next)
	require.NoError(t, RepeatResetStep(t, l1Src.ResetStep, nil, 1))
	require.Equal(t, ref, l1Src.CurrentOrigin(), "stage needs to adopt the origin of next stage on reset")
}
