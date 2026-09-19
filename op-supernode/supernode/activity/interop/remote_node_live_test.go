package interop

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	coreinterop "github.com/ethereum-optimism/optimism/op-core/interop"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/remote"
)

// TestLiveRemoteNodeAgainstOPSepolia is the whole chain of custody in one test: a real
// shim process talking to real OP Sepolia, a real HTTPAdapter, the real ingester, and a
// real LogsDB — ending with the check that actually matters, that a genuine on-chain log
// is referenceable by the checksum an executing message would carry.
//
// Every link is a place the leg can break silently: an anchor at the wrong height, a
// logHash derived differently on the two sides, a log index shifted by a filtered log.
// None of those show up in a unit test with fabricated data, and all of them would
// surface only as an opaque verification failure once the supernode is running.
//
// It is opt-in because it needs the network and a built shim:
//
//	ALTDA_SHIM_BIN=/path/to/opsepolia-shim \
//	  go test ./op-supernode/supernode/activity/interop/ -run TestLiveRemoteNodeAgainstOPSepolia -v
func TestLiveRemoteNodeAgainstOPSepolia(t *testing.T) {
	shimBin := os.Getenv("ALTDA_SHIM_BIN")
	if shimBin == "" {
		t.Skip("set ALTDA_SHIM_BIN to the opsepolia-shim binary to run the live interop test")
	}
	rpcURL := os.Getenv("ALTDA_SHIM_RPC")
	if rpcURL == "" {
		rpcURL = "https://sepolia.optimism.io"
	}
	const opSepolia = 11155420
	chainID := eth.ChainIDFromUInt64(opSepolia)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	baseURL := startShim(t, ctx, shimBin, rpcURL, opSepolia)

	// The real production adapter, pointed at the real shim.
	db, err := openLogsDB(testLogger(), chainID, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	node := &remoteNode{
		log:         testLogger(),
		adapter:     remote.NewHTTPAdapter(chainID, baseURL, nil),
		db:          db,
		poll:        2 * time.Second,
		maxPerCycle: 8,
	}

	// Ingest at least three live finalized blocks. The first is the anchor, at
	// whatever height OP Sepolia's finalized head happens to be — the case the
	// start-height patch exists for.
	const wantBlocks = 3
	ingested := 0
	for ingested < wantBlocks {
		require.NoError(t, ctx.Err(), "timed out ingesting live blocks")
		ok, err := node.ingestOnce(ctx)
		require.NoError(t, err)
		if !ok {
			// Nothing finalized yet; the shim's anchor lookback should make this
			// rare, but wait for the chain rather than failing on timing.
			select {
			case <-ctx.Done():
				t.Fatal("timed out waiting for the next finalized block")
			case <-time.After(2 * time.Second):
			}
			continue
		}
		ingested++
	}

	first, err := db.FirstSealedBlock()
	require.NoError(t, err)
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	t.Logf("ingested OP Sepolia blocks %d..%d (anchored %d blocks past genesis)",
		first.Number, latest.Number, first.Number)

	require.Greater(t, first.Number, uint64(1_000_000),
		"the anchor must be at OP Sepolia's real height, not replayed from genesis")
	require.Equal(t, first.Number+wantBlocks-1, latest.Number, "blocks must be contiguous")

	// Now the payoff: take a real log from an ingested block, derive the checksum an
	// executing message would carry, and ask the LogsDB for it.
	client, err := ethclient.DialContext(ctx, rpcURL)
	require.NoError(t, err)
	defer client.Close()

	var (
		found     bool
		foundLog  types.Log
		foundSeal messages.BlockSeal
	)
	for num := first.Number; num <= latest.Number && !found; num++ {
		seal, err := db.FindSealedBlock(num)
		require.NoError(t, err)
		logs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(num),
			ToBlock:   new(big.Int).SetUint64(num),
		})
		require.NoError(t, err)
		if len(logs) == 0 {
			continue
		}
		found, foundLog, foundSeal = true, logs[0], seal
	}
	require.True(t, found, "none of the ingested blocks carried logs; rerun to sample another window")

	query := messages.ContainsQuery{
		BlockNum:  foundSeal.Number,
		LogIdx:    uint32(foundLog.Index),
		Timestamp: foundSeal.Timestamp,
		Checksum: messages.ChecksumArgs{
			BlockNumber: foundSeal.Number,
			LogIndex:    uint32(foundLog.Index),
			Timestamp:   foundSeal.Timestamp,
			ChainID:     chainID,
			LogHash:     messages.LogToLogHash(&foundLog),
		}.Checksum(),
	}
	sealed, err := db.Contains(query)
	require.NoError(t, err,
		"a real OP Sepolia log ingested through the shim must be referenceable")
	require.Equal(t, foundSeal.Number, sealed.Number)
	t.Logf("verified real log: block %d, logIndex %d, address %s",
		foundSeal.Number, foundLog.Index, foundLog.Address)

	// And the negative, so the positive means something: a tampered checksum at the
	// same real position must be rejected.
	bad := query
	bad.Checksum = messages.MessageChecksum(common.HexToHash("0xdeadbeef"))
	_, err = db.Contains(bad)
	require.ErrorIs(t, err, coreinterop.ErrConflict)
}

// startShim launches the shim on a free port and waits for it to report healthy,
// returning its base URL. It waits on the service's own readiness signal rather than a
// fixed sleep, so a slow public RPC lengthens the wait instead of failing the test.
func startShim(t *testing.T, ctx context.Context, bin, rpcURL string, chainID uint64) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	var out bytes.Buffer
	cmd := exec.Command(bin,
		"--listen", addr,
		"--rpc", rpcURL,
		"--datadir", t.TempDir(),
		"--chain-id", fmt.Sprint(chainID),
		// A lookback, so finalized history is available immediately: an L2's
		// finalized head jumps forward every few minutes rather than every block.
		"--anchor-offset", "300",
		"--log.level", "info",
	)
	cmd.Stdout = &out
	cmd.Stderr = &out
	require.NoError(t, cmd.Start())

	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		if t.Failed() {
			t.Logf("shim output:\n%s", out.String())
		}
	})

	baseURL := "http://" + addr
	healthy := false
	for !healthy {
		require.NoError(t, ctx.Err(), "shim never became healthy; output:\n%s", out.String())
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			t.Fatalf("shim exited early:\n%s", out.String())
		}
		resp, err := http.Get(baseURL + "/healthz")
		if err == nil {
			healthy = resp.StatusCode == http.StatusOK
			resp.Body.Close()
		}
		if !healthy {
			select {
			case <-ctx.Done():
				t.Fatalf("shim never became healthy; output:\n%s", out.String())
			case <-time.After(200 * time.Millisecond):
			}
		}
	}
	t.Logf("shim ready at %s", baseURL)
	return baseURL
}
