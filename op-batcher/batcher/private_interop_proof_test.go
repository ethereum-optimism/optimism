package batcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-private-interop/builder"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var updatePublicationRequest = flag.Bool("update-publication-request", false,
	"regenerate testdata/publication-request-v2.json, the §G.2 golden request the Rust host test reads")

const publicationRequestGolden = "testdata/publication-request-v2.json"

// publicationRequestKeys is the normative §G.2 field set.
var publicationRequestKeys = []string{
	"version", "prover", "private_rpc", "projection_rpc", "l1_rpc",
	"private_config", "l1_config", "projection_config", "dependency_set",
	"parent_hash", "anchor", "anchor_output", "recovery_hash", "blocks", "private_data",
	"l1_head", "expected_public_values",
}

// piSP1 switches a test encoder to a sp1-private-projection-v1 deployment with mock proofs.
func piSP1(enc *PrivateInteropEncoder, ranges *staticRanges) (privateJSON, l1JSON []byte) {
	piDeployedShape(enc.cfg.PrivateRollup, common.Hash{0x1d})
	piDeployedShape(enc.cfg.Rollup, common.Hash{0x9e})
	piExecutionMock(enc, ranges)
	privateJSON, _ = json.Marshal(enc.cfg.PrivateRollup)
	l1JSON, _ = json.Marshal(params.MergedTestChainConfig)
	cfg := enc.cfg.Rollup
	cfg.PrivateProjection = &projection.Config{
		Verifier:          projection.SP1PrivateProjectionV1,
		GenesisOutputRoot: cfg.PrivateProjection.GenesisOutputRoot,
		ProgramVKey:       common.Hash{31: 1},
		PrivateConfigHash: projection.PrivateConfigHash(privateJSON, l1JSON),
		DependencySetHash: cfg.PrivateProjection.DependencySetHash,
		MockProofs:        true,
	}
	enc.cfg.RollupConfigHash = projection.ConfigHash(cfg.PrivateProjection, projection.Context{
		ChainID: cfg.L2ChainID, GenesisNumber: cfg.Genesis.L2.Number, GenesisTime: cfg.Genesis.L2Time,
		BlockTime: cfg.BlockTime, GenesisHash: cfg.Genesis.L2.Hash,
	})
	return privateJSON, l1JSON
}

// piDeployedShape fills the fields a deployed rollup.json always has (and Kona's RollupConfig
// requires) on a test config, without changing anything the seam renders from.
func piDeployedShape(cfg *rollup.Config, genesisHash common.Hash) {
	cfg.L1ChainID = big.NewInt(900)
	cfg.ChannelTimeoutBedrock = 300
	cfg.Genesis.L1 = eth.BlockID{Hash: piL1Chain(1)[0].Hash, Number: 0}
	cfg.Genesis.L2 = eth.BlockID{Hash: genesisHash, Number: 0}
	cfg.Genesis.SystemConfig.BatcherAddr = common.Address{0x42}
	cfg.Genesis.SystemConfig.Scalar = eth.Bytes32{31: 1}
	cfg.BatchInboxAddress = common.HexToAddress("0x00a4fe4c6aaa0729d7699c387e7f281dd64afa2a")
	cfg.DepositContractAddress = common.HexToAddress("0x4e6c7f6b8a1c1a1e2e5d2c0e7b5d9d7f1c3a5b7e")
	cfg.L1SystemConfigAddress = common.HexToAddress("0x6b3e3f7c1d9a2e5c8b4f0a7d6e1c9b2a5f8e3d71")
	cfg.ChainOpConfig = &opparams.OptimismConfig{EIP1559Elasticity: 6, EIP1559Denominator: 50, EIP1559DenominatorCanyon: ptr(uint64(250))}
}

func ptr[T any](v T) *T { return &v }

// capturedRange is what the seam hands the producer.
type capturedRange struct {
	candidate   *builder.BuiltRange
	privateData []byte
	start       RangeStart
}

var errCaptured = errors.New("captured")

// piCapture drives the seam until the producer is called and returns its inputs.
func piCapture(t *testing.T, enc *PrivateInteropEncoder) capturedRange {
	t.Helper()
	var got capturedRange
	enc.cfg.Prove = func(_ context.Context, candidate *builder.BuiltRange, data []byte, start RangeStart) ([]byte, error) {
		got = capturedRange{candidate, data, start}
		return nil, errCaptured
	}
	co := piFill(t, enc, 901, piCadence)
	require.Eventually(t, func() bool { return errors.Is(co.Close(), errCaptured) }, 5*time.Second, time.Millisecond)
	return got
}

func piSP1Capture(t *testing.T) (*PrivateInteropEncoder, capturedRange, projectionProducerConfig) {
	t.Helper()
	enc, ranges, _ := piEncoderWithTxs(t, piProjectionTxs())
	privateJSON, l1JSON := piSP1(enc, ranges)
	captured := piCapture(t, enc)
	projectionJSON, err := json.Marshal(enc.cfg.Rollup)
	require.NoError(t, err)
	depSet, err := depset.NewStaticConfigDependencySet(map[eth.ChainID]*depset.StaticConfigDependency{
		eth.ChainIDFromUInt64(901): {}, eth.ChainIDFromUInt64(902): {},
	})
	require.NoError(t, err)
	depSetJSON, err := json.Marshal(depSet)
	require.NoError(t, err)
	return enc, captured, projectionProducerConfig{
		Prover:           ProverNativeMock,
		PrivateRPC:       "http://private-el:8545",
		ProjectionRPC:    "http://projection-el:8545",
		L1RPC:            "http://l1-el:8545",
		PublicRollup:     enc.cfg.Rollup,
		PrivateConfig:    privateJSON,
		L1Config:         l1JSON,
		ProjectionConfig: projectionJSON,
		DependencySet:    depSetJSON,
		Timeout:          time.Minute,
		TimeoutPerBlock:  100 * time.Millisecond,
	}
}

func requestKeys(t *testing.T, raw []byte) []string {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestPublicationRequestV2Golden pins the §G.2 wire format the Rust host consumes.
func TestPublicationRequestV2Golden(t *testing.T) {
	enc, captured, cfg := piSP1Capture(t)
	statement, err := projectionPreflight(enc.cfg.Rollup, captured.candidate, captured.start)
	require.NoError(t, err)
	raw, err := buildPublicationRequest(cfg, statement, captured.candidate, captured.privateData, captured.start)
	require.NoError(t, err)

	want := append([]string(nil), publicationRequestKeys...)
	sort.Strings(want)
	require.Equal(t, want, requestKeys(t, raw), "sp1 requests carry exactly the §G.2 fields and no expected_digest")
	var req publicationRequest
	require.NoError(t, json.Unmarshal(raw, &req))
	require.Equal(t, uint64(2), req.Version)
	pv := projection.PublicValues(statement)
	require.Equal(t, hexutil.Bytes(pv[:]), req.ExpectedPublicValues)
	require.Len(t, req.ExpectedPublicValues, 672)
	last := captured.candidate.Blocks[len(captured.candidate.Blocks)-1]
	require.Equal(t, last.Origin.Hash, req.L1Head)
	require.Equal(t, cfg.PrivateConfig, []byte(req.PrivateConfig), "the pinned private rollup bytes, unmodified")
	require.Equal(t, cfg.L1Config, []byte(req.L1Config))
	require.Len(t, req.Blocks, len(captured.candidate.Blocks))

	var pretty bytes.Buffer
	require.NoError(t, json.Indent(&pretty, raw, "", "  "))
	pretty.WriteByte('\n')
	if *updatePublicationRequest {
		require.NoError(t, os.MkdirAll(filepath.Dir(publicationRequestGolden), 0o755))
		require.NoError(t, os.WriteFile(publicationRequestGolden, pretty.Bytes(), 0o644))
	}
	golden, err := os.ReadFile(publicationRequestGolden)
	require.NoError(t, err)
	require.Equal(t, string(golden), pretty.String(), "run with -update-publication-request to regenerate")
}

// TestPublicationRequestExecutionMockCarriesDigest: only execution-mock-v1 requests carry the
// legacy admission digest, and their prover is native.
func TestPublicationRequestExecutionMockCarriesDigest(t *testing.T) {
	enc, ranges, _ := piEncoderWithTxs(t, piProjectionTxs())
	piExecutionMock(enc, ranges)
	captured := piCapture(t, enc)
	statement, err := projectionPreflight(enc.cfg.Rollup, captured.candidate, captured.start)
	require.NoError(t, err)
	raw, err := buildPublicationRequest(projectionProducerConfig{Prover: ProverNative, PublicRollup: enc.cfg.Rollup},
		statement, captured.candidate, captured.privateData, captured.start)
	require.NoError(t, err)
	want := append([]string{"expected_digest"}, publicationRequestKeys...)
	sort.Strings(want)
	require.Equal(t, want, requestKeys(t, raw))
	var req publicationRequest
	require.NoError(t, json.Unmarshal(raw, &req))
	require.Equal(t, ProverNative, req.Prover)
	require.Equal(t, projection.AdmissionDigest(*statement), *req.ExpectedDigest)
}

// TestPublicationRequestHasNoRecoveryIntervalData: the request is span-bounded. A range continuing
// from an anchor far behind it carries the same blocks and the same fields; recovery blocks and
// witnesses are the child's to fetch over RPC.
func TestPublicationRequestHasNoRecoveryIntervalData(t *testing.T) {
	enc, captured, cfg := piSP1Capture(t)
	statement, err := projectionPreflight(enc.cfg.Rollup, captured.candidate, captured.start)
	require.NoError(t, err)
	near, err := buildPublicationRequest(cfg, statement, captured.candidate, captured.privateData, captured.start)
	require.NoError(t, err)
	farStart := captured.start
	farStart.Continuation.Anchor = eth.BlockID{Number: 1, Hash: common.Hash{0xa1}}
	farStart.Continuation.RecoveryHash = common.Hash{0xec}
	far, err := buildPublicationRequest(cfg, statement, captured.candidate, captured.privateData, farStart)
	require.NoError(t, err)

	require.Equal(t, requestKeys(t, near), requestKeys(t, far))
	var a, b publicationRequest
	require.NoError(t, json.Unmarshal(near, &a))
	require.NoError(t, json.Unmarshal(far, &b))
	require.Equal(t, a.Blocks, b.Blocks, "only the span's own blocks")
	require.Equal(t, common.Hash{0xec}, b.RecoveryHash, "the recovery interval is committed by hash only")
	require.InDelta(t, len(near), len(far), 3, "900 recovery blocks add no request bytes")
}

// fakeProofCommand writes an executable that saves its stdin next to itself and prints stdout.
func fakeProofCommand(t *testing.T, stdout string, extra string) (command, requestPath string) {
	t.Helper()
	dir := t.TempDir()
	requestPath = filepath.Join(dir, "request.json")
	command = filepath.Join(dir, "producer")
	script := fmt.Sprintf("#!/bin/sh\n[ \"$1\" = --publication-request ] || exit 7\n%s\ncat > %q\nprintf '%%s\\n' %q\n",
		extra, requestPath, stdout)
	require.NoError(t, os.WriteFile(command, []byte(script), 0o755))
	return command, requestPath
}

// TestProjectionProducerVerifiesWithDeployedVerifier: the producer checks the child's envelope with
// projection.VerifierFor on the deployed profile. A valid mock envelope passes; an execution-mock
// envelope, which the old hard-coded verifier accepted, does not; nor does a tampered one.
func TestProjectionProducerVerifiesWithDeployedVerifier(t *testing.T) {
	enc, captured, cfg := piSP1Capture(t)
	statement, err := projectionPreflight(enc.cfg.Rollup, captured.candidate, captured.start)
	require.NoError(t, err)
	envelope := projection.MockEnvelope(enc.cfg.Rollup.PrivateProjection.ProgramVKey, projection.PublicValues(statement))

	run := func(stdout string) ([]byte, string, error) {
		command, requestPath := fakeProofCommand(t, stdout, "")
		cfg := cfg
		cfg.Command = command
		proof, err := newProjectionProducer(context.Background(), cfg)(t.Context(), captured.candidate, captured.privateData, captured.start)
		request, _ := os.ReadFile(requestPath)
		return proof, string(request), err
	}
	proof, request, err := run(hexutil.Encode(envelope))
	require.NoError(t, err)
	require.Equal(t, envelope, proof)
	require.Contains(t, request, `"version":2`)
	require.Contains(t, request, `"prover":"native-mock"`)

	_, _, err = run(hexutil.Encode(projection.ExecutionMockProof(*statement)))
	require.ErrorContains(t, err, "does not verify")

	tampered := bytes.Clone(envelope)
	tampered[len(tampered)-1] ^= 1
	_, _, err = run(hexutil.Encode(tampered))
	require.ErrorContains(t, err, "does not verify")

	_, _, err = run("not hex")
	require.ErrorContains(t, err, "malformed envelope")
}

// TestProofTimeoutScalesWithInterval: effective = base + perBlock × (lastBlock − anchorBlock).
func TestProofTimeoutScalesWithInterval(t *testing.T) {
	require.Equal(t, 2*time.Minute, effectiveProofTimeout(2*time.Minute, 100*time.Millisecond, 10, 10))
	require.Equal(t, 2*time.Minute+100*time.Millisecond, effectiveProofTimeout(2*time.Minute, 100*time.Millisecond, 11, 10))
	// A 12-hour outage at 2-second blocks: 21,600 blocks, about 38 minutes.
	require.Equal(t, 2*time.Minute+36*time.Minute, effectiveProofTimeout(2*time.Minute, 100*time.Millisecond, 21_600, 0))
	require.Equal(t, time.Duration(1<<63-1), effectiveProofTimeout(time.Minute, time.Hour, 1<<62, 0), "saturates")
	require.Equal(t, time.Minute, effectiveProofTimeout(time.Minute, time.Second, 5, 9), "an anchor at or past the end adds nothing")

	// The producer applies it: a child that hangs is stopped at base + perBlock × interval.
	_, captured, cfg := piSP1Capture(t)
	command, _ := fakeProofCommand(t, "0x", "sleep 30")
	cfg.Command = command
	// The fixture range is 901-908 continuing from anchor 900: eight blocks.
	require.Equal(t, uint64(8), captured.candidate.LastBlock-captured.start.Continuation.Anchor.Number)
	cfg.Timeout, cfg.TimeoutPerBlock = 200*time.Millisecond, 25*time.Millisecond
	began := time.Now()
	_, err := newProjectionProducer(context.Background(), cfg)(t.Context(), captured.candidate, captured.privateData, captured.start)
	require.ErrorContains(t, err, "timed out after 400ms")
	require.Less(t, time.Since(began), 10*time.Second)
}

// TestOversizeProofRequestFailsBeforeSpawning: the 128 MiB stdin cap is enforced by the batcher.
func TestOversizeProofRequestFailsBeforeSpawning(t *testing.T) {
	_, captured, cfg := piSP1Capture(t)
	command, requestPath := fakeProofCommand(t, "0x", "")
	cfg.Command = command
	huge := make([]byte, MaxPublicationRequestBytes/2+1)
	_, err := newProjectionProducer(context.Background(), cfg)(t.Context(), captured.candidate, huge, captured.start)
	require.ErrorContains(t, err, "stdin cap")
	_, statErr := os.Stat(requestPath)
	require.True(t, os.IsNotExist(statErr), "the child was never started")
	require.False(t, strings.Contains(err.Error(), "proof command failed"))
}
