package bundle

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

const testCompileCache = `{
  "paths": {"artifacts": "forge-artifacts", "build_infos": "artifacts/build-info", "sources": "src", "libraries": ["lib"]},
  "files": {"src/A.sol": {}, "lib/dep/src/Used.sol": {}}
}`

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
}

// readArchive returns the regular files of a tar archive by name, and the names of its directories.
func readArchive(t *testing.T, data []byte) (map[string]string, []string) {
	t.Helper()
	files := make(map[string]string)
	var dirs []string
	tr := tar.NewReader(bytes.NewReader(data))
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return files, dirs
		}
		require.NoError(t, err)
		if hdr.Typeflag == tar.TypeDir {
			dirs = append(dirs, hdr.Name)
			continue
		}
		content, err := io.ReadAll(tr)
		require.NoError(t, err)
		files[hdr.Name] = string(content)
	}
}

func TestWrite(t *testing.T) {
	const artifact = `{"abi":[],"bytecode":{"object":"0x00"},"ast":{"absolutePath":"src/A.sol"},"devdoc":{"details":"a < b"},"id":3}`
	const buildInfo = `{"id":"abc","ast":{"kept":true}}`
	base := t.TempDir()
	writeFiles(t, base, map[string]string{
		CompileCacheFile:                      testCompileCache,
		"forge-artifacts/A.sol/A.json":        artifact,
		"forge-artifacts/B.sol/B.json":        `{"abi":[]}`,
		"artifacts/build-info/abc.json":       buildInfo,
		"src/A.sol":                           "contract A {}",
		"src/A.t.sol":                         "contract ATest {}",
		"test/setup/Helper.sol":               "import {Test} from \"forge-std/Test.sol\";",
		"interfaces/IUnused.sol":              "interface IUnused {}",
		"deploy-config/devnet.json":           "{}",
		"book/index.md":                       "book",
		"lib/dep/LICENSE":                     "license",
		"lib/dep/foundry.toml":                "[profile.default]",
		"lib/dep/remappings.txt":              "dep/=src/",
		"lib/dep/src/Used.sol":                "contract Used {}",
		"lib/dep/src/Unused.sol":              "contract Unused {}",
		"lib/dep/docs/README.md":              "docs",
		"lib/dep/lib/sub/LICENSE-MIT":         "nested license",
		"lib/dep/lib/sub/contracts/Other.sol": "contract Other {}",
	})

	var buf bytes.Buffer
	require.NoError(t, Write(&buf, base, []string{"*.t.sol"}, ""))
	files, dirs := readArchive(t, buf.Bytes())

	var stripped map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(files["forge-artifacts/A.sol/A.json"]), &stripped))
	require.NotContains(t, stripped, "ast")
	var want map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(artifact), &want))
	delete(want, "ast")
	require.Equal(t, want, stripped)
	require.Contains(t, files["forge-artifacts/A.sol/A.json"], "a < b", "the artifact keeps its text unescaped")
	require.Equal(t, `{"abi":[]}`, files["forge-artifacts/B.sol/B.json"], "an artifact without an AST is unchanged")

	// forge discards the cached artifacts without the compile cache or the build infos.
	require.Equal(t, testCompileCache, files[CompileCacheFile])
	require.Equal(t, buildInfo, files["artifacts/build-info/abc.json"])

	require.Contains(t, files, "src/A.sol")
	require.NotContains(t, files, "src/A.t.sol")
	require.NotContains(t, files, "book/index.md")
	require.NotContains(t, files, "test/setup/Helper.sol", "a Solidity file outside the build is left out")
	require.NotContains(t, files, "interfaces/IUnused.sol")
	require.Contains(t, files, "deploy-config/devnet.json")

	require.Contains(t, files, "lib/dep/src/Used.sol")
	require.Contains(t, files, "lib/dep/LICENSE")
	require.Contains(t, files, "lib/dep/foundry.toml")
	require.Contains(t, files, "lib/dep/remappings.txt")
	require.Contains(t, files, "lib/dep/lib/sub/LICENSE-MIT")
	require.NotContains(t, files, "lib/dep/src/Unused.sol")
	require.NotContains(t, files, "lib/dep/docs/README.md")
	require.NotContains(t, files, "lib/dep/lib/sub/contracts/Other.sol")
	for _, dir := range dirs {
		require.NotRegexp(t, `^lib/.`, dir, "untar creates the library directories")
	}
}

func TestWrite_Errors(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		exclude []string
		wantErr string
	}{
		{
			name:    "no compile cache",
			files:   map[string]string{"src/A.sol": "contract A {}"},
			wantErr: "build the contracts first",
		},
		{
			name:    "artifacts outside the project",
			files:   map[string]string{CompileCacheFile: `{"paths": {"artifacts": "/abs/forge-artifacts", "libraries": ["lib"]}}`},
			wantErr: "not a directory in the project",
		},
		{
			name:    "no artifacts path",
			files:   map[string]string{CompileCacheFile: `{"paths": {"libraries": ["lib"]}}`},
			wantErr: "not a directory in the project",
		},
		{
			name:    "bad exclude pattern",
			files:   map[string]string{CompileCacheFile: testCompileCache},
			exclude: []string{"["},
			wantErr: "invalid exclude pattern",
		},
		{
			name:    "bad artifact",
			files:   map[string]string{CompileCacheFile: testCompileCache, "forge-artifacts/A.sol/A.json": "["},
			wantErr: "strip the AST of forge-artifacts/A.sol/A.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			writeFiles(t, base, tt.files)
			require.ErrorContains(t, Write(io.Discard, base, tt.exclude, ""), tt.wantErr)
		})
	}
}

// TestWrite_CleansCachePaths checks that cache paths with a "./" or a trailing slash match.
func TestWrite_CleansCachePaths(t *testing.T) {
	base := t.TempDir()
	writeFiles(t, base, map[string]string{
		CompileCacheFile:         `{"paths": {"artifacts": "./forge-artifacts/", "libraries": ["lib/"]}, "files": {}}`,
		"forge-artifacts/A.json": `{"abi":[],"ast":{}}`,
		"lib/dep/README.md":      "docs",
	})
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, base, nil, ""))
	files, _ := readArchive(t, buf.Bytes())
	require.NotContains(t, files["forge-artifacts/A.json"], "ast")
	require.NotContains(t, files, "lib/dep/README.md")
}

func TestWrite_Links(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret")
	require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o600))
	tests := []struct {
		name   string
		link   string
		target string
		// wantErr is the error of the pack, or "" when the bundle leaves the link out.
		wantErr string
	}{
		{name: "artifact to a file outside", link: "forge-artifacts/leak", target: outside, wantErr: "is a link"},
		{name: "artifact to the environment", link: "forge-artifacts/env", target: "/proc/self/environ", wantErr: "is a link"},
		{name: "artifacts directory", link: "forge-artifacts", target: t.TempDir(), wantErr: "is a link"},
		{name: "source outside the project", link: "src/Linked.sol", target: outside, wantErr: "is a link"},
		{name: "source in the project", link: "src/Linked.sol", target: "../other/Linked.sol", wantErr: "is a link"},
		{name: "artifact directory", link: "forge-artifacts/B.sol", target: t.TempDir(), wantErr: "is a link"},
		{name: "directory of sources", link: "lib/dep", target: "../other", wantErr: "is a link"},
		// On a filesystem that ignores case, forge-artifacts names these directories too.
		{name: "source in another spelling of the artifacts", link: "Forge-Artifacts/leak", target: "../notes-file", wantErr: "is a link"},
		{name: "other file in another spelling of the artifacts", link: "FORGE-ARTIFACTS/other", target: "../notes-file", wantErr: "is a link"},
		{name: "library remappings", link: "lib/dep/remappings.txt", target: "../../notes-file", wantErr: "is a link"},
		{name: "build infos", link: "artifacts/build-info", target: t.TempDir(), wantErr: "is a link"},
		{name: "other file", link: "AGENTS.md", target: "CLAUDE.md"},
		{name: "other file outside", link: "notes.txt", target: outside},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			writeFiles(t, base, map[string]string{
				CompileCacheFile:   `{"paths": {"artifacts": "forge-artifacts", "build_infos": "artifacts/build-info", "libraries": ["lib"]}, "files": {"src/Linked.sol": {}, "other/Linked.sol": {}, "lib/dep/D.sol": {}, "Forge-Artifacts/leak": {}, "notes-file": {}}}`,
				"CLAUDE.md":        "instructions",
				"other/Linked.sol": "contract Linked {}",
				"notes-file":       "notes",
				".env":             "KEY=secret",
			})
			if tt.link != "forge-artifacts" {
				writeFiles(t, base, map[string]string{"forge-artifacts/A.sol/A.json": `{"abi":[]}`})
			}
			require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(base, tt.link)), 0o755))
			require.NoError(t, os.Symlink(tt.target, filepath.Join(base, tt.link)))

			var buf bytes.Buffer
			err := Write(&buf, base, nil, "")
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			files, _ := readArchive(t, buf.Bytes())
			require.NotContains(t, files, tt.link)
			for name, content := range files {
				require.NotContains(t, content, "secret", name)
			}
		})
	}
}

func TestWrite_LeavesOutEnvFiles(t *testing.T) {
	base := t.TempDir()
	writeFiles(t, base, map[string]string{
		CompileCacheFile:            testCompileCache,
		".env":                      "KEY=secret",
		".envrc":                    "export KEY=secret",
		"deploy-config/.env.local":  "KEY=secret",
		".envs/mainnet":             "KEY=secret",
		"deploy-config/devnet.json": "{}",
	})
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, base, nil, ""))
	files, _ := readArchive(t, buf.Bytes())
	require.Contains(t, files, "deploy-config/devnet.json")
	for _, name := range []string{".env", ".envrc", "deploy-config/.env.local", ".envs/mainnet"} {
		require.NotContains(t, files, name)
	}
}

// TestWrite_RefusesLinkWhereScriptsRead checks that a link in the directory that op-deployer lets
// scripts read fails the pack, as one in the artifacts does, also when the compile cache names other
// artifacts and lists neither link as a source.
func TestWrite_RefusesLinkWhereScriptsRead(t *testing.T) {
	for _, link := range []string{ScriptReadDir + "/x", "out/A.sol/leak"} {
		t.Run(link, func(t *testing.T) {
			base := t.TempDir()
			writeFiles(t, base, map[string]string{
				CompileCacheFile:   `{"paths": {"artifacts": "out", "libraries": ["lib"]}, "files": {}}`,
				"out/A.sol/A.json": `{"abi":[]}`,
				".env":             "KEY=secret",
			})
			require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(base, link)), 0o755))
			require.NoError(t, os.Symlink(filepath.Join(base, ".env"), filepath.Join(base, link)))
			require.ErrorContains(t, Write(io.Discard, base, nil, ""), "is a link")
		})
	}
}

// TestWrite_ThroughLinkedProject checks that a project at a link packs.
func TestWrite_ThroughLinkedProject(t *testing.T) {
	base := t.TempDir()
	writeFiles(t, base, map[string]string{
		CompileCacheFile:               `{"paths": {"artifacts": "forge-artifacts", "libraries": ["lib"]}, "files": {"src/A.sol": {}}}`,
		"src/A.sol":                    "contract A {}",
		"forge-artifacts/A.sol/A.json": `{"abi":[]}`,
	})
	link := filepath.Join(t.TempDir(), "project")
	require.NoError(t, os.Symlink(base, link))

	var buf bytes.Buffer
	require.NoError(t, Write(&buf, link, nil, ""))
	files, _ := readArchive(t, buf.Bytes())
	require.Equal(t, "contract A {}", files["src/A.sol"])
}

// TestScriptReadDir_IsWhatScriptsMayRead ties ScriptReadDir, which the link check guards, to the
// directory that op-deployer lets scripts read.
func TestScriptReadDir_IsWhatScriptsMayRead(t *testing.T) {
	require.Equal(t, "./"+ScriptReadDir+"/", forge.ArtifactsReadPermission.Path)
}
