package sdmcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestDeriveSDMAccountKey(t *testing.T) {
	funder, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)
	first, err := DeriveSDMAccountKey(funder)
	require.NoError(t, err)
	second, err := DeriveSDMAccountKey(funder)
	require.NoError(t, err)
	require.Equal(t, crypto.FromECDSA(first), crypto.FromECDSA(second))
	require.NotEqual(t, crypto.PubkeyToAddress(funder.PublicKey), crypto.PubkeyToAddress(first.PublicKey))
}

func TestFundingDeficit(t *testing.T) {
	minimum := big.NewInt(20)
	require.Equal(t, big.NewInt(20), fundingDeficit(big.NewInt(0), minimum))
	require.Equal(t, big.NewInt(5), fundingDeficit(big.NewInt(15), minimum))
	require.Zero(t, fundingDeficit(big.NewInt(20), minimum).Sign())
	require.Zero(t, fundingDeficit(big.NewInt(25), minimum).Sign())
	require.Equal(t, big.NewInt(20), minimum)
}

type fakeProducer struct {
	status    *sdmStatus
	statusErr error
	optInSets []bool
	enableErr error
	block     *sdm.RPCBlock
}

func (f *fakeProducer) CallContext(_ context.Context, result any, method string, args ...any) error {
	switch method {
	case "admin_sdmStatus":
		if f.statusErr != nil {
			return f.statusErr
		}
		*result.(*sdmStatus) = *f.status
		return nil
	case "admin_setOperatorSdmOptIn":
		f.optInSets = append(f.optInSets, args[0].(bool))
		if args[0].(bool) {
			return f.enableErr
		}
		return nil
	case "eth_getBlockByNumber":
		encoded, err := json.Marshal(f.block)
		if err != nil {
			return err
		}
		*result.(*json.RawMessage) = encoded
		return nil
	default:
		return fmt.Errorf("unexpected method %s", method)
	}
}

func TestEnsureOptIn(t *testing.T) {
	cfg := Config{Log: testlog.Logger(t, log.LevelDebug)}

	t.Run("already opted in", func(t *testing.T) {
		producer := &fakeProducer{status: &sdmStatus{OperatorSDMOptIn: true}}
		toggled, err := ensureOptIn(context.Background(), Config{Log: cfg.Log, OptIn: true}, producer)
		require.NoError(t, err)
		require.False(t, toggled)
		require.Empty(t, producer.optInSets)
	})
	t.Run("disabled operator is not overridden by default", func(t *testing.T) {
		producer := &fakeProducer{status: &sdmStatus{}}
		_, err := ensureOptIn(context.Background(), cfg, producer)
		require.ErrorContains(t, err, "has not opted in")
		require.Empty(t, producer.optInSets)
	})
	t.Run("explicit opt-in enables and restores", func(t *testing.T) {
		producer := &fakeProducer{status: &sdmStatus{}}
		optInCfg := Config{Log: cfg.Log, OptIn: true}
		toggled, err := ensureOptIn(context.Background(), optInCfg, producer)
		require.NoError(t, err)
		require.True(t, toggled)
		restoreOptIn(optInCfg, producer)
		require.Equal(t, []bool{true, false}, producer.optInSets)
	})
	t.Run("failed enable is still restored", func(t *testing.T) {
		producer := &fakeProducer{status: &sdmStatus{}, enableErr: errors.New("context deadline exceeded")}
		toggled, err := ensureOptIn(context.Background(), Config{Log: cfg.Log, OptIn: true}, producer)
		require.ErrorContains(t, err, "admin_setOperatorSdmOptIn(true)")
		require.False(t, toggled)
		require.Equal(t, []bool{true, false}, producer.optInSets)
	})
	t.Run("unreadable status without opt-in proceeds", func(t *testing.T) {
		producer := &fakeProducer{statusErr: errors.New("method not found")}
		toggled, err := ensureOptIn(context.Background(), cfg, producer)
		require.NoError(t, err)
		require.False(t, toggled)
	})
	t.Run("unreadable status with opt-in fails", func(t *testing.T) {
		producer := &fakeProducer{statusErr: errors.New("method not found")}
		_, err := ensureOptIn(context.Background(), Config{Log: cfg.Log, OptIn: true}, producer)
		require.ErrorContains(t, err, "admin_sdmStatus")
		require.Empty(t, producer.optInSets)
	})
}

type fakeRollup struct {
	cfg    *rollup.Config
	safe   uint64
	output common.Hash
}

func (f *fakeRollup) RollupConfig(context.Context) (*rollup.Config, error) {
	if f.cfg == nil {
		return nil, errors.New("not used")
	}
	return f.cfg, nil
}

func (f *fakeRollup) SyncStatus(context.Context) (*eth.SyncStatus, error) {
	return &eth.SyncStatus{SafeL2: eth.L2BlockRef{Number: f.safe}}, nil
}

func (f *fakeRollup) OutputAtBlock(_ context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	return &eth.OutputResponse{BlockRef: eth.L2BlockRef{Number: blockNum, Hash: f.output}}, nil
}

func TestCheckSafeHead(t *testing.T) {
	validated := &sdm.RPCBlock{Number: 7, Hash: common.HexToHash("0x07")}
	other := common.HexToHash("0x08")
	cases := []struct {
		name      string
		safe      uint64
		output    common.Hash
		canonical common.Hash
		wantErr   string
	}{
		{name: "safe and canonical", safe: 9, output: validated.Hash, canonical: validated.Hash},
		{name: "safe head behind", safe: 6, output: validated.Hash, canonical: validated.Hash, wantErr: "safe head is 6, waiting for 7"},
		{name: "rollup node disagrees", safe: 7, output: other, canonical: validated.Hash, wantErr: "rollup node's safe block"},
		{name: "producer reorged", safe: 7, output: validated.Hash, canonical: other, wantErr: "reorged before becoming safe"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{Rollup: &fakeRollup{safe: tc.safe, output: tc.output}}
			// Expire mid-wait: the error must keep the last failure reason.
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			producer := &fakeProducer{block: &sdm.RPCBlock{Number: validated.Number, Hash: tc.canonical}}
			err := checkSafeHead(ctx, cfg, producer, validated, uint64(validated.Number))
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// stubAdmin and stubEth let CheckAll enable the opt-in, then fail the deploy.
type stubAdmin struct {
	optInSets []bool
}

func (s *stubAdmin) SdmStatus() sdmStatus { return sdmStatus{} }

func (s *stubAdmin) SetOperatorSdmOptIn(enabled bool) {
	s.optInSets = append(s.optInSets, enabled)
}

type stubEth struct{}

func (stubEth) ChainId() hexutil.Uint64 { return 901 }

func (stubEth) GetBalance(common.Address, string) *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(1e18))
}

func (stubEth) GetBlockByNumber(string, bool) map[string]any {
	return map[string]any{"number": hexutil.Uint64(5), "timestamp": hexutil.Uint64(100)}
}

func (stubEth) GetTransactionCount(common.Address, string) (hexutil.Uint64, error) {
	return 0, errors.New("nonce unavailable")
}

func TestCheckAllRestoresOptInOnFailure(t *testing.T) {
	admin := &stubAdmin{}
	server := gethrpc.NewServer()
	require.NoError(t, server.RegisterName("admin", admin))
	require.NoError(t, server.RegisterName("eth", stubEth{}))
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)
	t.Cleanup(server.Stop)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	cfg := Config{
		RPCURL: httpServer.URL,
		Key:    key,
		OptIn:  true,
		Log:    testlog.Logger(t, log.LevelDebug),
	}
	_, err = CheckAll(context.Background(), cfg)
	require.ErrorContains(t, err, "nonce unavailable")
	require.Equal(t, []bool{true, false}, admin.optInSets)
}

func TestCheckLagoonActive(t *testing.T) {
	lagoonAt := func(ts uint64) *fakeRollup { return &fakeRollup{cfg: &rollup.Config{LagoonTime: &ts}} }
	producer := &fakeProducer{block: &sdm.RPCBlock{Number: 7, Timestamp: 100}}

	require.NoError(t, checkLagoonActive(context.Background(), Config{}, producer, "0x7"))
	require.NoError(t, checkLagoonActive(context.Background(), Config{Rollup: lagoonAt(100)}, producer, "0x7"))
	err := checkLagoonActive(context.Background(), Config{Rollup: lagoonAt(101)}, producer, "0x7")
	require.ErrorContains(t, err, "Lagoon is not active at block 7 timestamp 100")
}

func TestCheckAllChecksLagoonBeforeWorkload(t *testing.T) {
	admin := &stubAdmin{}
	server := gethrpc.NewServer()
	require.NoError(t, server.RegisterName("admin", admin))
	require.NoError(t, server.RegisterName("eth", stubEth{}))
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)
	t.Cleanup(server.Stop)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	lagoonTime := uint64(200)
	cfg := Config{
		RPCURL: httpServer.URL,
		Key:    key,
		OptIn:  true,
		Log:    testlog.Logger(t, log.LevelDebug),
		Rollup: &fakeRollup{cfg: &rollup.Config{LagoonTime: &lagoonTime}},
	}
	_, err = CheckAll(context.Background(), cfg)
	require.ErrorContains(t, err, "Lagoon is not active at block 5 timestamp 100")
	require.Empty(t, admin.optInSets, "no opt-in toggle before the activation check")
}
