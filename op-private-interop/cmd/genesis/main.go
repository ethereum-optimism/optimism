// Command genesis creates private ETH and public projection artifacts from one source deployment.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	privategenesis "github.com/ethereum-optimism/optimism/op-private-interop/genesis"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	source := flag.String("genesis", "", "source ETH genesis path or URL")
	rollupPath := flag.String("rollup", "", "matching source rollup JSON path")
	out := flag.String("out", "", "new output directory (must not exist)")
	baseURL := flag.String("artifact-base-url", "", "immutable HTTP(S) directory for NetChef overrides (optional)")
	flag.Parse()
	if *source == "" || *rollupPath == "" || *out == "" {
		return fmt.Errorf("--genesis, --rollup and --out are required")
	}
	g, err := privategenesis.LoadPrivateChainGenesis(context.Background(), *source)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(*rollupPath)
	if err != nil {
		return err
	}
	var cfg rollup.Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}
	private, privateCfg, err := privategenesis.ConfigurePrivateGenesis(g, &cfg)
	if err != nil {
		return err
	}
	projection, err := privategenesis.ProjectGenesisFrom(private)
	if err != nil {
		return err
	}
	projectionCfg, err := privategenesis.ProjectRollupConfigFrom(privateCfg, projection)
	if err != nil {
		return err
	}
	objects := map[string]any{
		"private-genesis.json":    private,
		"private-rollup.json":     privateCfg,
		"projection-genesis.json": projection,
		"projection-rollup.json":  projectionCfg,
	}
	if *baseURL != "" {
		if !strings.HasPrefix(*baseURL, "https://") && !strings.HasPrefix(*baseURL, "http://") {
			return fmt.Errorf("--artifact-base-url must be HTTP(S)")
		}
		genesisURL := strings.TrimRight(*baseURL, "/") + "/private-genesis.json"
		rollupBytes, err := json.Marshal(privateCfg)
		if err != nil {
			return err
		}
		encoded := base64.StdEncoding.EncodeToString(rollupBytes)
		// Existing runtime projection mode: all consumers receive the same private source.
		// The projection EL retains --rollup.private; the supernode transforms its rollup config.
		objects["netchef-chain-values.json"] = map[string]any{
			"op-reth":      map[string]any{"env": map[string]string{"RETH_GENESIS_URL": genesisURL}},
			"op-node":      map[string]string{"rollupConfig": encoded},
			"op-supernode": map[string]any{"chains": map[string]any{cfg.L2ChainID.String(): map[string]string{"rollupConfig": encoded}}},
		}
		objects["netchef-batcher-service-values.json"] = map[string]any{"env": map[string]string{
			"OP_BATCHER_PRIVATE_INTEROP_GENESIS": genesisURL,
		}}
		objects["netchef-supernode-service-values.json"] = map[string]any{"env": map[string]string{
			"OP_SUPERNODE_PRIVATE_INTEROP_GENESIS":  genesisURL,
			"OP_SUPERNODE_PRIVATE_INTEROP_CHAIN_ID": cfg.L2ChainID.String(),
		}}
	}
	files := make(map[string][]byte)
	digests := make(map[string]string)
	for name, object := range objects {
		data, err := json.MarshalIndent(object, "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		files[name] = data
		digests[name] = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	report := map[string]any{
		"profile":                 "private-eth-v1",
		"chainId":                 cfg.L2ChainID.String(),
		"sourceGenesisHash":       g.ToBlock().Hash(),
		"privateGenesisHash":      privateCfg.Genesis.L2.Hash,
		"projectionGenesisHash":   projectionCfg.Genesis.L2.Hash,
		"messengerCodeHash":       privategenesis.PolicyMessengerCodeHash,
		"bridgeCodeHash":          privategenesis.PolicyBridgeCodeHash,
		"requirePaidMessagesSlot": privategenesis.RequirePaidMessagesSlot,
		"nativeETHRoutes":         []string{},
		"l1BackingVerified":       false,
		"sha256":                  digests,
	}
	files["report.json"], err = json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(*out, 0755); err != nil {
		return fmt.Errorf("create new artifact directory: %w", err)
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(*out, name), data, 0644); err != nil {
			return err
		}
	}
	fmt.Printf("private %s\nprojection %s\nartifacts %s\n", privateCfg.Genesis.L2.Hash, projectionCfg.Genesis.L2.Hash, *out)
	return nil
}
