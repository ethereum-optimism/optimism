package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-lagoon/sdmcheck"
	"github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestRPCReplayFlagConfigAndDefaults(t *testing.T) {
	for _, command := range [][]string{{"all"}, {"sdm", "all"}, {"sdm", "block"}, {"sdm", "verifier"}, {"sdm", "rpc"}} {
		t.Run(command[len(command)-1], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			require.NoError(t, os.WriteFile(path, []byte("[sdm]\ncheck-rpc = true\n"), 0o600))
			for _, test := range []struct {
				args    []string
				enabled bool
			}{
				{nil, false}, {[]string{"--config", path}, true},
				{[]string{"--config", path, "--sdm.check-rpc=false"}, false},
				{[]string{"--sdm-check-rpc"}, true},
			} {
				app := newApp()
				app.Writer = io.Discard
				app.ErrWriter = io.Discard
				seen := false
				action := func(c *cli.Context) error {
					seen = true
					require.Equal(t, test.enabled, c.Bool(SDMCheckRPC.Name))
					if len(command) == 2 && command[1] != "all" {
						cfg, cleanup, err := resolveSDMConfig(context.Background(), log.New(), c, false)
						require.NoError(t, err)
						defer cleanup()
						require.Equal(t, test.enabled, cfg.CheckRPC)
					}
					return nil
				}
				for _, cmd := range app.Commands {
					if cmd.Name != command[0] {
						continue
					}
					if len(command) == 1 {
						cmd.Action = action
						continue
					}
					for _, sub := range cmd.Subcommands {
						if sub.Name == command[1] {
							sub.Action = action
						}
					}
				}
				args := append([]string{"check-lagoon"}, command...)
				args = append(args, test.args...)
				require.NoError(t, app.Run(args))
				require.True(t, seen)
			}
		})
	}
}

func TestRPCReplayEnvironmentFlag(t *testing.T) {
	t.Setenv("CHECK_LAGOON_SDM_CHECK_RPC", "true")
	app := newApp()
	app.Writer = io.Discard
	app.ErrWriter = io.Discard
	for _, command := range app.Commands {
		if command.Name == "all" {
			command.Action = func(c *cli.Context) error { require.True(t, c.Bool(SDMCheckRPC.Name)); return nil }
		}
	}
	require.NoError(t, app.Run([]string{"check-lagoon", "all"}))
}

// A conforming payload/receipt source with no debug methods: unsupported replay
// must fail visibly, while both producer and verifier failure reports survive.
type replayReceiptAPI struct {
	block    *sdm.RPCBlock
	receipts map[common.Hash]*sdm.RPCReceipt
}

func (a *replayReceiptAPI) ChainId() *hexutil.Big { return (*hexutil.Big)(big.NewInt(901)) }
func (a *replayReceiptAPI) GetBlockByNumber(context.Context, string, bool) *sdm.RPCBlock {
	return a.block
}
func (a *replayReceiptAPI) GetBlockByHash(context.Context, common.Hash, bool) map[string]any {
	return map[string]any{"hash": a.block.Hash, "parentHash": common.HexToHash("0x99"), "transactions": a.block.Transactions}
}
func (a *replayReceiptAPI) GetTransactionReceipt(_ context.Context, hash common.Hash) *sdm.RPCReceipt {
	return a.receipts[hash]
}

func TestRPCReplayCLIReportsFailuresAsJSON(t *testing.T) {
	payload, err := rlp.EncodeToBytes(optypes.PostExecPayload{Version: 1, BlockNumber: 42, GasRefundEntries: []optypes.SDMGasEntry{{Index: 0, GasRefund: 7}}})
	require.NoError(t, err)
	block := &sdm.RPCBlock{Number: 42, Hash: common.HexToHash("0xbeef"), Transactions: []sdm.RPCTransaction{
		{Hash: common.HexToHash("0x1"), Type: 2},
		{Hash: common.HexToHash("0x2"), Type: optypes.PostExecTxType, Input: payload},
	}}
	api := &replayReceiptAPI{block: block, receipts: make(map[common.Hash]*sdm.RPCReceipt)}
	for i, tx := range block.Transactions {
		api.receipts[tx.Hash] = &sdm.RPCReceipt{TxHash: tx.Hash, BlockHash: block.Hash, TransactionIndex: hexutil.Uint64(i), Type: tx.Type, Status: 1}
	}
	api.receipts[block.Transactions[0].Hash].GasUsed = 93
	api.receipts[block.Transactions[0].Hash].OPGasRefund = ptr.New(hexutil.Uint64(7))
	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("eth", api))
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	defer server.Stop()
	var output bytes.Buffer
	app := newApp()
	app.Writer = &output
	app.ErrWriter = io.Discard
	err = app.Run([]string{"check-lagoon", "sdm", "rpc", "--sdm.l2", httpServer.URL, "--sdm.l2-verifier", httpServer.URL, "--sdm.block", "42", "--sdm.json"})
	require.ErrorContains(t, err, "producer RPC replay")
	require.ErrorContains(t, err, "verifier RPC replay")
	var result sdmcheck.Result
	require.NoError(t, json.Unmarshal(output.Bytes(), &result), output.String())
	for _, report := range []*sdm.RPCReplayResult{result.RPCReplay, result.VerifierRPCReplay} {
		require.NotNil(t, report)
		require.Equal(t, uint64(42), report.BlockNumber)
		require.NotEmpty(t, report.Checks)
		require.NotEmpty(t, report.Checks[0].Error)
	}
}

func TestRPCReplayCLIRequiresBlockAndRejectsWorkloadFlags(t *testing.T) {
	for _, args := range [][]string{
		{"check-lagoon", "sdm", "rpc"},
		{"check-lagoon", "sdm", "rpc", "--sdm.account", "abcd"},
	} {
		app := newApp()
		app.Writer = io.Discard
		app.ErrWriter = io.Discard
		require.Error(t, app.Run(args))
	}
}
