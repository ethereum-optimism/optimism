package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/nuts"
)

// Reporting is opt-in. Copy original bytes before the temporary worktree is
// removed; a failed generator must never label the historical snapshot as a
// successfully regenerated bundle.
func retainSource(root, worktree, report string, fork forks.Name, entry nuts.ForkLockEntry) error {
	if err := os.MkdirAll(report, 0755); err != nil {
		return err
	}
	commit, err := provenanceGit(worktree, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(commit)) != entry.Commit {
		return fmt.Errorf("worktree revision differs from recorded commit")
	}
	metadata, err := json.MarshalIndent(struct {
		Fork    forks.Name         `json:"fork"`
		Entry   nuts.ForkLockEntry `json:"entry"`
		Command []string           `json:"generator_argv"`
	}{fork, entry, []string{"just", "generate-nut-bundle"}}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(report, "source.json"), append(metadata, '\n'), 0644); err != nil {
		return err
	}
	for name, source := range map[string]string{
		"mise.toml":          filepath.Join(worktree, "mise.toml"),
		"contracts.justfile": filepath.Join(worktree, "packages/contracts-bedrock/justfile"),
		"foundry.toml":       filepath.Join(worktree, "packages/contracts-bedrock/foundry.toml"),
		"locked-bundle.json": filepath.Join(root, entry.Bundle),
	} {
		data, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(report, name), data, 0644); err != nil {
			return err
		}
	}
	data, err := provenanceGit(worktree, "ls-files", "--stage", "-z")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(report, "tracked-stage.bin"), data, 0644)
}

func retainGeneration(worktree, report string, generated bool) error {
	var modules []byte
	for name, args := range map[string][]string{
		"submodules.txt":      {"submodule", "status", "--recursive"},
		"worktree-status.txt": {"status", "--porcelain=v1", "--untracked-files=no"},
	} {
		data, err := provenanceGit(worktree, args...)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(report, name), data, 0644); err != nil {
			return err
		}
		if name == "submodules.txt" {
			modules = data
		}
	}
	// Keep the original index and dirty-state bytes for every initialized
	// submodule, including nested libraries. Uninitialized repositories remain
	// visible in submodules.txt; the contract generator does not use all root
	// repositories (for example op-rbuilder).
	type moduleEvidence struct {
		Head   string `json:"head"`
		Index  []byte `json:"tracked_stage"`
		Status []byte `json:"status"`
	}
	inventories := make(map[string]moduleEvidence)
	for _, line := range strings.Split(string(modules), "\n") {
		if line == "" || line[0] == '-' {
			continue
		}
		fields := strings.Fields(line[1:])
		if len(fields) < 2 || filepath.IsAbs(fields[1]) || strings.Contains(fields[1], "..") {
			return fmt.Errorf("invalid original submodule status: %q", line)
		}
		dir := filepath.Join(worktree, fields[1])
		head, err := provenanceGit(dir, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		index, err := provenanceGit(dir, "ls-files", "--stage", "-z")
		if err != nil {
			return err
		}
		status, err := provenanceGit(dir, "status", "--porcelain=v1", "--untracked-files=no")
		if err != nil {
			return err
		}
		inventories[fields[1]] = moduleEvidence{strings.TrimSpace(string(head)), index, status}
	}
	data, err := json.MarshalIndent(inventories, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(report, "submodule-inventories.json"), append(data, '\n'), 0644); err != nil {
		return err
	}
	if generated {
		data, err := os.ReadFile(filepath.Join(worktree, "packages/contracts-bedrock/snapshots/upgrades/current-upgrade-bundle.json"))
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(report, "regenerated-bundle.json"), data, 0644)
	}
	return nil
}

func provenanceGit(worktree string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = worktree
	cmd.Stderr = os.Stderr
	return cmd.Output()
}
