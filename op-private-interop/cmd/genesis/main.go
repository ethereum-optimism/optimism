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
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	privategenesis "github.com/ethereum-optimism/optimism/op-private-interop/genesis"
	piprojection "github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
	verifier := flag.String("verifier", piprojection.SP1PrivateProjectionV1, "projection verifier ID (sp1-private-projection-v1; the other IDs are test-only)")
	programVKey := flag.String("program-vkey", "", "SP1 program vkey of the private-projection guest (sp1 only; from the reproducible ELF build)")
	l1ConfigPath := flag.String("l1-chain-config", "", "L1 chain config JSON pinned into private_config_hash (sp1 only)")
	depSetFlag := flag.String("dependency-set", "", "comma-separated chain IDs of the private dependency set (default: the chain itself)")
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
	// private-rollup.json is written from exactly these bytes: private_config_hash covers them.
	privateRollupJSON, err := json.MarshalIndent(privateCfg, "", "  ")
	if err != nil {
		return err
	}
	privateRollupJSON = append(privateRollupJSON, '\n')
	opts := privategenesis.ProjectionOptions{Verifier: *verifier, PrivateRollupJSON: privateRollupJSON}
	for _, id := range strings.Split(*depSetFlag, ",") {
		if id = strings.TrimSpace(id); id == "" {
			continue
		}
		n, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			return fmt.Errorf("--dependency-set: %w", err)
		}
		opts.DependencySet = append(opts.DependencySet, eth.ChainIDFromUInt64(n))
	}
	if len(opts.DependencySet) == 0 {
		opts.DependencySet = []eth.ChainID{eth.ChainIDFromBig(cfg.L2ChainID)}
	}
	var l1ChainConfigJSON []byte
	if *verifier == piprojection.SP1PrivateProjectionV1 {
		if *programVKey == "" || *l1ConfigPath == "" {
			return fmt.Errorf("--program-vkey and --l1-chain-config are required for %s", piprojection.SP1PrivateProjectionV1)
		}
		opts.ProgramVKey = common.HexToHash(*programVKey)
		if l1ChainConfigJSON, err = os.ReadFile(*l1ConfigPath); err != nil {
			return err
		}
		opts.L1ChainConfigJSON = l1ChainConfigJSON
	}
	projectionCfg, err := privategenesis.ProjectRollupConfigFrom(privateCfg, private, projection, opts)
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
	delete(objects, "private-rollup.json")
	files := map[string][]byte{"private-rollup.json": privateRollupJSON}
	if l1ChainConfigJSON != nil {
		files["l1-chain-config.json"] = l1ChainConfigJSON
	}
	digests := make(map[string]string)
	for name, data := range files {
		digests[name] = fmt.Sprintf("%x", sha256.Sum256(data))
	}
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
		"profile":               "private-eth-v1",
		"chainId":               cfg.L2ChainID.String(),
		"sourceGenesisHash":     g.ToBlock().Hash(),
		"privateGenesisHash":    privateCfg.Genesis.L2.Hash,
		"projectionGenesisHash": projectionCfg.Genesis.L2.Hash,
		"messengerCodeHash":     privategenesis.PolicyMessengerCodeHash,
		"bridgeCodeHash":        privategenesis.PolicyBridgeCodeHash,
		"nativeETHRoutes":       []string{},
		"l1BackingVerified":     false,
		"sha256":                digests,
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
