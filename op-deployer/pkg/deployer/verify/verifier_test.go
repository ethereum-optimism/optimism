package verify

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func bootstrapContractAddresses() map[string]common.Address {
	addrType := reflect.TypeOf(common.Address{})
	structTypes := []reflect.Type{
		reflect.TypeOf((*opcm.DeploySuperchainOutput)(nil)).Elem(),
		reflect.TypeOf((*opcm.DeployImplementationsOutput)(nil)).Elem(),
	}

	addresses := make(map[string]common.Address)
	index := int64(1)

	for _, structType := range structTypes {
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			if field.Type == addrType {
				addresses[field.Name] = common.BigToAddress(big.NewInt(index))
				index++
			}
		}
	}

	return addresses
}

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

	bundle := bootstrapContractAddresses()

	bundleFile := filepath.Join(testCacheDir, "contracts.json")
	bundleData, err := json.Marshal(bundle)
	require.NoError(t, err)
	err = os.WriteFile(bundleFile, bundleData, 0o644)
	require.NoError(t, err)

	err = verifier.verifyContractBundle(context.Background(), bundleFile, "")
	require.NoError(t, err)
	require.Equal(t, len(bundle), verifier.numSkipped, "all contracts should be skipped as already verified")
	// Ensure we actually have contracts to test with
	require.Greater(t, len(bundle), 0, "contract bundle is empty")
	require.Equal(t, 0, verifier.numFailed)
	require.Equal(t, 0, verifier.numVerified)
}
