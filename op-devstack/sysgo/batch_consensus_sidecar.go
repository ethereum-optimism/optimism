package sysgo

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-service/batchconsensus"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const batchConsensusMockProofSignerKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func startBatchConsensusMockProofSidecar(t devtest.T, valid bool) string {
	signer, err := batchConsensusMockProofSigner()
	t.Require().NoError(err)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req batchconsensus.ProofRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("decode request: %v", err), http.StatusBadRequest)
			return
		}
		l1ChainID, ok := new(big.Int).SetString(req.L1ChainID, 10)
		if !ok {
			http.Error(w, "invalid l1_chain_id", http.StatusBadRequest)
			return
		}
		l2ChainID, ok := new(big.Int).SetString(req.L2ChainID, 10)
		if !ok {
			http.Error(w, "invalid l2_chain_id", http.StatusBadRequest)
			return
		}
		normalizedReq, err := batchconsensus.NewProofRequest(l1ChainID, l2ChainID, req.BatchInbox, req.Batcher, req.BlobVersionedHashes)
		if err != nil {
			http.Error(w, fmt.Sprintf("normalize request: %v", err), http.StatusBadRequest)
			return
		}
		resp, err := batchconsensus.BuildSignedProofResponse(normalizedReq, signer, valid)
		if err != nil {
			http.Error(w, fmt.Sprintf("build signed proof: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Logger().Warn("Failed to encode batch consensus proof response", "err", err)
		}
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	t.Require().NoError(err)
	server := &http.Server{Handler: mux}
	t.Cleanup(func() {
		_ = server.Close()
	})
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Logger().Error("Batch consensus mock proof sidecar failed", "err", err)
		}
	}()
	endpoint := "http://" + listener.Addr().String()
	t.Logger().Info("Started batch consensus mock proof sidecar", "endpoint", endpoint, "valid", valid, "signer", crypto.PubkeyToAddress(signer.PublicKey))
	return endpoint
}

func batchConsensusMockProofSigner() (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(batchConsensusMockProofSignerKey)
}

func batchConsensusMockProofSignerAddress() common.Address {
	key, err := batchConsensusMockProofSigner()
	if err != nil {
		panic(err)
	}
	return crypto.PubkeyToAddress(key.PublicKey)
}

func startBatchConsensusCommonwareProofSidecar(t devtest.T, valid bool) string {
	port, err := getAvailableLocalPort()
	t.Require().NoError(err)
	endpoint := "http://127.0.0.1:" + port

	stdOut := logpipe.LogCallback(func(line []byte) {
		t.Logger().Info("Batch consensus Commonware Simplex sidecar stdout", "line", string(line))
	})
	stdErr := logpipe.LogCallback(func(line []byte) {
		t.Logger().Info("Batch consensus Commonware Simplex sidecar stderr", "line", string(line))
	})
	sub := NewSubProcess(t, stdOut, stdErr)

	execPath, err := rustbin.Spec{
		SrcDir:  "op-batcher-consensus-sidecar",
		Package: "op-batcher-consensus-sidecar",
		Binary:  "op-batcher-consensus-sidecar",
	}.EnsureExists(t.Ctx(), t.Logger())
	t.Require().NoError(err, "prepare batch consensus Commonware Simplex sidecar binary")

	args := []string{"--listen", "127.0.0.1:" + port, "--commonware-simplex"}
	if !valid {
		args = append(args, "--invalid")
	}
	t.Require().NoError(sub.Start(execPath, args, nil), "start batch consensus Commonware Simplex sidecar")
	waitTCPReady(t, endpoint, 10*time.Second)

	t.Logger().Info("Started batch consensus Commonware Simplex proof sidecar", "endpoint", endpoint, "valid", valid)
	return endpoint
}
