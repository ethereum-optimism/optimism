package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const configExportRPCTimeout = 2 * time.Second

type configExport struct {
	Dir      string
	Files    []configExportFile
	Warnings []string
}

type configExportFile struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Source string `json:"source,omitempty"`
}

type configExportManifest struct {
	Preset    string             `json:"preset"`
	CreatedAt string             `json:"created_at"`
	Files     []configExportFile `json:"files"`
	Warnings  []string           `json:"warnings,omitempty"`
}

type endpointConfig struct {
	Name       string `json:"name"`
	Layer      string `json:"layer"`
	Chain      string `json:"chain"`
	ChainID    string `json:"chain_id,omitempty"`
	ChainLabel string `json:"chain_label,omitempty"`
	URL        string `json:"url"`
	Proxied    bool   `json:"proxied"`
}

type contractSetConfig struct {
	Network   string                  `json:"network"`
	ChainID   string                  `json:"chain_id"`
	Contracts []contractAddressConfig `json:"contracts"`
}

type contractAddressConfig struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type rollupConfigIndexEntry struct {
	Network string `json:"network"`
	ChainID string `json:"chain_id"`
	Path    string `json:"path"`
}

func exportDevnetConfigs(ctx context.Context, cfg opUpConfig, spec *devnetPreset, tempRoot string, devnet *runningDevnet, endpoints []*localEndpoint) (*configExport, error) {
	configRoot := filepath.Join(cfg.Dir, "configs")
	if err := os.MkdirAll(configRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create config export root: %w", err)
	}
	if err := os.Chmod(configRoot, 0o700); err != nil {
		return nil, fmt.Errorf("restrict config export root: %w", err)
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)
	prefix := fmt.Sprintf("op-up-%s-%s-", tempDirSafePrefix(spec.Name), time.Now().UTC().Format("20060102T150405Z"))
	exportDir, err := os.MkdirTemp(configRoot, prefix)
	if err != nil {
		return nil, fmt.Errorf("create config export dir: %w", err)
	}
	if err := os.Chmod(exportDir, 0o700); err != nil {
		return nil, fmt.Errorf("restrict config export dir: %w", err)
	}

	out := &configExport{Dir: exportDir}
	addFile := func(kind string, relPath string, source string) {
		out.Files = append(out.Files, configExportFile{
			Path:   filepath.ToSlash(relPath),
			Kind:   kind,
			Source: source,
		})
	}

	if err := writeJSONConfig(exportDir, "endpoints.json", endpointsForConfig(endpoints)); err != nil {
		return nil, err
	}
	addFile("endpoints", "endpoints.json", "op-up")

	if err := writeJSONConfig(exportDir, "contracts.json", contractsForConfig(devnet.Contracts)); err != nil {
		return nil, err
	}
	addFile("contracts", "contracts.json", "op-up")

	rollupIndex, err := exportRollupConfigs(exportDir, devnet.L2Networks, addFile)
	if err != nil {
		return nil, err
	}
	if err := writeJSONConfig(exportDir, "rollups/index.json", rollupIndex); err != nil {
		return nil, err
	}
	addFile("rollup-index", "rollups/index.json", "op-up")

	if devnet.ExportDepset {
		if err := exportDependencySets(ctx, exportDir, endpoints, addFile); err != nil {
			return nil, err
		}
	}
	if err := copyGeneratedJSONConfigs(tempRoot, exportDir, addFile); err != nil {
		return nil, err
	}

	manifestFile := configExportFile{Path: "manifest.json", Kind: "manifest", Source: "op-up"}
	manifestFiles := append([]configExportFile{}, out.Files...)
	manifestFiles = append(manifestFiles, manifestFile)
	manifest := configExportManifest{
		Preset:    spec.Name,
		CreatedAt: createdAt,
		Files:     manifestFiles,
		Warnings:  out.Warnings,
	}
	if err := writeJSONConfig(exportDir, "manifest.json", manifest); err != nil {
		return nil, err
	}
	out.Files = manifestFiles
	return out, nil
}

func endpointsForConfig(endpoints []*localEndpoint) []endpointConfig {
	out := make([]endpointConfig, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint == nil || endpoint.devnetEndpoint == nil {
			continue
		}
		chainID := ""
		if endpoint.ChainID != (eth.ChainID{}) {
			chainID = endpoint.ChainID.String()
		}
		out = append(out, endpointConfig{
			Name:       endpoint.Name,
			Layer:      endpoint.Layer,
			Chain:      endpoint.Chain(),
			ChainID:    chainID,
			ChainLabel: endpoint.ChainLabel,
			URL:        endpoint.LocalURL,
			Proxied:    endpoint.Listener != nil,
		})
	}
	return out
}

func contractsForConfig(sets []*contractSet) []contractSetConfig {
	out := make([]contractSetConfig, 0, len(sets))
	for _, set := range sets {
		if set == nil {
			continue
		}
		contracts := make([]contractAddressConfig, 0, len(set.Contracts))
		for _, contract := range set.Contracts {
			contracts = append(contracts, contractAddressConfig{
				Name:    contract.Name,
				Address: contract.Address.Hex(),
			})
		}
		out = append(out, contractSetConfig{
			Network:   set.Network,
			ChainID:   set.ChainID.String(),
			Contracts: contracts,
		})
	}
	return out
}

func exportRollupConfigs(exportDir string, networks []*namedL2Network, addFile func(kind string, relPath string, source string)) ([]rollupConfigIndexEntry, error) {
	out := make([]rollupConfigIndexEntry, 0, len(networks))
	for _, network := range networks {
		if network == nil || network.Network == nil {
			continue
		}
		chainID := network.Network.ChainID()
		relPath := filepath.Join("rollups", fmt.Sprintf("rollup-%s-%s.json", configFilenameComponent(network.Name), chainID))
		if err := writeJSONConfig(exportDir, relPath, network.Network.Escape().RollupConfig()); err != nil {
			return nil, err
		}
		relPath = filepath.ToSlash(relPath)
		addFile("rollup", relPath, "op-devstack dsl")
		out = append(out, rollupConfigIndexEntry{
			Network: network.Name,
			ChainID: chainID.String(),
			Path:    relPath,
		})
	}
	return out, nil
}

func exportDependencySets(ctx context.Context, exportDir string, endpoints []*localEndpoint, addFile func(kind string, relPath string, source string)) error {
	seen := make(map[string]struct{})
	written := 0
	for _, endpoint := range endpoints {
		if endpoint == nil || endpoint.RPC == nil || !shouldFetchDependencySet(endpoint) {
			continue
		}
		raw, ok := fetchDependencySet(ctx, endpoint)
		if !ok {
			continue
		}
		key := string(raw)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		relPath := "depset.json"
		if written > 0 {
			relPath = filepath.Join("depsets", fmt.Sprintf("depset-%s-%s.json", configFilenameComponent(endpoint.Name), configFilenameComponent(endpoint.Chain())))
		}
		if err := writeRawJSONConfig(exportDir, relPath, raw); err != nil {
			return err
		}
		addFile("dependency-set", relPath, "optimism_dependencySet")
		written++
	}
	return nil
}

func shouldFetchDependencySet(endpoint *localEndpoint) bool {
	return endpoint.Layer == "CL" || endpoint.Layer == "RPC" || endpoint.ChainLabel == "shared"
}

func fetchDependencySet(ctx context.Context, endpoint *localEndpoint) (json.RawMessage, bool) {
	callCtx, cancel := context.WithTimeout(ctx, configExportRPCTimeout)
	defer cancel()

	var raw json.RawMessage
	if err := endpoint.RPC.CallContext(callCtx, &raw, "optimism_dependencySet"); err != nil {
		return nil, false
	}
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, false
	}
	return raw, true
}

func copyGeneratedJSONConfigs(tempRoot string, exportDir string, addFile func(kind string, relPath string, source string)) error {
	generatedRoot := filepath.Join(exportDir, "generated")
	return filepath.WalkDir(tempRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !isGeneratedConfigJSON(entry.Name()) {
			return nil
		}
		rel, err := filepath.Rel(tempRoot, path)
		if err != nil {
			return fmt.Errorf("relativize generated config %s: %w", path, err)
		}
		dest := filepath.Join(generatedRoot, rel)
		if err := copyFile(path, dest); err != nil {
			return err
		}
		relDest := filepath.ToSlash(filepath.Join("generated", rel))
		addFile("generated-json", relDest, path)
		return nil
	})
}

func isGeneratedConfigJSON(name string) bool {
	if filepath.Ext(name) != ".json" {
		return false
	}
	name = strings.ToLower(name)
	return strings.Contains(name, "genesis") ||
		strings.Contains(name, "rollup") ||
		strings.Contains(name, "depset") ||
		strings.Contains(name, "config")
}

func writeJSONConfig(root string, relPath string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", relPath, err)
	}
	data = append(data, '\n')
	return writeConfigFile(root, relPath, data)
}

func writeRawJSONConfig(root string, relPath string, raw json.RawMessage) error {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		return fmt.Errorf("format %s: %w", relPath, err)
	}
	pretty.WriteByte('\n')
	return writeConfigFile(root, relPath, pretty.Bytes())
}

func writeConfigFile(root string, relPath string, data []byte) error {
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir for %s: %w", relPath, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", relPath, err)
	}
	return nil
}

func copyFile(src string, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return fmt.Errorf("create destination dir for %s: %w", dest, err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open generated config %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create generated config copy %s: %w", dest, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copy generated config %s: %w", src, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close generated config copy %s: %w", dest, err)
	}
	return nil
}

func configFilenameComponent(v string) string {
	v = strings.ToLower(tempDirSafePrefix(v))
	if v == "" {
		return "unknown"
	}
	return v
}
