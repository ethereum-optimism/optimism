package sysgo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	sv2proxy "github.com/ethereum-optimism/optimism/sv2-proxy"
)

// TestSupervisorV2ProxyUserRPC validates that a transparent HTTP reverse proxy can forward
// rollup RPC requests to the embedded op-node user RPC exposed by Supervisor v2 under /opnode/{chainId}/.
func TestSupervisorV2ProxyUserRPC(gt *testing.T) {
	// test setup
	t := devtest.SerialT(gt)
	logger := testlog.Logger(gt, log.LevelInfo)
	onFail, onSkipNow := exiters(gt)
	p := devtest.NewP(context.Background(), logger, onFail, onSkipNow)
	gt.Cleanup(p.Close)
	ctx, cancel := context.WithTimeout(t.Ctx(), 120*time.Second)
	defer cancel()

	// bring up minimal system with SV2 and embedded op-node
	var ids DefaultMinimalSystemIDs
	opt := stack.Combine[*Orchestrator](
		DefaultMinimalSystemNoCL(&ids),
		WithInterop2ActivationOffsetForSV2(4),
		WithSupervisorV2OnFirstChain(),
	)
	orch := NewOrchestrator(p, stack.Combine[*Orchestrator]())
	stack.ApplyOptionLifecycle(opt, orch)
	system := shim.NewSystem(t)
	orch.Hydrate(system)

	// wait for SV2 HTTP to be ready
	sv2URL := os.Getenv("SV2_DENYLIST_URL")
	t.Require().NotEmpty(sv2URL)
	{
		ctx2, cancel2 := context.WithTimeout(t.Ctx(), 60*time.Second)
		defer cancel2()
		t.Require().NoError(WaitSV2Ready(ctx2, sv2URL))
	}

	// compute chain ID and start sv2-proxy rollup proxy
	l2Net := system.L2Networks()[0]
	chainID := l2Net.RollupConfig().L2ChainID.Uint64()
	proxy, proxyURL, err := sv2proxy.StartRollupProxy(ctx, sv2URL, chainID)
	t.Require().NoError(err)
	defer proxy.Close(context.Background())

	// simple health: call rollup SyncStatus via the proxy; expect 200 and json
	reqBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"rollup_syncStatus","params":[]}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, proxyURL, bytesReader(reqBody))
	t.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	t.Require().NoError(err)
	defer resp.Body.Close()
	t.Require().Equal(http.StatusOK, resp.StatusCode)
}

// bytesReader avoids importing bytes in multiple places; returns a ReadCloser from a []byte.
func bytesReader(b []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(b))
}

// TestSV2TwoChainProxyPreset_SafeProgress brings up the two-chain preset (no external proxy yet)
// and waits for cross-safe to progress on at least one chain.
func TestSV2TwoChainProxyPreset_SafeProgress(gt *testing.T) {
	t := devtest.SerialT(gt)
	logger := testlog.Logger(gt, log.LevelInfo)
	onFail, onSkipNow := exiters(gt)
	p := devtest.NewP(context.Background(), logger, onFail, onSkipNow)
	gt.Cleanup(p.Close)
	ctx, cancel := context.WithTimeout(t.Ctx(), 300*time.Second)
	defer cancel()

	const interopOffset = uint64(6)
	const confirmDepth = uint64(1)

	opt := stack.Combine[*Orchestrator](WithSV2TwoChainMinimalDepthProxy(interopOffset, confirmDepth))
	orch := NewOrchestrator(p, stack.Combine[*Orchestrator]())
	stack.ApplyOptionLifecycle(opt, orch)
	system := shim.NewSystem(t)
	orch.Hydrate(system)

	// Wait for SV2 to be ready
	sv2URL := os.Getenv("SV2_DENYLIST_URL")
	t.Require().NotEmpty(sv2URL)
	{
		ctx2, cancel2 := context.WithTimeout(t.Ctx(), 60*time.Second)
		defer cancel2()
		t.Require().NoError(WaitSV2Ready(ctx2, sv2URL))
	}

	// Fetch the first chain ID and wait for cross-safe to be available
	l2Nets := system.L2Networks()
	t.Require().GreaterOrEqual(len(l2Nets), 1)
	chainID, _ := l2Nets[0].ID().ChainID().Uint64()
	requireEventuallyHTTPOK(ctx, t, fmt.Sprintf("%s/v1/cross_safe?chainId=%d", sv2URL, chainID))
}

func requireEventuallyHTTPOK(ctx context.Context, t devtest.T, url string) {
	deadline := time.Now().Add(240 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err == nil {
			if resp, err2 := http.DefaultClient.Do(req); err2 == nil {
				if resp.StatusCode == http.StatusOK {
					_ = resp.Body.Close()
					return
				}
				_ = resp.Body.Close()
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Require().FailNow("cross_safe did not reach target in time")
}
