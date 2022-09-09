package derive

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type MockChannelBankOutput struct {
	MockOriginStage
}

func (m *MockChannelBankOutput) WriteChannel(data []byte) {
	m.MethodCalled("WriteChannel", data)
}

func (m *MockChannelBankOutput) ExpectWriteChannel(data []byte) {
	m.On("WriteChannel", data).Once().Return()
}

var _ ChannelBankOutput = (*MockChannelBankOutput)(nil)

type bankTestSetup struct {
	origins []eth.L1BlockRef
	t       *testing.T
	rng     *rand.Rand
	cb      *ChannelBank
	out     *MockChannelBankOutput
	l1      *testutils.MockL1Source
}

type channelBankTestCase struct {
	name           string
	originTimes    []uint64
	nextStartsAt   int
	channelTimeout uint64
	fn             func(bt *bankTestSetup)
}

func (ct *channelBankTestCase) Run(t *testing.T) {
	cfg := &rollup.Config{
		ChannelTimeout: ct.channelTimeout,
	}

	bt := &bankTestSetup{
		t:   t,
		rng: rand.New(rand.NewSource(1234)),
		l1:  &testutils.MockL1Source{},
	}

	bt.origins = append(bt.origins, testutils.RandomBlockRef(bt.rng))
	for i := range ct.originTimes[1:] {
		ref := testutils.NextRandomRef(bt.rng, bt.origins[i])
		bt.origins = append(bt.origins, ref)
	}
	for i, x := range ct.originTimes {
		bt.origins[i].Time = x
	}

	bt.out = &MockChannelBankOutput{MockOriginStage{progress: Progress{Origin: bt.origins[ct.nextStartsAt], Closed: false}}}
	bt.cb = NewChannelBank(testlog.Logger(t, log.LvlError), cfg, bt.out)

	ct.fn(bt)
}

// format: <channelID-data>:<frame-number>:<content><optional-last-frame-marker "!">
// example: "abc:0:helloworld!"
type testFrame string

func (tf testFrame) ChannelID() ChannelID {
	parts := strings.Split(string(tf), ":")
	var chID ChannelID
	copy(chID[:], parts[0])
	return chID
}

func (tf testFrame) FrameNumber() uint64 {
	parts := strings.Split(string(tf), ":")
	frameNum, err := strconv.ParseUint(parts[1], 0, 64)
	if err != nil {
		panic(err)
	}
	return frameNum
}

func (tf testFrame) IsLast() bool {
	parts := strings.Split(string(tf), ":")
	return strings.HasSuffix(parts[2], "!")
}

func (tf testFrame) Content() []byte {
	parts := strings.Split(string(tf), ":")
	return []byte(strings.TrimSuffix(parts[2], "!"))
}

func (tf testFrame) ToFrame() Frame {
	return Frame{
		ID:          tf.ChannelID(),
		FrameNumber: uint16(tf.FrameNumber()),
		Data:        tf.Content(),
		IsLast:      tf.IsLast(),
	}
}

func (bt *bankTestSetup) ingestData(data []byte) {
	bt.cb.IngestData(data)
}

func (bt *bankTestSetup) ingestFrames(frames ...testFrame) {
	data := new(bytes.Buffer)
	data.WriteByte(DerivationVersion0)
	for _, fr := range frames {
		f := fr.ToFrame()
		if err := f.MarshalBinary(data); err != nil {
			panic(fmt.Errorf("error in making frame during test: %w", err))
		}
	}
	bt.ingestData(data.Bytes())
}
func (bt *bankTestSetup) repeatStep(max int, outer int, outerClosed bool, err error) {
	require.Equal(bt.t, err, RepeatStep(bt.t, bt.cb.Step, Progress{Origin: bt.origins[outer], Closed: outerClosed}, max))
}
func (bt *bankTestSetup) repeatResetStep(max int, err error) {
	require.Equal(bt.t, err, RepeatResetStep(bt.t, bt.cb.ResetStep, bt.l1, max))
}

func (bt *bankTestSetup) assertOrigin(i int) {
	require.Equal(bt.t, bt.cb.progress.Origin, bt.origins[i])
}
func (bt *bankTestSetup) assertOriginTime(x uint64) {
	require.Equal(bt.t, x, bt.cb.progress.Origin.Time)
}
func (bt *bankTestSetup) expectChannel(data string) {
	bt.out.ExpectWriteChannel([]byte(data))
}
func (bt *bankTestSetup) expectL1BlockRefByNumber(i int) {
	bt.l1.ExpectL1BlockRefByNumber(bt.origins[i].Number, bt.origins[i], nil)
}
func (bt *bankTestSetup) assertExpectations() {
	bt.l1.AssertExpectations(bt.t)
	bt.l1.ExpectedCalls = nil
	bt.out.AssertExpectations(bt.t)
	bt.out.ExpectedCalls = nil
}

func TestL1ChannelBank(t *testing.T) {
	testCases := []channelBankTestCase{
		{
			name:           "time outs and buffering",
			originTimes:    []uint64{0, 1, 2, 3, 4, 5},
			nextStartsAt:   3, // Start next stage at block #3
			channelTimeout: 2, // Start at block #1
			fn: func(bt *bankTestSetup) {
				bt.expectL1BlockRefByNumber(1)
				bt.repeatResetStep(10, nil)
				bt.ingestFrames("a:0:first") // will time out b/c not closed

				bt.repeatStep(10, 1, true, nil)
				bt.repeatStep(10, 2, false, nil)
				bt.assertOrigin(2)

				bt.repeatStep(10, 2, true, nil)
				bt.repeatStep(10, 3, false, nil)
				bt.assertOrigin(3)

				bt.repeatStep(10, 3, true, nil)
				bt.repeatStep(10, 4, false, nil)
				bt.assertOrigin(4)

				// Properly closed channel
				bt.expectChannel("foobarclosed")
				bt.ingestFrames("b:0:foobar")
				bt.ingestFrames("b:1:closed!")
				bt.repeatStep(10, 4, true, nil)
				bt.assertExpectations()
			},
		},
		{
			name:           "duplicate frames",
			originTimes:    []uint64{0, 1, 2, 3, 4, 5},
			nextStartsAt:   3, // Start next stage at block #3
			channelTimeout: 2, // Start at block #1c
			fn: func(bt *bankTestSetup) {
				bt.expectL1BlockRefByNumber(1)
				bt.repeatResetStep(10, nil)
				bt.ingestFrames("a:0:first") // will time out b/c not closed

				bt.repeatStep(10, 1, true, nil)
				bt.repeatStep(10, 2, false, nil)
				bt.assertOrigin(2)

				bt.repeatStep(10, 2, true, nil)
				bt.repeatStep(10, 3, false, nil)
				bt.assertOrigin(3)

				bt.repeatStep(10, 3, true, nil)
				bt.repeatStep(10, 4, false, nil)
				bt.assertOrigin(4)

				bt.ingestFrames("a:0:first")
				bt.repeatStep(1, 4, false, nil)
				bt.ingestFrames("a:1:second")
				bt.repeatStep(1, 4, false, nil)
				bt.ingestFrames("a:0:altfirst") // ignored as duplicate
				bt.repeatStep(1, 4, false, nil)
				bt.ingestFrames("a:1:altsecond") // ignored as duplicate
				bt.repeatStep(1, 4, false, nil)
				bt.ingestFrames("b:0:new")
				bt.repeatStep(1, 4, false, nil)

				// close origin 4
				bt.repeatStep(2, 4, true, nil)

				// open origin 1
				bt.repeatStep(2, 5, false, nil)
				bt.ingestFrames("b:1:hi!") // close the other one first, but blocked
				bt.repeatStep(1, 5, false, nil)
				bt.ingestFrames("a:2:!") // empty closing frame
				bt.expectChannel("firstsecond")
				bt.expectChannel("newhi")
				bt.repeatStep(5, 5, false, nil)
				bt.assertExpectations()
			},
		},
		{
			name:           "skip bad frames",
			originTimes:    []uint64{101, 102},
			nextStartsAt:   0,
			channelTimeout: 3,
			fn: func(bt *bankTestSetup) {
				// don't do the whole setup process, just override where the stages are
				bt.cb.progress = Progress{Origin: bt.origins[0], Closed: false}
				bt.out.progress = Progress{Origin: bt.origins[0], Closed: false}

				bt.assertOriginTime(101)

				badTx := new(bytes.Buffer)
				badTx.WriteByte(DerivationVersion0)
				goodFrame := testFrame("a:0:helloworld!").ToFrame()
				if err := goodFrame.MarshalBinary(badTx); err != nil {
					panic(fmt.Errorf("error in marshalling frame: %w", err))
				}
				badTx.Write(testutils.RandomData(bt.rng, 30)) // incomplete frame data
				bt.ingestData(badTx.Bytes())
				// Expect the bad frame to render the entire chunk invalid.
				bt.repeatStep(2, 0, false, nil)
				bt.assertExpectations()
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, testCase.Run)
	}
}
