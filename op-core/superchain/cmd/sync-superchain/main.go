// sync-superchain reads the superchain-registry submodule at
// packages/contracts-bedrock/lib/superchain-registry and emits a deterministic
// op-core/superchain/superchain-configs.zip. The submodule's HEAD SHA is the
// single source of truth — it becomes the zip's COMMIT entry and is what
// VerifyEmbeddedCommit cross-checks against op-geth's embedded copy at runtime.
//
// Run via `just sync-superchain`. The output zip is gitignored.
package main

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const submodulePath = "packages/contracts-bedrock/lib/superchain-registry"

// zipEpoch is the timestamp baked into every zip entry. The ZIP format's epoch
// is 1980-01-01, so this is the earliest representable mtime.
var zipEpoch = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// skipChainIDs lists chains the embedded bundle must not include. Boba
// {Mainnet, Sepolia} have non-standard genesis and Celo Mainnet is a converted
// L1 (not a bedrock genesis), so the Go superchain package can't load them.
var skipChainIDs = map[uint64]bool{
	288:   true, // Boba Mainnet
	28882: true, // Boba Sepolia
	42220: true, // Celo Mainnet
}

type chainConfigTOML struct {
	ChainID uint64 `toml:"chain_id"`
}

type chainsIndexEntry struct {
	Name    string `json:"name"`
	Network string `json:"network"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sync-superchain:", err)
		os.Exit(1)
	}
}

func run() error {
	repoRoot, err := gitTopLevel()
	if err != nil {
		return fmt.Errorf("resolving monorepo root: %w", err)
	}

	submoduleDir := filepath.Join(repoRoot, submodulePath)
	if err := assertSubmodulePopulated(submoduleDir); err != nil {
		return err
	}

	submoduleCommit, err := gitHead(submoduleDir)
	if err != nil {
		return fmt.Errorf("reading submodule HEAD at %s: %w", submoduleDir, err)
	}

	outZip := filepath.Join(repoRoot, "op-core", "superchain", "superchain-configs.zip")
	if upToDate, err := zipAlreadyAt(outZip, submoduleCommit); err != nil {
		return fmt.Errorf("checking existing zip: %w", err)
	} else if upToDate {
		fmt.Printf("[sync-superchain] up to date at commit %s\n", submoduleCommit)
		return nil
	}

	configsDir := filepath.Join(submoduleDir, "superchain", "configs")
	genesisDir := filepath.Join(submoduleDir, "superchain", "extra", "genesis")
	dictionaryPath := filepath.Join(submoduleDir, "superchain", "extra", "dictionary")

	entries, index, err := collectChainEntries(configsDir, genesisDir)
	if err != nil {
		return err
	}

	dictionary, err := os.ReadFile(dictionaryPath)
	if err != nil {
		return fmt.Errorf("reading dictionary at %s: %w", dictionaryPath, err)
	}

	chainsJSON, err := encodeChainsJSON(index)
	if err != nil {
		return fmt.Errorf("encoding chains.json: %w", err)
	}

	if err := writeDeterministicZip(outZip, entries, dictionary, chainsJSON, submoduleCommit); err != nil {
		return fmt.Errorf("writing %s: %w", outZip, err)
	}

	fmt.Printf("[sync-superchain] wrote %s at commit %s\n", outZip, submoduleCommit)
	return nil
}

func gitTopLevel() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitHead(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

// assertSubmodulePopulated returns a directed error if the submodule directory
// is missing or empty (commonly because submodules haven't been initialised).
func assertSubmodulePopulated(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("superchain-registry submodule not found at %s: %w\nRun `just source` (or `git submodule update --init`) to populate it.", dir, err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("superchain-registry submodule at %s is empty.\nRun `just source` (or `git submodule update --init`) to populate it.", dir)
	}
	return nil
}

// zipAlreadyAt returns true when outZip exists and its COMMIT entry already
// matches expectedCommit, so the caller can skip regeneration.
func zipAlreadyAt(outZip, expectedCommit string) (bool, error) {
	f, err := os.Open(outZip)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return false, err
	}
	zr, err := zip.NewReader(f, info.Size())
	if err != nil {
		return false, nil // malformed zip: treat as stale, will be overwritten
	}
	commitFile, err := zr.Open("COMMIT")
	if err != nil {
		return false, nil
	}
	defer commitFile.Close()
	raw, err := io.ReadAll(commitFile)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(raw)) == expectedCommit, nil
}

type zipEntry struct {
	name string
	data []byte
}

// collectChainEntries walks <submodule>/superchain/configs/<network>/*.toml,
// pairs each chain config with its <submodule>/superchain/extra/genesis/<network>/<name>.json.zst,
// applies the skip-list, and returns the zip entries plus a chain_id →
// {name, network} index for chains.json.
//
// Per-superchain `superchain.toml` files (L1 config for the superchain itself)
// are included in the zip but not in the index — they're loaded by name at
// runtime, not by chain id.
func collectChainEntries(configsDir, genesisDir string) ([]zipEntry, map[string]chainsIndexEntry, error) {
	var entries []zipEntry
	index := make(map[string]chainsIndexEntry)

	networks, err := os.ReadDir(configsDir)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", configsDir, err)
	}
	for _, networkEnt := range networks {
		if !networkEnt.IsDir() {
			continue
		}
		network := networkEnt.Name()
		networkDir := filepath.Join(configsDir, network)

		chainFiles, err := os.ReadDir(networkDir)
		if err != nil {
			return nil, nil, fmt.Errorf("reading %s: %w", networkDir, err)
		}
		for _, chainFileEnt := range chainFiles {
			name := chainFileEnt.Name()
			if !strings.HasSuffix(name, ".toml") {
				continue
			}
			chainTOMLPath := filepath.Join(networkDir, name)
			tomlBytes, err := os.ReadFile(chainTOMLPath)
			if err != nil {
				return nil, nil, fmt.Errorf("reading %s: %w", chainTOMLPath, err)
			}

			// superchain.toml is the per-superchain L1 config; keep it in the
			// bundle but don't index it as a chain.
			if name == "superchain.toml" {
				entries = append(entries, zipEntry{name: "configs/" + network + "/" + name, data: tomlBytes})
				continue
			}

			var cfg chainConfigTOML
			if err := toml.Unmarshal(tomlBytes, &cfg); err != nil {
				return nil, nil, fmt.Errorf("parsing %s: %w", chainTOMLPath, err)
			}
			if cfg.ChainID == 0 || skipChainIDs[cfg.ChainID] {
				continue
			}

			chainName := strings.TrimSuffix(name, ".toml")
			genesisPath := filepath.Join(genesisDir, network, chainName+".json.zst")
			genesisBytes, err := os.ReadFile(genesisPath)
			if err != nil {
				return nil, nil, fmt.Errorf("reading genesis for %s/%s (chain id %d) at %s: %w", network, chainName, cfg.ChainID, genesisPath, err)
			}

			entries = append(entries,
				zipEntry{name: "configs/" + network + "/" + name, data: tomlBytes},
				zipEntry{name: "genesis/" + network + "/" + chainName + ".json.zst", data: genesisBytes},
			)
			index[strconv.FormatUint(cfg.ChainID, 10)] = chainsIndexEntry{
				Name:    chainName,
				Network: network,
			}
		}
	}
	return entries, index, nil
}

// encodeChainsJSON serialises the chain_id → entry index with sorted keys to
// keep the output byte-stable across runs.
func encodeChainsJSON(index map[string]chainsIndexEntry) ([]byte, error) {
	keys := make([]string, 0, len(index))
	for k := range index {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyJSON, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')
		valJSON, err := json.Marshal(index[k])
		if err != nil {
			return nil, err
		}
		buf.Write(valJSON)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// writeDeterministicZip writes outPath with sorted entries, fixed mtimes, mode
// 0o755, and DEFLATE level 9. Two runs against the same submodule + same script
// must produce byte-identical output.
func writeDeterministicZip(outPath string, perChain []zipEntry, dictionary, chainsJSON []byte, commit string) error {
	all := append([]zipEntry(nil), perChain...)
	all = append(all,
		zipEntry{name: "dictionary", data: dictionary},
		zipEntry{name: "chains.json", data: chainsJSON},
		zipEntry{name: "COMMIT", data: []byte(commit + "\n")},
	)
	sort.Slice(all, func(i, j int) bool { return all[i].name < all[j].name })

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(outPath), ".sync-superchain.*.zip")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	zw := zip.NewWriter(tmp)
	zw.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, flate.BestCompression)
	})

	for _, e := range all {
		hdr := &zip.FileHeader{
			Name:     e.name,
			Method:   zip.Deflate,
			Modified: zipEpoch,
		}
		hdr.SetMode(0o755)
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return fmt.Errorf("creating zip entry %s: %w", e.name, err)
		}
		if _, err := w.Write(e.data); err != nil {
			return fmt.Errorf("writing zip entry %s: %w", e.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, outPath)
}
