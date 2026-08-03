package mon

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/config"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/extract"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestServiceInitMonitorWiresVariantMonitors(t *testing.T) {
	ctx := context.Background()
	factoryAddress := common.Address{0xaa}
	logger := testlog.Logger(t, log.LevelCrit)
	stubRPC := &serviceRPCStub{AbiBasedRpc: batchingTest.NewAbiBasedRpc(t, factoryAddress, snapshots.LoadDisputeGameFactoryABI())}
	stubRPC.SetResponse(factoryAddress, "version", rpcblock.Latest, nil, []interface{}{"1.4.0"})
	caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)
	factory, err := contracts.NewDisputeGameFactoryContract(ctx, metrics.NoopMetrics, factoryAddress, caller)
	require.NoError(t, err)
	l1Client, err := sources.NewL1Client(stubRPC, logger, metrics.NoopMetrics, sources.L1ClientSimpleConfig(true, sources.RPCKindAny, 100))
	require.NoError(t, err)

	service := &Service{
		logger:          logger,
		metrics:         metrics.NoopMetrics,
		honestActors:    monTypes.NewHonestActors(nil),
		factoryContract: factory,
		cl:              clock.NewDeterministicClock(time.Unix(1, 0)),
		l1Client:        l1Client,
		l1Caller:        caller,
	}
	service.game = extract.NewGameCallerCreator(service.metrics, caller)
	cfg := config.NewCombinedConfig(factoryAddress, "unused", nil, nil)

	require.NoError(t, service.initMonitor(ctx, &cfg))
	require.NotNil(t, service.monitor)
	require.NotNil(t, service.monitor.fetchHeadBlock)
	require.NotNil(t, service.monitor.extract)
	require.NotNil(t, service.monitor.forecast)
	require.NotNil(t, service.monitor.checkAnchorState)
	requireMonitorMethods(t, service.monitor.commonMonitors, []string{
		"UpdateTimeMonitor).CheckUpdateTimes",
		"NodeEndpointErrorsMonitor).CheckNodeEndpointErrors",
		"NodeEndpointErrorCountMonitor).CheckNodeEndpointErrorCount",
		"NodeEndpointOutOfSyncMonitor).CheckNodeEndpointOutOfSync",
		"MixedAvailability).CheckMixedAvailability",
		"MixedSafetyMonitor).CheckMixedSafety",
		"DifferentRootMonitor).CheckDifferentRoots",
		"GameTypeMonitor).CheckGameTypes",
	})
	requireMonitorMethods(t, service.monitor.faultMonitors, []string{
		"ResolutionMonitor).CheckResolutions",
		"ClaimMonitor).CheckClaims",
		"L2ChallengesMonitor).CheckL2Challenges",
	})
	requireMonitorMethods(t, service.monitor.bondMonitors, []string{
		"Bonds).CheckBonds",
		"WithdrawalMonitor).CheckWithdrawals",
	})
}

type serviceRPCStub struct {
	*batchingTest.AbiBasedRpc
}

func (*serviceRPCStub) Close() {}

func (*serviceRPCStub) Subscribe(context.Context, string, any, ...any) (ethereum.Subscription, error) {
	return nil, errors.New("subscriptions are not supported by the test RPC")
}

func requireMonitorMethods[T any](t *testing.T, methods []T, expected []string) {
	t.Helper()
	require.Len(t, methods, len(expected))
	for i, method := range methods {
		function := runtime.FuncForPC(reflect.ValueOf(method).Pointer())
		require.NotNil(t, function)
		require.Contains(t, function.Name(), expected[i])
	}
}
