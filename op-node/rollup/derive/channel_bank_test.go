package derive

import (
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

	bt.out = &MockChannelBankOutput{MockOriginStage{originOpen: true, currentOrigin: bt.origins[ct.nextStartsAt]}}
	bt.cb = NewChannelBank(testlog.Logger(t, log.LvlError), cfg, bt.out)

	ct.fn(bt)
}

// format: <channelID-data>:<channelID-time>:<frame-number>:<content><optional-last-frame-marker "!">
// example: "abc:123:0:helloworld!"
type testFrame string

func (tf testFrame) ChannelID() ChannelID {
	parts := strings.Split(string(tf), ":")
	var chID ChannelID
	copy(chID.Data[:], parts[0])
	x, err := strconv.ParseUint(parts[1], 0, 64)
	if err != nil {
		panic(err)
	}
	chID.Time = x
	return chID
}

func (tf testFrame) FrameNumber() uint64 {
	parts := strings.Split(string(tf), ":")
	frameNum, err := strconv.ParseUint(parts[2], 0, 64)
	if err != nil {
		panic(err)
	}
	return frameNum
}

func (tf testFrame) IsLast() bool {
	parts := strings.Split(string(tf), ":")
	return strings.HasSuffix(parts[3], "!")
}

func (tf testFrame) Content() []byte {
	parts := strings.Split(string(tf), ":")
	return []byte(strings.TrimSuffix(parts[3], "!"))
}

func (tf testFrame) Encode() []byte {
	chID := tf.ChannelID()
	var out []byte
	out = append(out, chID.Data[:]...)
	out = append(out, makeUVarint(chID.Time)...)
	out = append(out, makeUVarint(tf.FrameNumber())...)
	content := tf.Content()
	out = append(out, makeUVarint(uint64(len(content)))...)
	out = append(out, content...)
	if tf.IsLast() {
		out = append(out, 1)
	} else {
		out = append(out, 0)
	}
	return out
}

func (bt *bankTestSetup) ingestData(data []byte) {
	require.NoError(bt.t, bt.cb.IngestData(data))
}
func (bt *bankTestSetup) ingestFrames(frames ...testFrame) {
	data := []byte{DerivationVersion0}
	for _, fr := range frames {
		data = append(data, fr.Encode()...)
	}
	bt.ingestData(data)
}
func (bt *bankTestSetup) repeatStep(max int, err error) {
	require.Equal(bt.t, err, RepeatStep(bt.t, bt.cb.Step, max))
}
func (bt *bankTestSetup) repeatResetStep(max int, err error) {
	require.Equal(bt.t, err, RepeatResetStep(bt.t, bt.cb.ResetStep, bt.l1, max))
}
func (bt *bankTestSetup) openOrigin(i int) {
	require.NoError(bt.t, bt.cb.OpenOrigin(bt.origins[i]))
}
func (bt *bankTestSetup) closeOrigin() {
	bt.cb.CloseOrigin()
}
func (bt *bankTestSetup) assertOrigin(i int) {
	require.Equal(bt.t, bt.cb.CurrentOrigin(), bt.origins[i])
}
func (bt *bankTestSetup) assertOriginTime(x uint64) {
	require.Equal(bt.t, x, bt.cb.CurrentOrigin().Time)
}
func (bt *bankTestSetup) expectOpenOrigin(i int) {
	bt.out.ExpectOpenOrigin(bt.origins[i], nil)
}
func (bt *bankTestSetup) expectCloseOrigin() {
	bt.out.ExpectCloseOrigin()
}
func (bt *bankTestSetup) doOpen(i int, expectNextOpen bool, addSteps int) {
	steps := 1
	if expectNextOpen {
		bt.expectOpenOrigin(i)
		steps += 1
	}
	steps += addSteps
	bt.openOrigin(i)
	bt.repeatStep(steps, nil)
	bt.assertExpectations()
	bt.assertOrigin(i)
	require.True(bt.t, bt.cb.IsOriginOpen())
}
func (bt *bankTestSetup) doClose(expectNextClose bool) {
	steps := 1
	if expectNextClose {
		bt.expectCloseOrigin()
		steps += 1
	}
	bt.closeOrigin()
	bt.repeatStep(steps, nil)
	bt.assertExpectations()
	require.False(bt.t, bt.cb.IsOriginOpen())
}
func (bt *bankTestSetup) expectChannel(data string) {
	bt.out.ExpectWriteChannel([]byte(data))
}
func (bt *bankTestSetup) expectL1RefByHash(i int) {
	bt.l1.ExpectL1BlockRefByHash(bt.origins[i].Hash, bt.origins[i], nil)
}
func (bt *bankTestSetup) assertExpectations() {
	bt.l1.AssertExpectations(bt.t)
	bt.l1.ExpectedCalls = nil
	bt.out.AssertExpectations(bt.t)
	bt.out.ExpectedCalls = nil
}
func (bt *bankTestSetup) logf(format string, args ...any) {
	bt.t.Logf(format, args...)
}

func TestL1ChannelBank(t *testing.T) {
	testCases := []channelBankTestCase{
		{
			name:           "time outs and buffering",
			originTimes:    []uint64{101, 102, 105, 107, 109},
			nextStartsAt:   3, // start next stage at 107
			channelTimeout: 3, // 107-3 = 104, reset to next lower origin, thus 102
			fn: func(bt *bankTestSetup) {
				bt.logf("reset to an origin that is timed out")
				bt.expectL1RefByHash(2)
				bt.expectL1RefByHash(1)
				bt.repeatResetStep(10, nil) // bank rewinds to origin pre-timeout
				bt.assertOrigin(1)
				bt.assertOriginTime(102)

				bt.logf("first step after reset should be EOF to start getting data")
				bt.repeatStep(1, nil)

				bt.logf("read from there onwards, but drop content since we did not reach start origin yet")
				bt.ingestFrames("a:98:0:too old") // timed out, can continue
				bt.repeatStep(3, nil)
				bt.ingestFrames("b:99:0:just new enough!") // closed frame, can be read, but dropped
				bt.repeatStep(3, nil)
				bt.assertExpectations()

				bt.logf("close origin 1")
				bt.doClose(false)
				bt.assertOrigin(1)

				bt.logf("open and close 2 without data")
				bt.doOpen(2, false, 0)
				bt.doClose(false)
				bt.assertOrigin(2)

				bt.logf("open 3, where we meet the next stage. Data isn't dropped anymore")
				bt.doOpen(3, false, 0)
				bt.assertOriginTime(107)

				bt.ingestFrames("c:104:0:foobar")
				bt.repeatStep(1, nil)
				bt.ingestFrames("d:104:0:other!")
				bt.repeatStep(1, nil)
				bt.ingestFrames("e:105:0:time-out-later") // timed out when we get to 109
				bt.repeatStep(1, nil)
				bt.ingestFrames("c:104:1:close!")
				bt.expectChannel("foobarclose")
				bt.expectChannel("other")
				bt.repeatStep(3, nil)
				bt.assertExpectations()

				bt.logf("close 3")
				bt.doClose(true)

				bt.logf("open 4")
				bt.expectChannel("time-out-later") // not closed, but processed after timeout
				bt.doOpen(4, true, 1)
				bt.assertOriginTime(109)

				bt.logf("data from 4")
				bt.ingestFrames("f:108:0:hello!")
				bt.expectChannel("hello")
				bt.repeatStep(2, nil)
				bt.assertExpectations()
			},
		},
		{
			name:           "duplicate frames",
			originTimes:    []uint64{101, 102},
			nextStartsAt:   0,
			channelTimeout: 3,
			fn: func(bt *bankTestSetup) {
				// don't do the whole setup process, just override where the stages are
				bt.cb.currentOrigin = bt.origins[0]
				bt.cb.originOpen = true
				bt.out.currentOrigin = bt.origins[0]
				bt.out.originOpen = true

				bt.assertOriginTime(101)

				bt.ingestFrames("x:102:0:foobar") // future frame is ignored when included too early
				bt.repeatStep(1, nil)

				bt.ingestFrames("a:101:0:first")
				bt.repeatStep(1, nil)
				bt.ingestFrames("a:101:1:second")
				bt.repeatStep(1, nil)
				bt.ingestFrames("a:101:0:altfirst") // ignored as duplicate
				bt.repeatStep(1, nil)
				bt.ingestFrames("a:101:1:altsecond") // ignored as duplicate
				bt.repeatStep(1, nil)
				bt.ingestFrames("a:100:0:new") // different time, considered to be different channel
				bt.repeatStep(1, nil)

				bt.doClose(true)

				bt.doOpen(1, true, 0)
				bt.ingestFrames("a:100:1:hi!") // close the other one first, but blocked
				bt.repeatStep(1, nil)
				bt.ingestFrames("a:101:2:!") // empty closing frame
				bt.expectChannel("firstsecond")
				bt.expectChannel("newhi")
				bt.repeatStep(3, nil)
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
				bt.cb.currentOrigin = bt.origins[0]
				bt.cb.originOpen = true
				bt.out.currentOrigin = bt.origins[0]
				bt.out.originOpen = true

				bt.assertOriginTime(101)

				badTx := []byte{DerivationVersion0}
				badTx = append(badTx, testFrame("a:101:0:helloworld!").Encode()...)
				badTx = append(badTx, testutils.RandomData(bt.rng, 30)...) // incomplete frame data
				bt.ingestData(badTx)
				bt.expectChannel("helloworld") // can still read the frames before the invalid data
				bt.repeatStep(2, nil)
				bt.assertExpectations()
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, testCase.Run)
	}
}
