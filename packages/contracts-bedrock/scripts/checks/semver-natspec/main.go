package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

type ArtifactsWrapper struct {
	RawMetadata string `json:"rawMetadata"`
}

type Artifacts struct {
	Output struct {
		Devdoc struct {
			StateVariables struct {
				Version struct {
					Semver string `json:"custom:semver"`
				} `json:"version"`
			} `json:"stateVariables,omitempty"`
			Methods struct {
				Version struct {
					Semver string `json:"custom:semver"`
				} `json:"version()"`
			} `json:"methods,omitempty"`
		} `json:"devdoc"`
	} `json:"output"`
}

var ConstantVersionPattern = regexp.MustCompile(`string.*constant.*version\s+=\s+"([^"]+)";`)

var FunctionVersionPattern = regexp.MustCompile(`^\s+return\s+"((?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?)";$`)

var InteropVersionPattern = regexp.MustCompile(`^\s+return\s+string\.concat\(super\.version\(\), "((.*)\+interop(.*)?)"\);`)

func main() {
	if err := run(); err != nil {
		writeStderr("an error occurred: %v", err)
		os.Exit(1)
	}
}

func writeStderr(msg string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, msg+"\n", args...)
}

func run() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	writeStderr("working directory: %s", cwd)

	artifactsDir := filepath.Join(cwd, "forge-artifacts")
	srcDir := filepath.Join(cwd, "src")

	artifactFiles, err := glob(artifactsDir, ".json")
	if err != nil {
		return fmt.Errorf("failed to get artifact files: %w", err)
	}
	contractFiles, err := glob(srcDir, ".sol")
	if err != nil {
		return fmt.Errorf("failed to get contract files: %w", err)
	}

	var hasErr int32
	var outMtx sync.Mutex
	fail := func(msg string, args ...any) {
		outMtx.Lock()
		writeStderr("❌  "+msg, args...)
		outMtx.Unlock()
		atomic.StoreInt32(&hasErr, 1)
	}

	sem := make(chan struct{}, runtime.NumCPU())
	for contractName, artifactPath := range artifactFiles {
		contractName := contractName
		artifactPath := artifactPath

		sem <- struct{}{}

		go func() {
			defer func() {
				<-sem
			}()

			af, err := os.Open(artifactPath)
			if err != nil {
				fail("%s: failed to open contract artifact: %v", contractName, err)
				return
			}
			defer af.Close()

			var wrapper ArtifactsWrapper
			if err := json.NewDecoder(af).Decode(&wrapper); err != nil {
				fail("%s: failed to parse artifact file: %v", contractName, err)
				return
			}

			if wrapper.RawMetadata == "" {
				return
			}

			var artifactData Artifacts
			if err := json.Unmarshal([]byte(wrapper.RawMetadata), &artifactData); err != nil {
				fail("%s: failed to unwrap artifact metadata: %v", contractName, err)
				return
			}

			artifactVersion := artifactData.Output.Devdoc.StateVariables.Version.Semver

			isConstant := true
			if artifactData.Output.Devdoc.StateVariables.Version.Semver == "" {
				artifactVersion = artifactData.Output.Devdoc.Methods.Version.Semver
				isConstant = false
			}

			if artifactVersion == "" {
				return
			}

			// Skip mock contracts
			if strings.HasPrefix(contractName, "Mock") {
				return
			}

			contractPath := contractFiles[contractName]
			if contractPath == "" {
				fail("%s: Source file not found (For test mock contracts, prefix the name with 'Mock' to ignore this warning)", contractName)
				return
			}

			cf, err := os.Open(contractPath)
			if err != nil {
				fail("%s: failed to open contract source: %v", contractName, err)
				return
			}
			defer cf.Close()

			sourceData, err := io.ReadAll(cf)
			if err != nil {
				fail("%s: failed to read contract source: %v", contractName, err)
				return
			}

			var sourceVersion string

			if isConstant {
				sourceVersion = findLine(sourceData, ConstantVersionPattern)
			} else {
				sourceVersion = findLine(sourceData, FunctionVersionPattern)
			}

			// Need to define a special case for interop contracts since they technically
			// use an invalid semver format. Checking for sourceVersion == "" allows the
			// team to update the format to a valid semver format in the future without
			// needing to change this program.
			if sourceVersion == "" && strings.HasSuffix(contractName, "Interop") {
				sourceVersion = findLine(sourceData, InteropVersionPattern)
			}

			if sourceVersion == "" {
				fail("%s: version not found in source", contractName)
				return
			}

			if sourceVersion != artifactVersion {
				fail("%s: version mismatch: source=%s, artifact=%s", contractName, sourceVersion, artifactVersion)
				return
			}

			_, _ = fmt.Fprintf(os.Stderr, "✅  %s: code: %s, artifact: %s\n", contractName, sourceVersion, artifactVersion)
		}()
	}

	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}

	if atomic.LoadInt32(&hasErr) == 1 {
		return fmt.Errorf("semver check failed, see logs above")
	}

	return nil
}

func glob(dir string, ext string) (map[string]string, error) {
	out := make(map[string]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && filepath.Ext(path) == ext {
			out[strings.TrimSuffix(filepath.Base(path), ext)] = path
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}
	return out, nil
}

func findLine(in []byte, pattern *regexp.Regexp) string {
	scanner := bufio.NewScanner(bytes.NewReader(in))
	for scanner.Scan() {
		match := pattern.FindStringSubmatch(scanner.Text())
		if len(match) > 0 {
			return match[1]
		}
	}
	return ""
}
