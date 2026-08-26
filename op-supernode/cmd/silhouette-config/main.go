// Command silhouette-config emits a silhouette chain's rollup config from the handful of values an
// operator actually decides.
//
// It exists because the rollup config is OUR artifact, not a registry entry, and the three ways it
// differs from a stock chain's — a finite sequencing window, a deposit contract that reverts, every
// fork active at genesis — are properties a hand-edited JSON file loses silently. Generating it runs
// checkSilhouetteInvariants over the result, so a config that would break the forced-extension
// convention or the guest's headers-only L1 walk fails here rather than in a proof six hours later.
//
// The output is the file the verifier, the sequencer's op-node and the guest all read, and its
// sha256-of-canonical-JSON is the rollupConfigHash the wire binds. See the rotation runbook for how
// that hash is computed (it is kona's serialization of the PARSED config, so it is computed by the
// guest, not here — a second implementation of a consensus-critical hash is exactly what we do not
// want).
package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/silhouette"
)

func main() {
	app := cli.NewApp()
	app.Name = "silhouette-config"
	app.Usage = "Generate a silhouette chain's rollup config"
	app.Flags = []cli.Flag{
		&cli.Uint64Flag{Name: "l2-chain-id", Required: true},
		&cli.Uint64Flag{Name: "l1-chain-id", Required: true},
		&cli.StringFlag{Name: "l1-genesis-hash", Required: true, Usage: "L1 block the chain starts from"},
		&cli.Uint64Flag{Name: "l1-genesis-number", Required: true},
		&cli.StringFlag{Name: "l2-genesis-hash", Required: true, Usage: "P's own genesis block hash"},
		&cli.Uint64Flag{Name: "l2-genesis-number", Value: 0},
		&cli.Uint64Flag{Name: "l2-time", Required: true, Usage: "L2 genesis timestamp"},
		&cli.Uint64Flag{Name: "block-time", Value: 2},
		&cli.Uint64Flag{Name: "seq-window", Value: silhouette.DefaultSeqWindowSize,
			Usage: "FINITE sequencing window in L1 blocks (DR-2)"},
		&cli.Uint64Flag{Name: "max-sequencer-drift", Value: 600},
		&cli.StringFlag{Name: "gated-portal", Required: true,
			Usage: "the deployed-but-gated OptimismPortal: real address, reverts on deposit"},
		&cli.StringFlag{Name: "batch-inbox", Required: true,
			Usage: "retained for shape only; a silhouette chain has no batcher"},
		&cli.StringFlag{Name: "system-config", Required: true, Usage: "L1 SystemConfig proxy (FROZEN)"},
		&cli.StringFlag{Name: "batcher-addr", Required: true},
		&cli.Uint64Flag{Name: "gas-limit", Required: true},
		&cli.StringFlag{Name: "eip1559-params", Value: "0x0000000000000000",
			Usage: "8 bytes: be32(denominator) || be32(elasticity). All-zero means chain defaults, " +
				"which the generator refuses -- a silhouette chain must state them (G2 F6)"},
		&cli.Uint64Flag{Name: "min-base-fee", Value: 0},
		&cli.Uint64Flag{Name: "base-fee-scalar", Required: true},
		&cli.Uint64Flag{Name: "blob-base-fee-scalar", Value: 0},
		&cli.StringFlag{Name: "out", Usage: "write here instead of stdout"},
	}
	app.Action = run
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx *cli.Context) error {
	var params eth.Bytes8
	raw := common.FromHex(ctx.String("eip1559-params"))
	if len(raw) != 8 {
		return fmt.Errorf("eip1559-params must be exactly 8 bytes, got %d", len(raw))
	}
	copy(params[:], raw)

	cfg, err := silhouette.RollupConfigFor(silhouette.SilhouetteParams{
		L2ChainID: new(big.Int).SetUint64(ctx.Uint64("l2-chain-id")),
		L1ChainID: new(big.Int).SetUint64(ctx.Uint64("l1-chain-id")),
		L1Genesis: eth.BlockID{
			Hash:   common.HexToHash(ctx.String("l1-genesis-hash")),
			Number: ctx.Uint64("l1-genesis-number"),
		},
		L2Genesis: eth.BlockID{
			Hash:   common.HexToHash(ctx.String("l2-genesis-hash")),
			Number: ctx.Uint64("l2-genesis-number"),
		},
		L2Time:            ctx.Uint64("l2-time"),
		BlockTime:         ctx.Uint64("block-time"),
		SeqWindowSize:     ctx.Uint64("seq-window"),
		MaxSequencerDrift: ctx.Uint64("max-sequencer-drift"),
		GatedPortal:       common.HexToAddress(ctx.String("gated-portal")),
		BatchInbox:        common.HexToAddress(ctx.String("batch-inbox")),
		SystemConfigProxy: common.HexToAddress(ctx.String("system-config")),
		BatcherAddr:       common.HexToAddress(ctx.String("batcher-addr")),
		GasLimit:          ctx.Uint64("gas-limit"),
		EIP1559Params:     params,
		MinBaseFee:        ctx.Uint64("min-base-fee"),
		BaseFeeScalar:     uint32(ctx.Uint64("base-fee-scalar")),
		BlobBaseFeeScalar: uint32(ctx.Uint64("blob-base-fee-scalar")),
	})
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if p := ctx.String("out"); p != "" {
		if err := os.WriteFile(p, out, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", p)
		return nil
	}
	_, err = os.Stdout.Write(out)
	return err
}
