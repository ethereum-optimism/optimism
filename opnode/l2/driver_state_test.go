package l2

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/internal/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testID string

func (id testID) ID() eth.BlockID {
	parts := strings.Split(string(id), ":")
	if len(parts) != 2 {
		panic("bad id")
	}
	if len(parts[0]) > 32 {
		panic("test ID hash too long")
	}
	var h common.Hash
	copy(h[:], parts[0])
	v, err := strconv.ParseUint(parts[1], 0, 64)
	if err != nil {
		panic(err)
	}
	return eth.BlockID{
		Hash:   h,
		Number: v,
	}
}

type testState struct {
	l1Head      testID
	l2Head      testID
	l2Finalized testID
	l1Target    testID
	genesisL1   testID
	genesisL2   testID
}

func makeState(st testState) *EngineDriverState {
	return &EngineDriverState{
		l1Head:      st.l1Head.ID(),
		l2Head:      st.l2Head.ID(),
		l2Finalized: st.l2Finalized.ID(),
		l1Target:    st.l1Target.ID(),
		Genesis: Genesis{
			L1: st.genesisL1.ID(),
			L2: st.genesisL2.ID(),
		},
	}
}

type mockDriver struct {
	mock.Mock
}

func (m *mockDriver) requestEngineHead(ctx context.Context) (refL1 eth.BlockID, refL2 eth.BlockID, err error) {
	returnArgs := m.Called(ctx)
	refL1 = returnArgs.Get(0).(eth.BlockID)
	refL2 = returnArgs.Get(1).(eth.BlockID)
	err = returnArgs.Get(2).(error)
	return
}

func (m *mockDriver) findSyncStart(ctx context.Context) (nextRefL1 eth.BlockID, refL2 eth.BlockID, err error) {
	returnArgs := m.Called(ctx)
	nextRefL1 = returnArgs.Get(0).(eth.BlockID)
	refL2 = returnArgs.Get(1).(eth.BlockID)
	err, _ = returnArgs.Get(2).(error)
	return
}

func (m *mockDriver) driverStep(ctx context.Context, nextRefL1 eth.BlockID, refL2 eth.BlockID, finalized eth.BlockID) (l2ID eth.BlockID, err error) {
	returnArgs := m.Called(ctx, nextRefL1, refL2, finalized)
	l2ID = returnArgs.Get(0).(eth.BlockID)
	err, _ = returnArgs.Get(1).(error)
	return
}

var _ Driver = (*mockDriver)(nil)

func TestEngineDriverState_RequestSync(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	driver := new(mockDriver)
	ctx := context.Background()

	state := makeState(testState{
		l1Head:      "c:2",
		l2Head:      "C:2",
		l2Finalized: "B:1",
		l1Target:    "e:4",
		genesisL1:   "a:0",
		genesisL2:   "b:0",
	})
	driver.On("findSyncStart", ctx).Return(testID("d:3").ID(), testID("C:2").ID(), nil)
	driver.On("driverStep", ctx, testID("d:3").ID(), testID("C:2").ID(), testID("B:1").ID()).Return(testID("D:3").ID(), nil)

	l2Updated := state.RequestSync(ctx, log, driver)

	assert.Equal(t, state.l1Head, testID("d:3").ID())
	assert.Equal(t, state.l2Head, testID("D:3").ID())
	assert.True(t, l2Updated)
}
