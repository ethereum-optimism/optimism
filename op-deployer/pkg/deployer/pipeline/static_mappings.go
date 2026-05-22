package pipeline

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
)

const defaultLegacyInputMappingDir = "op-deployer-types/adapters/legacy"

func evaluateLegacyInputMapping(env *Env, mappingFile string, sources opcm.StaticInputSources) (opcm.ScriptInput, error) {
	mappingPath, err := legacyInputMappingPath(env, mappingFile)
	if err != nil {
		return nil, err
	}
	mapping, err := opcm.LoadStaticInputMappingYAMLFile(mappingPath)
	if err != nil {
		return nil, err
	}
	input, err := opcm.EvaluateStaticInputMapping(*mapping, sources)
	if err != nil {
		return nil, fmt.Errorf("evaluate %s: %w", mappingFile, err)
	}
	return input, nil
}

func legacyInputMappingPath(env *Env, mappingFile string) (string, error) {
	baseDir := ""
	if env != nil {
		baseDir = env.LegacyInputMappingDir
	}
	if baseDir == "" {
		repoRoot, err := repoRootFromThisFile()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(repoRoot, defaultLegacyInputMappingDir)
	}
	return filepath.Join(baseDir, mappingFile), nil
}

func repoRootFromThisFile() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot resolve repository root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..")), nil
}
