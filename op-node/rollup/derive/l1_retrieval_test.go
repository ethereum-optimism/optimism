package derive

import (
	"context"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type fakeDataIter struct {
	idx  int
	data []eth.Data
	errs []error
}

func (cs *fakeDataIter) Next(ctx context.Context) (eth.Data, error) {
	i := cs.idx
	cs.idx += 1
	return cs.data[i], cs.errs[i]
}

type MockDataSource struct {
	mock.Mock
}

func (m *MockDataSource) OpenData(ctx context.Context, id eth.BlockID, batcherAddr common.Address) DataIter {
	out := m.Mock.MethodCalled("OpenData", id, batcherAddr)
	return out[0].(DataIter)
}

func (m *MockDataSource) ExpectOpenData(id eth.BlockID, iter DataIter, batcherAddr common.Address) {
	m.Mock.On("OpenData", id, batcherAddr).Return(iter)
}

var _ DataAvailabilitySource = (*MockDataSource)(nil)

type MockL1Traversal struct {
	mock.Mock
}

func (m *MockL1Traversal) ExpectSystemConfig(sysCfg eth.SystemConfig) {
	m.Mock.On("SystemConfig").Return(sysCfg)
}

func (m *MockL1Traversal) SystemConfig() eth.SystemConfig {
	out := m.Mock.MethodCalled("SystemConfig")
	return out[0].(eth.SystemConfig)
}

func (m *MockL1Traversal) Origin() eth.L1BlockRef {
	out := m.Mock.MethodCalled("Origin")
	return out[0].(eth.L1BlockRef)
}

func (m *MockL1Traversal) ExpectOrigin(block eth.L1BlockRef) {
	m.Mock.On("Origin").Return(block)
}

func (m *MockL1Traversal) NextL1Block(_ context.Context) (eth.L1BlockRef, error) {
	out := m.Mock.MethodCalled("NextL1Block")
	return out[0].(eth.L1BlockRef), *out[1].(*error)
}

func (m *MockL1Traversal) ExpectNextL1Block(block eth.L1BlockRef, err error) {
	m.Mock.On("NextL1Block").Return(block, &err)
}

var _ NextBlockProvider = (*MockL1Traversal)(nil)

// TestL1RetrievalReset tests the reset. The reset just opens up a new
// data for the specified block.
func TestL1RetrievalReset(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	dataSrc := &MockDataSource{}
	a := testutils.RandomBlockRef(rng)
	l1Cfg := eth.SystemConfig{
		BatcherAddr: common.Address{42},
	}

	dataSrc.ExpectOpenData(a.ID(), &fakeDataIter{}, l1Cfg.BatcherAddr)
	defer dataSrc.AssertExpectations(t)

	l1r := NewL1Retrieval(testlog.Logger(t, log.LvlError), dataSrc, nil)

	// We assert that it opens up the correct data on a reset
	_ = l1r.Reset(context.Background(), a, l1Cfg)
}

// TestL1RetrievalNextData tests that the `NextData` function properly
// handles different error cases and returns the expected data
// if there is no error.
func TestL1RetrievalNextData(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	a := testutils.RandomBlockRef(rng)

	tests := []struct {
		name         string
		prevBlock    eth.L1BlockRef
		sysCfg       eth.SystemConfig
		prevErr      error // error returned by prev.NextL1Block
		openErr      error // error returned by NextData if prev.NextL1Block fails
		datas        []eth.Data
		datasErrs    []error
		expectedErrs []error
	}{
		{
			name:         "simple retrieval",
			prevBlock:    a,
			sysCfg:       eth.SystemConfig{BatcherAddr: common.Address{0x55}},
			prevErr:      nil,
			openErr:      nil,
			datas:        []eth.Data{testutils.RandomData(rng, 10), testutils.RandomData(rng, 10), testutils.RandomData(rng, 10), nil},
			datasErrs:    []error{nil, nil, nil, io.EOF},
			expectedErrs: []error{nil, nil, nil, io.EOF},
		},
		{
			name:    "out of data",
			prevErr: io.EOF,
			openErr: io.EOF,
		},
		{
			name:         "fail to open data",
			prevBlock:    a,
			sysCfg:       eth.SystemConfig{BatcherAddr: common.Address{0x55}},
			prevErr:      nil,
			openErr:      nil,
			datas:        []eth.Data{nil},
			datasErrs:    []error{NewCriticalError(ethereum.NotFound)},
			expectedErrs: []error{ErrCritical},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			l1t := &MockL1Traversal{}
			l1t.ExpectNextL1Block(test.prevBlock, test.prevErr)
			dataSrc := &MockDataSource{}
			dataSrc.ExpectOpenData(test.prevBlock.ID(), &fakeDataIter{data: test.datas, errs: test.datasErrs}, test.sysCfg.BatcherAddr)

			ret := NewL1Retrieval(testlog.Logger(t, log.LvlCrit), dataSrc, l1t)

			// If prevErr != nil we forced an error while getting data from the previous stage
			if test.openErr != nil {
				data, err := ret.NextData(context.Background())
				require.Nil(t, data)
				require.ErrorIs(t, err, test.openErr)
			}

			// Go through the fake data an assert that data is passed through and the correct
			// errors are returned.
			for i := range test.expectedErrs {
				l1t.ExpectSystemConfig(test.sysCfg)
				data, err := ret.NextData(context.Background())
				require.Equal(t, test.datas[i], hexutil.Bytes(data))
				require.ErrorIs(t, err, test.expectedErrs[i])
			}

			l1t.AssertExpectations(t)
		})
	}

}
