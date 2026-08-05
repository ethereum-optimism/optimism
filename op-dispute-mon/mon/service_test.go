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

func TestInitMonitorWiresEveryTypedLane(t *testing.T) {
	ctx := t.Context()
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

	require.Equal(t, []string{
		"*extract.L1HeadBlockNumEnricher",
		"*extract.OutputAgreementEnricher",
		"*extract.SuperAgreementEnricher",
		"*extract.AnchorStateRegistryEnricher",
	}, registeredTypeNames(service.commonEnrichers()))
	require.Equal(t, []string{
		"*extract.ClaimEnricher",
		"*extract.RecipientEnricher",
		"*extract.WithdrawalsEnricher",
		"*extract.BondEnricher",
		"*extract.BalanceEnricher",
	}, registeredTypeNames(service.faultEnrichers()))

	service.initMonitor(ctx, &cfg)
	require.NotNil(t, service.monitor)
	require.NotNil(t, service.monitor.fetchHeadBlock)
	require.NotNil(t, service.monitor.extract)
	require.NotNil(t, service.monitor.forecast)
	require.NotNil(t, service.monitor.checkAnchorState)
	expectedCommon := []string{
		"(*UpdateTimeMonitor).CheckUpdateTimes-fm",
		"(*NodeEndpointErrorsMonitor).CheckNodeEndpointErrors-fm",
		"(*NodeEndpointErrorCountMonitor).CheckNodeEndpointErrorCount-fm",
		"(*NodeEndpointOutOfSyncMonitor).CheckNodeEndpointOutOfSync-fm",
		"(*MixedAvailability).CheckMixedAvailability-fm",
		"(*MixedSafetyMonitor).CheckMixedSafety-fm",
		"(*DifferentRootMonitor).CheckDifferentRoots-fm",
		"(*GameTypeMonitor).CheckGameTypes-fm",
	}
	require.Len(t, service.monitor.commonMonitors, len(expectedCommon))
	for i, expected := range expectedCommon {
		require.Contains(t, registeredFunctionName(service.monitor.commonMonitors[i]), expected)
	}
	expectedFault := []string{
		"bonds.(*Bonds).CheckBonds-fm",
		"(*ResolutionMonitor).CheckResolutions-fm",
		"(*ClaimMonitor).CheckClaims-fm",
		"(*WithdrawalMonitor).CheckWithdrawals-fm",
		"(*L2ChallengesMonitor).CheckL2Challenges-fm",
	}
	require.Len(t, service.monitor.faultMonitors, len(expectedFault))
	for i, expected := range expectedFault {
		require.Contains(t, registeredFunctionName(service.monitor.faultMonitors[i]), expected)
	}
}

func registeredFunctionName(fn any) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func registeredTypeNames[T any](values []T) []string {
	names := make([]string, len(values))
	for i, value := range values {
		names[i] = reflect.TypeOf(value).String()
	}
	return names
}

type serviceRPCStub struct {
	*batchingTest.AbiBasedRpc
}

func (*serviceRPCStub) Close() {}

func (*serviceRPCStub) Subscribe(context.Context, string, any, ...any) (ethereum.Subscription, error) {
	return nil, errors.New("subscriptions are not supported by the test RPC")
}
