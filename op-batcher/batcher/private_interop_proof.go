package batcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-private-interop/builder"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// nativeProjectionProducer is operator-side I/O. Consensus verification remains
// pure and does not call this command, an RPC, or the private execution client.
// No witness files are created and no paid proving backend is selected.
func nativeProjectionProducer(lifecycle context.Context, command, privateRPC, publicRPC string,
	privateCfg, publicCfg *rollup.Config, dependencySet []byte,
) func(context.Context, *builder.BuiltRange, []byte, RangeStart) ([]byte, error) {
	return func(jobCtx context.Context, candidate *builder.BuiltRange, privateData []byte, start RangeStart) ([]byte, error) {
		statement, err := projection.ValidateProjectionRange(publicCfg.PrivateProjection, projection.Context{
			ChainID: publicCfg.L2ChainID, GenesisNumber: publicCfg.Genesis.L2.Number,
			GenesisTime: publicCfg.Genesis.L2Time, BlockTime: publicCfg.BlockTime,
			ParentHash: start.PrevTerminalRenderingHash, Continuation: start.Continuation,
		}, candidate.SpanBatch, projection.StubVerifier{})
		if err != nil {
			return nil, err
		}
		privateJSON, err := json.Marshal(privateCfg)
		if err != nil {
			return nil, err
		}
		publicJSON, err := json.Marshal(publicCfg)
		if err != nil {
			return nil, err
		}
		type block struct {
			Timestamp    uint64          `json:"timestamp"`
			Epoch        uint64          `json:"epoch"`
			Transactions []hexutil.Bytes `json:"transactions"`
		}
		request := struct {
			PrivateRPC       string        `json:"private_rpc"`
			ProjectionRPC    string        `json:"projection_rpc"`
			PrivateConfig    hexutil.Bytes `json:"private_config"`
			ProjectionConfig hexutil.Bytes `json:"projection_config"`
			DependencySet    hexutil.Bytes `json:"dependency_set"`
			ParentHash       common.Hash   `json:"parent_hash"`
			Anchor           eth.BlockID   `json:"anchor"`
			AnchorOutput     common.Hash   `json:"anchor_output"`
			Blocks           []block       `json:"blocks"`
			PrivateData      hexutil.Bytes `json:"private_data"`
			ExpectedDigest   common.Hash   `json:"expected_digest"`
		}{
			PrivateRPC: privateRPC, ProjectionRPC: publicRPC,
			PrivateConfig: privateJSON, ProjectionConfig: publicJSON, DependencySet: dependencySet,
			ParentHash: start.PrevTerminalRenderingHash, Anchor: start.Continuation.Anchor,
			AnchorOutput: start.Continuation.OutputRoot, PrivateData: privateData,
			ExpectedDigest: projection.AdmissionDigest(*statement),
		}
		for _, b := range candidate.Blocks {
			request.Blocks = append(request.Blocks, block{b.Timestamp, b.Origin.Number, b.Txs})
		}
		input, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(jobCtx, 2*time.Minute)
		stop := context.AfterFunc(lifecycle, cancel)
		defer stop()
		defer cancel()
		// Exact executable path, never shell interpolation. Sensitive witness
		// input is passed on stdin, never argv, logs, or a persistent file.
		cmd := exec.CommandContext(ctx, command, "--publication-request")
		cmd.Stdin = bytes.NewReader(input)
		var stdout, stderr proofCommandOutput
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("native projection checker failed: %w: %s", err, stderr.String())
		}
		proof, err := hexutil.Decode(strings.TrimSpace(stdout.String()))
		if err != nil {
			return nil, fmt.Errorf("native checker returned malformed proof envelope: %w", err)
		}
		if err := (projection.ExecutionMockVerifier{}).Verify(*statement, proof); err != nil {
			return nil, err
		}
		return proof, nil
	}
}

// Bound diagnostics from a failing child without blocking its pipe.
type proofCommandOutput struct{ bytes.Buffer }

func (b *proofCommandOutput) Write(p []byte) (int, error) {
	n := len(p)
	if remaining := 16384 - b.Len(); remaining > 0 {
		_, _ = b.Buffer.Write(p[:min(n, remaining)])
	}
	return n, nil
}
