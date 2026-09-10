// Command proofbatch-inspect decodes a proof-batch envelope and prints what it is bound to.
//
// It exists for one runbook step and it is worth naming which: the wire-binding cross-check. A
// verifier whose configured `rollupConfigHash` or `depSetHash` disagrees with the wire rejects every
// batch, loudly and forever, and the log line says so — but "why is nothing accepting" and "the
// configured value is one character different from the posted one" are separated by a decode, and
// having to write that decode during a cutover is how a five-minute answer becomes an hour.
//
// It reads files, not L1. That is deliberate: this tool must be runnable against a blob saved from a
// beacon node, a fixture, or a submitter's own output, without credentials and without a chance of
// touching a live cluster. The runbook's §10.2 pairs it with one curl.
//
// It VERIFIES NOTHING about the proof. Checking a proof is the verifier's job and needs a proving
// system this tool has no part of, and a tool that half-checked would be worse than one that plainly
// does not: everything printed here is a claim read off the wire, and the whole value of the output
// is that it says what the PRODUCER asserted rather than what anybody believes.
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

func main() {
	app := cli.NewApp()
	app.Name = "proofbatch-inspect"
	app.Usage = "Decode a proof-batch envelope and print its wire bindings"
	app.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name: "blob",
			Usage: "path to a blob sidecar, as saved from a beacon node's blob_sidecars endpoint " +
				"(hex with or without 0x, or raw 131072 bytes). Repeat in posting order for a " +
				"multi-blob batch.",
		},
		&cli.StringFlag{
			Name: "envelope",
			Usage: "path to raw envelope bytes (hex or binary), for a batch already lifted out of " +
				"its blobs. '-' reads stdin.",
		},
		&cli.BoolFlag{Name: "json", Usage: "emit machine-readable JSON instead of the report"},
		&cli.BoolFlag{
			Name: "blocks",
			Usage: "list every block, not just the range. Off by default: a 300-block batch is the " +
				"normal case and the bindings are what the check is about.",
		},
	}
	app.Action = run
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "proofbatch-inspect:", err)
		os.Exit(1)
	}
}

func run(ctx *cli.Context) error {
	blobPaths := ctx.StringSlice("blob")
	envPath := ctx.String("envelope")
	switch {
	case len(blobPaths) == 0 && envPath == "":
		return fmt.Errorf("nothing to inspect: pass --blob (repeatable) or --envelope")
	case len(blobPaths) > 0 && envPath != "":
		return fmt.Errorf("pass either --blob or --envelope, not both: they are two encodings of " +
			"the same bytes and accepting both at once would leave which one was read ambiguous")
	}

	var raw []byte
	var err error
	if envPath != "" {
		raw, err = readMaybeHex(envPath)
		if err != nil {
			return err
		}
	} else {
		blobs := make([]*eth.Blob, 0, len(blobPaths))
		for _, p := range blobPaths {
			b, err := readBlob(p)
			if err != nil {
				return err
			}
			blobs = append(blobs, b)
		}
		// FromBlobs is the submitter's own inverse, so the framing — length prefix, ordering, padding
		// — is read by the same code that wrote it rather than by a second implementation here.
		raw, err = proofbatch.FromBlobs(blobs)
		if err != nil {
			return fmt.Errorf("lift the envelope out of %d blob(s): %w", len(blobs), err)
		}
	}

	// DecodeAny, not Decode: an inspector that could only read the codec's CURRENT wire version would
	// be unable to read the version the live chain is actually running, which is precisely when
	// somebody reaches for it. Acceptance is the opposite — there the version is part of the rule.
	env, err := proofbatch.DecodeAny(raw)
	if err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	rep := describe(env)
	if ctx.Bool("json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	}
	printReport(rep, ctx.Bool("blocks"))
	return nil
}

// blockLine is one exported block, as much of it as is worth printing.
type blockLine struct {
	Number     uint64      `json:"number"`
	Timestamp  uint64      `json:"timestamp"`
	Hash       common.Hash `json:"hash"`
	Logs       int         `json:"logs"`
	OutputRoot common.Hash `json:"outputRoot"`
}

// report is everything this tool has to say. The field order is the order a cutover reads them in:
// the two bindings that are being cross-checked first, then the chaining, then the shape.
type report struct {
	RollupConfigHash common.Hash `json:"rollupConfigHash"`
	DepSetHash       common.Hash `json:"depSetHash"`
	ExportPolicyHash common.Hash `json:"exportPolicyHash"`
	// ExportPolicyIsAllHashes says whether the policy hash is the default "export every log" one, so
	// a reader does not have to know that constant by sight.
	ExportPolicyIsAllHashes bool        `json:"exportPolicyIsAllHashes"`
	L1Head                  common.Hash `json:"l1Head"`
	PrevOutputRoot          common.Hash `json:"prevOutputRoot"`
	NewOutputRoot           common.Hash `json:"newOutputRoot"`
	// NewOutputRootDerives is whether the last block's committed roots really produce NewOutputRoot.
	// The verifier checks this too (acceptance rule 2); it is repeated here because it is the one
	// self-consistency check that needs no configuration at all, so a false here means the batch is
	// malformed rather than merely mismatched with this node.
	NewOutputRootDerives bool   `json:"newOutputRootDerives"`
	FirstBlock           uint64 `json:"firstBlock"`
	LastBlock            uint64 `json:"lastBlock"`
	BlockCount           int    `json:"blockCount"`
	ExportedLogs         int    `json:"exportedLogs"`
	// StructureError is CheckStructure's verdict: contiguity, advancing timestamps, ascending log
	// indices. Empty means it passed.
	StructureError string `json:"structureError,omitempty"`
	// ProofBytes is 0 for an attested batch, which carries no proof at all. It is reported as a size
	// rather than as a mode because the MODE is a verifier's configuration and this is the wire: the
	// honest statement is how many bytes of proof were posted.
	ProofBytes int `json:"proofBytes"`
	// PublicValuesBytes and PublicValuesKeccak identify the exact bytes a proof would commit to.
	PublicValuesBytes  int         `json:"publicValuesBytes"`
	PublicValuesKeccak common.Hash `json:"publicValuesKeccak"`
	Blocks             []blockLine `json:"blocks"`
}

func describe(env *proofbatch.Envelope) *report {
	b := &env.Batch
	rep := &report{
		RollupConfigHash:        b.RollupConfigHash,
		DepSetHash:              b.DepSetHash,
		ExportPolicyHash:        b.ExportPolicyHash,
		ExportPolicyIsAllHashes: b.ExportPolicyHash == proofbatch.ExportPolicyAllHashes,
		L1Head:                  b.L1Head,
		PrevOutputRoot:          b.PrevOutputRoot,
		NewOutputRoot:           b.NewOutputRoot,
		BlockCount:              len(b.Blocks),
		ProofBytes:              len(env.Proof),
		PublicValuesBytes:       len(env.PublicValues),
		PublicValuesKeccak:      crypto.Keccak256Hash(env.PublicValues),
	}
	if err := b.CheckStructure(); err != nil {
		rep.StructureError = err.Error()
	}
	if len(b.Blocks) > 0 {
		rep.FirstBlock = b.Blocks[0].Number
		last := b.Blocks[len(b.Blocks)-1]
		rep.LastBlock = last.Number
		rep.NewOutputRootDerives = last.OutputRoot() == b.NewOutputRoot
	}
	for i := range b.Blocks {
		blk := &b.Blocks[i]
		rep.ExportedLogs += len(blk.Logs)
		rep.Blocks = append(rep.Blocks, blockLine{
			Number: blk.Number, Timestamp: blk.Timestamp, Hash: blk.Hash,
			Logs: len(blk.Logs), OutputRoot: blk.OutputRoot(),
		})
	}
	return rep
}

func printReport(rep *report, withBlocks bool) {
	p := func(k string, v any) { fmt.Printf("%-24s %v\n", k+":", v) }
	fmt.Println("wire bindings — cross-check these against the verifier config")
	p("rollupConfigHash", rep.RollupConfigHash)
	p("depSetHash", rep.DepSetHash)
	policy := rep.ExportPolicyHash.String()
	if rep.ExportPolicyIsAllHashes {
		policy += "  (the default: every log exported)"
	}
	p("exportPolicyHash", policy)
	fmt.Println()
	fmt.Println("chaining")
	p("prevOutputRoot", rep.PrevOutputRoot)
	p("newOutputRoot", rep.NewOutputRoot)
	derives := "yes"
	if !rep.NewOutputRootDerives {
		derives = "NO — the last block's committed roots do not produce it; this batch is malformed"
	}
	p("newOutputRoot derives", derives)
	p("l1Head", rep.L1Head)
	fmt.Println()
	fmt.Println("shape")
	p("blocks", fmt.Sprintf("%d  (%d..%d)", rep.BlockCount, rep.FirstBlock, rep.LastBlock))
	p("exported logs", rep.ExportedLogs)
	if rep.StructureError != "" {
		p("structure", "INVALID — "+rep.StructureError)
	} else {
		p("structure", "ok")
	}
	if rep.ProofBytes == 0 {
		p("proof", "0 bytes — ATTESTED: no proof was posted. A verifier configured with "+
			"proofType: attested requires exactly this, and rests the batch on the submitter's "+
			"L1 signature; one configured for a proving system will reject it")
	} else {
		p("proof", fmt.Sprintf("%d bytes", rep.ProofBytes))
	}
	p("publicValues", fmt.Sprintf("%d bytes, keccak %s", rep.PublicValuesBytes, rep.PublicValuesKeccak))
	if withBlocks {
		fmt.Println()
		fmt.Println("blocks")
		for _, b := range rep.Blocks {
			fmt.Printf("  %8d  t=%d  %s  logs=%d  out=%s\n",
				b.Number, b.Timestamp, b.Hash, b.Logs, b.OutputRoot)
		}
	}
}

// readMaybeHex reads a file that may be hex text or raw bytes.
//
// Both, because the two ways an operator gets these bytes produce different encodings: a beacon
// node's JSON gives hex, and a fixture or a pipe gives binary. Sniffing beats a flag, since guessing
// wrong is immediately obvious (the decode fails) rather than subtly wrong.
func readMaybeHex(path string) ([]byte, error) {
	var raw []byte
	var err error
	if path == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if decoded, ok := decodeHex(raw); ok {
		return decoded, nil
	}
	return raw, nil
}

func decodeHex(raw []byte) ([]byte, bool) {
	s := strings.TrimSpace(string(raw))
	s = strings.TrimPrefix(s, "0x")
	if s == "" || len(s)%2 != 0 {
		return nil, false
	}
	decoded, err := hex.DecodeString(s)
	if err != nil {
		return nil, false
	}
	return decoded, true
}

// readBlob reads one blob sidecar. A blob is exactly eth.BlobSize bytes, and a file that is not is
// refused rather than padded: padding a short blob would silently invent field elements, and the
// framing FromBlobs reads is inside those bytes.
func readBlob(path string) (*eth.Blob, error) {
	data, err := readMaybeHex(path)
	if err != nil {
		return nil, err
	}
	// A beacon sidecar is often delivered as a JSON object with the blob under "blob".
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '{' {
		var sidecar struct {
			Blob string `json:"blob"`
			Data struct {
				Blob string `json:"blob"`
			} `json:"data"`
		}
		if err := json.Unmarshal(trimmed, &sidecar); err != nil {
			return nil, fmt.Errorf("%s looks like JSON but does not parse: %w", path, err)
		}
		field := sidecar.Blob
		if field == "" {
			field = sidecar.Data.Blob
		}
		if field == "" {
			return nil, fmt.Errorf("%s is JSON with no \"blob\" field: pass one sidecar object, or "+
				"extract the blob with `jq -r '.data[N].blob'`", path)
		}
		decoded, ok := decodeHex([]byte(field))
		if !ok {
			return nil, fmt.Errorf("%s: the \"blob\" field is not hex", path)
		}
		data = decoded
	}
	if len(data) != eth.BlobSize {
		return nil, fmt.Errorf("%s is %d bytes; a blob is exactly %d", path, len(data), eth.BlobSize)
	}
	var blob eth.Blob
	copy(blob[:], data)
	return &blob, nil
}
