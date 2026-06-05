package mon

import (
	"context"
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	asrA = common.HexToAddress("0xaa00000000000000000000000000000000000000")
	asrB = common.HexToAddress("0xbb00000000000000000000000000000000000000")
)

func TestAnchorStateMonitor_NoGames(t *testing.T) {
	monitor, metrics, providers := setupAnchorStateTest(t)
	monitor.CheckAnchorState(context.Background(), common.Hash{0xaa}, nil)
	require.Empty(t, metrics.recorded)
	require.Empty(t, providers.created)
}

func TestAnchorStateMonitor_DedupesPerASR(t *testing.T) {
	monitor, metrics, providers := setupAnchorStateTest(t)
	providers.set(asrA, big.NewInt(100), nil)
	providers.set(asrB, big.NewInt(200), nil)

	games := []*types.EnrichedGameData{
		{AnchorStateRegistry: asrA},
		{AnchorStateRegistry: asrA},
		{AnchorStateRegistry: asrB},
		{AnchorStateRegistry: common.Address{}}, // skipped
	}
	monitor.CheckAnchorState(context.Background(), common.Hash{0xaa}, games)

	require.Equal(t, map[common.Address]uint64{asrA: 100, asrB: 200}, metrics.recorded)
	require.Equal(t, 1, providers.providers[asrA].calls, "asrA queried once despite two games")
	require.Equal(t, 1, providers.providers[asrB].calls)
}

func TestAnchorStateMonitor_ToleratesPerASRError(t *testing.T) {
	monitor, metrics, providers := setupAnchorStateTest(t)
	providers.set(asrA, nil, errors.New("boom"))
	providers.set(asrB, big.NewInt(200), nil)

	games := []*types.EnrichedGameData{
		{AnchorStateRegistry: asrA},
		{AnchorStateRegistry: asrB},
	}
	monitor.CheckAnchorState(context.Background(), common.Hash{0xaa}, games)

	require.Equal(t, map[common.Address]uint64{asrB: 200}, metrics.recorded)
}

func TestAnchorStateMonitor_SequenceNumberOverflow(t *testing.T) {
	monitor, metrics, providers := setupAnchorStateTest(t)
	tooBig := new(big.Int).Add(new(big.Int).SetUint64(math.MaxUint64), big.NewInt(1))
	providers.set(asrA, tooBig, nil)

	monitor.CheckAnchorState(context.Background(), common.Hash{0xaa}, []*types.EnrichedGameData{
		{AnchorStateRegistry: asrA},
	})

	require.Equal(t, map[common.Address]uint64{asrA: math.MaxUint64}, metrics.recorded)
}

func setupAnchorStateTest(t *testing.T) (*AnchorStateMonitor, *mockAnchorStateMetrics, *mockAnchorRootProviders) {
	logger := testlog.Logger(t, log.LevelError)
	metrics := &mockAnchorStateMetrics{recorded: map[common.Address]uint64{}}
	providers := &mockAnchorRootProviders{providers: map[common.Address]*mockAnchorRootProvider{}}
	monitor := NewAnchorStateMonitor(logger, metrics, providers.create)
	return monitor, metrics, providers
}

type mockAnchorStateMetrics struct {
	recorded map[common.Address]uint64
}

func (m *mockAnchorStateMetrics) RecordAnchorStateL2SequenceNumber(asr common.Address, l2SequenceNumber uint64) {
	m.recorded[asr] = l2SequenceNumber
}

type mockAnchorRootProviders struct {
	providers map[common.Address]*mockAnchorRootProvider
	created   []common.Address
}

func (m *mockAnchorRootProviders) set(addr common.Address, seq *big.Int, err error) {
	m.providers[addr] = &mockAnchorRootProvider{seq: seq, err: err}
}

func (m *mockAnchorRootProviders) create(addr common.Address) AnchorRootProvider {
	m.created = append(m.created, addr)
	if p, ok := m.providers[addr]; ok {
		return p
	}
	return &mockAnchorRootProvider{seq: big.NewInt(0)}
}

type mockAnchorRootProvider struct {
	seq   *big.Int
	err   error
	calls int
}

func (m *mockAnchorRootProvider) GetAnchorRoot(_ context.Context, _ rpcblock.Block) (common.Hash, *big.Int, error) {
	m.calls++
	if m.err != nil {
		return common.Hash{}, nil, m.err
	}
	return common.Hash{}, m.seq, nil
}
