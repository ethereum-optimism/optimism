package op_txpool

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/testlog"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/stretchr/testify/require"
)

type authAddingTransport struct {
	underlying     http.RoundTripper
	invalidateAddr bool
}

func (c *authAddingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get(authHeaderKey) == "" {
		body, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(io.NopCloser(bytes.NewBuffer(body)))

		privKey, _ := crypto.GenerateKey()
		sig, _ := crypto.Sign(accounts.TextHash(body), privKey)
		addr := crypto.PubkeyToAddress(privKey.PublicKey)
		if c.invalidateAddr {
			addr = common.Address{19: 1}
		}

		req.Header.Set(authHeaderKey, fmt.Sprintf("%s:%s", addr, hex.EncodeToString(sig)))
	}

	return c.underlying.RoundTrip(req)
}

func TestSendRawTransactionConditionalDisabled(t *testing.T) {
	log := testlog.Logger(t, log.LevelInfo)
	cfg := &CLIConfig{SendRawTransactionConditionalEnabled: false}
	svc, err := NewConditionalTxService(context.Background(), log, metrics.With(metrics.NewRegistry()), cfg)
	require.NoError(t, err)

	hash, err := svc.SendRawTransactionConditional(context.Background(), nil, nil)
	require.Zero(t, hash)
	require.Equal(t, endpointDisabledErr, err)
}

func TestSendRawTransactionConditionalMissingAuth(t *testing.T) {
	log := testlog.Logger(t, log.LevelInfo)
	cfg := &CLIConfig{SendRawTransactionConditionalEnabled: true}
	svc, err := NewConditionalTxService(context.Background(), log, metrics.With(metrics.NewRegistry()), cfg)
	require.NoError(t, err)

	// by default the peer info is not set in the ctx
	cond := &types.TransactionConditional{}
	hash, err := svc.SendRawTransactionConditional(context.Background(), nil, cond)
	require.Zero(t, hash)
	require.Equal(t, invalidAuthenticationErr, err)
}

func TestSendRawTransactionConditionalMissingConditional(t *testing.T) {
	log := testlog.Logger(t, log.LevelInfo)
	cfg := &CLIConfig{SendRawTransactionConditionalEnabled: true}
	svc, err := NewConditionalTxService(context.Background(), log, metrics.With(metrics.NewRegistry()), cfg)
	require.NoError(t, err)

	hash, err := svc.SendRawTransactionConditional(context.Background(), nil, nil)
	require.Zero(t, hash)
	require.Equal(t, missingConditionalErr, err)
}

func TestSendRawTransactionConditionalBadAuth(t *testing.T) {
	log := testlog.Logger(t, log.LevelInfo)
	cfg := &CLIConfig{SendRawTransactionConditionalEnabled: true}
	svc, err := NewConditionalTxService(context.Background(), log, metrics.With(metrics.NewRegistry()), cfg)
	require.NoError(t, err)

	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("test", svc))
	defer srv.Stop()

	ts := httptest.NewServer(srv)
	c, err := rpc.Dial(ts.URL)
	require.NoError(t, err)
	defer c.Close()

	c.SetHeader(authHeaderKey, "foobarbaz")
	err = c.Call(nil, "test_sendRawTransactionConditional", "", &types.TransactionConditional{})
	require.NotNil(t, err)
	require.Equal(t, invalidAuthenticationErr.Message, err.Error())
}

func TestSendRawTransactionConditionalBadSignature(t *testing.T) {
	log := testlog.Logger(t, log.LevelInfo)
	cfg := &CLIConfig{SendRawTransactionConditionalEnabled: true}
	svc, err := NewConditionalTxService(context.Background(), log, metrics.With(metrics.NewRegistry()), cfg)
	require.NoError(t, err)

	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("test", svc))
	defer srv.Stop()

	ts := httptest.NewServer(srv)
	c, err := rpc.Dial(ts.URL)
	require.NoError(t, err)
	defer c.Close()

	c.SetHeader(authHeaderKey, fmt.Sprintf("%s:%s", common.HexToAddress("0xa"), "foobar"))
	err = c.Call(nil, "test_sendRawTransactionConditional", "", &types.TransactionConditional{})
	require.NotNil(t, err)
	require.Equal(t, invalidAuthenticationErr.Message, err.Error())
}

func TestSendRawTransactionConditionalInvalidTxTarget(t *testing.T) {
	log := testlog.Logger(t, log.LevelInfo)
	cfg := &CLIConfig{SendRawTransactionConditionalEnabled: true, SendRawTransactionConditionalRateLimit: 1_000_000}
	svc, err := NewConditionalTxService(context.Background(), log, metrics.With(metrics.NewRegistry()), cfg)
	require.NoError(t, err)

	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("test", svc))
	defer srv.Stop()

	ts := httptest.NewServer(srv)
	httpc := &http.Client{Transport: &authAddingTransport{http.DefaultTransport, false}}
	c, err := rpc.DialOptions(context.Background(), ts.URL, rpc.WithHTTPClient(httpc))
	require.NoError(t, err)
	defer c.Close()

	txBytes, _ := rlp.EncodeToBytes(types.NewTransaction(0, common.Address{19: 1}, big.NewInt(0), 0, big.NewInt(0), nil))
	err = c.Call(nil, "test_sendRawTransactionConditional", hexutil.Encode(txBytes), &types.TransactionConditional{})
	require.NotNil(t, err)
	require.Equal(t, entrypointSupportErr.Message, err.Error())
}

func TestSendRawTransactionConditionalCaller(t *testing.T) {
	log := testlog.Logger(t, log.LevelInfo)
	cfg := &CLIConfig{SendRawTransactionConditionalEnabled: true, SendRawTransactionConditionalRateLimit: 1_000_000}
	svc, err := NewConditionalTxService(context.Background(), log, metrics.With(metrics.NewRegistry()), cfg)
	require.NoError(t, err)

	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("test", svc))
	defer srv.Stop()

	ts := httptest.NewServer(srv)
	httpc := &http.Client{Transport: &authAddingTransport{http.DefaultTransport, true}}
	c, err := rpc.DialOptions(context.Background(), ts.URL, rpc.WithHTTPClient(httpc))
	require.NoError(t, err)
	defer c.Close()

	txBytes, _ := rlp.EncodeToBytes(types.NewTransaction(0, common.Address{19: 1}, big.NewInt(0), 0, big.NewInt(0), nil))
	err = c.Call(nil, "test_sendRawTransactionConditional", hexutil.Encode(txBytes), &types.TransactionConditional{})
	require.NotNil(t, err)
	require.Equal(t, invalidAuthenticationErr.Message, err.Error())
}

func TestSendRawTransactionConditionalValidSignature(t *testing.T) {
	log := testlog.Logger(t, log.LevelInfo)
	cfg := &CLIConfig{SendRawTransactionConditionalEnabled: true, SendRawTransactionConditionalRateLimit: 1_000_000}
	svc, err := NewConditionalTxService(context.Background(), log, metrics.With(metrics.NewRegistry()), cfg)
	require.NoError(t, err)

	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("test", svc))
	defer srv.Stop()

	ts := httptest.NewServer(srv)
	httpc := &http.Client{Transport: &authAddingTransport{http.DefaultTransport, false}}
	c, err := rpc.DialOptions(context.Background(), ts.URL, rpc.WithHTTPClient(httpc))
	require.NoError(t, err)
	defer c.Close()

	txBytes, _ := rlp.EncodeToBytes(types.NewTransaction(0, predeploys.EntryPoint_v060Addr, big.NewInt(0), 0, big.NewInt(0), nil))
	require.Nil(t, c.Call(nil, "test_sendRawTransactionConditional", hexutil.Encode(txBytes), &types.TransactionConditional{}))
}

func TestSendRawTransactionConditionalInvalidConditionals(t *testing.T) {
	costExcessiveCond := types.TransactionConditional{KnownAccounts: make(types.KnownAccounts)}
	for i := 0; i < (types.TransactionConditionalMaxCost + 1); i++ {
		iBig := big.NewInt(int64(i))
		root := common.BigToHash(iBig)
		costExcessiveCond.KnownAccounts[common.BigToAddress(iBig)] = types.KnownAccount{StorageRoot: &root}
	}

	uint64Ptr := func(num uint64) *uint64 { return &num }
	tests := []struct {
		name     string
		cond     types.TransactionConditional
		mustFail bool
	}{

		{
			name:     "ok validation",
			cond:     types.TransactionConditional{BlockNumberMin: big.NewInt(1), BlockNumberMax: big.NewInt(2), TimestampMin: uint64Ptr(1), TimestampMax: uint64Ptr(2)},
			mustFail: false,
		},
		{
			name:     "validation. block min greater than max",
			cond:     types.TransactionConditional{BlockNumberMin: big.NewInt(2), BlockNumberMax: big.NewInt(1)},
			mustFail: true,
		},
		{
			name:     "validation. timestamp min greater than max",
			cond:     types.TransactionConditional{TimestampMin: uint64Ptr(2), TimestampMax: uint64Ptr(1)},
			mustFail: true,
		},
		{
			name:     "excessive cost",
			cond:     costExcessiveCond,
			mustFail: true,
		},
	}

	log := testlog.Logger(t, log.LevelInfo)
	cfg := &CLIConfig{SendRawTransactionConditionalEnabled: true, SendRawTransactionConditionalRateLimit: 1_000_000}
	svc, err := NewConditionalTxService(context.Background(), log, metrics.With(metrics.NewRegistry()), cfg)
	require.NoError(t, err)

	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("test", svc))
	defer srv.Stop()

	ts := httptest.NewServer(srv)
	httpc := &http.Client{Transport: &authAddingTransport{http.DefaultTransport, false}}
	c, err := rpc.DialOptions(context.Background(), ts.URL, rpc.WithHTTPClient(httpc))
	require.NoError(t, err)
	defer c.Close()

	txBytes, _ := rlp.EncodeToBytes(types.NewTransaction(0, predeploys.EntryPoint_v060Addr, big.NewInt(0), 0, big.NewInt(0), nil))
	txString := hexutil.Encode(txBytes)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err = c.Call(nil, "test_sendRawTransactionConditional", txString, &test.cond)
			if test.mustFail && err == nil {
				t.Errorf("Test %s should fail", test.name)
			}
			if !test.mustFail && err != nil {
				t.Errorf("Test %s should pass but got err: %v", test.name, err)
			}
		})
	}
}
