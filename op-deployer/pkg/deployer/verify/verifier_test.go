package verify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func TestVerifierWithEmbeddedArtifacts(t *testing.T) {
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	artifactsFS, err := artifacts.ExtractEmbedded(testCacheDir)
	require.NoError(t, err, "embedded artifacts should be extracted successfully")

	verifier, err := NewVerifier(testAPIKey, 1, artifactsFS, log.New(log.JSONHandler(io.Discard)), nil)
	require.NoError(t, err, "verifier should be created successfully with embedded artifacts")
	require.NotNil(t, verifier, "verifier should not be nil")

	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")

		if action == "getabi" {
			resp := EtherscanGenericResp{
				Status:  "1",
				Message: "OK",
				Result:  "[]",
			}
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
			return
		}

		http.NotFound(w, r)
	}))
	defer fakeServer.Close()

	verifier.etherscan = NewEtherscanClient(testAPIKey, fakeServer.URL, rate.NewLimiter(rate.Inf, 1))

	bundleFile := filepath.Join(testCacheDir, "contracts.json")
	bundle := map[string]common.Address{
		"SystemConfigProxy":   common.HexToAddress("0x02f909cf91c2134e70a67950b7f27db7c8ee55d6"),
		"OptimismPortalProxy": common.HexToAddress("0x7bd8879acf1e74547455c7ddc07f5c3f4a3c133d"),
	}
	bundleData, err := json.Marshal(bundle)
	require.NoError(t, err)
	err = os.WriteFile(bundleFile, bundleData, 0o644)
	require.NoError(t, err)

	err = verifier.verifyContractBundle(context.Background(), bundleFile, "")
	require.NoError(t, err)
	require.Equal(t, len(bundle), verifier.numSkipped, "all contracts should be skipped as already verified")
	require.Equal(t, 0, verifier.numFailed)
	require.Equal(t, 0, verifier.numVerified)
}
