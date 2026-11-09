package artifacts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts/bundle"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

const (
	testCompileCache = `{
  "paths": {"artifacts": "forge-artifacts", "libraries": ["lib"]},
  "files": {"src/A.sol": {}}
}`
	testArtifact = `{"abi":[],"bytecode":{"object":"0x00"},"ast":{"absolutePath":"src/A.sol"}}`
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
}

// writeProject writes a built forge project and returns its directory.
func writeProject(t *testing.T) string {
	t.Helper()
	project := t.TempDir()
	writeFiles(t, project, map[string]string{
		"foundry.toml":                 "[profile.default]",
		bundle.CompileCacheFile:        testCompileCache,
		"forge-artifacts/A.sol/A.json": testArtifact,
		"src/A.sol":                    "contract A {}",
	})
	return project
}

func readArtifact(t *testing.T, fsys fs.FS) string {
	t.Helper()
	data, err := fs.ReadFile(fsys, "A.sol/A.json")
	require.NoError(t, err)
	return string(data)
}

// bundlesIn returns the bundles in op-deployer's cache directory under home.
func bundlesIn(t *testing.T, home string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(home, ".op-deployer", "cache", "bundles", "*"))
	require.NoError(t, err)
	var bundles []string
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			bundles = append(bundles, p)
		}
	}
	return bundles
}

func TestOpenLocal(t *testing.T) {
	for name, locate := range map[string]func(project string) string{
		"project":             func(project string) string { return project },
		"artifacts directory": func(project string) string { return filepath.Join(project, "forge-artifacts") },
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			project := writeProject(t)

			fsys := openLocal(locate(project), t.TempDir())
			require.NotContains(t, readArtifact(t, fsys), `"ast"`)
			bundles := bundlesIn(t, home)
			require.Len(t, bundles, 1)
			require.FileExists(t, filepath.Join(bundles[0], bundle.CompileCacheFile))
			require.FileExists(t, filepath.Join(bundles[0], "foundry.toml"))

			// A later caller uses the same bundle.
			again := openLocal(locate(project), t.TempDir())
			require.Equal(t, readArtifact(t, fsys), readArtifact(t, again))
			require.Equal(t, bundles, bundlesIn(t, home))

			// A rebuild gets a new bundle, and the old one stays as it was.
			writeFiles(t, project, map[string]string{
				"forge-artifacts/A.sol/A.json": `{"abi":[1]}`,
				bundle.CompileCacheFile:        testCompileCache + " ",
			})
			rebuilt := openLocal(locate(project), t.TempDir())
			require.Equal(t, `{"abi":[1]}`, readArtifact(t, rebuilt))
			require.NotContains(t, readArtifact(t, fsys), `[1]`)
			require.Len(t, bundlesIn(t, home), 2)
		})
	}
}

func TestLocalBundle_UsesExistingBundle(t *testing.T) {
	project := writeProject(t)
	root := t.TempDir()
	key, err := buildKey(project)
	require.NoError(t, err)
	writeFiles(t, filepath.Join(root, key), map[string]string{"packed-by": "another process"})

	dir, err := localBundle(project, root, "forge-artifacts")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, key), dir)
	require.NoFileExists(t, filepath.Join(dir, "foundry.toml"), "the bundle is not packed again")
}

// TestLocalBundle_PacksOnceAcrossProcesses runs localBundle in processes that start together, as
// test binaries of sysgo and op-e2e do, and checks that only one of them packs.
func TestLocalBundle_PacksOnceAcrossProcesses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no file locks")
	}
	project := writeProject(t)
	root := t.TempDir()
	packs := filepath.Join(t.TempDir(), "packs")
	const processes = 4
	cmds := make([]*exec.Cmd, processes)
	outs := make([]*bytes.Buffer, processes)
	for i := range processes {
		cmds[i] = exec.Command(os.Args[0], "-test.run=^TestLocalBundleHelperProcess$")
		cmds[i].Env = append(os.Environ(), "BUNDLE_HELPER_PROJECT="+project, "BUNDLE_HELPER_ROOT="+root, "BUNDLE_HELPER_PACKS="+packs)
		outs[i] = new(bytes.Buffer)
		cmds[i].Stdout = outs[i]
		cmds[i].Stderr = outs[i]
		require.NoError(t, cmds[i].Start())
	}
	for i, cmd := range cmds {
		require.NoError(t, cmd.Wait(), outs[i].String())
	}
	data, err := os.ReadFile(packs)
	require.NoError(t, err)
	require.Equal(t, "packed\n", string(data))
	key, err := buildKey(project)
	require.NoError(t, err)
	for _, out := range outs {
		require.Contains(t, out.String(), "bundle="+filepath.Join(root, key)+"\n")
	}
}

// TestLocalBundleHelperProcess is a process of TestLocalBundle_PacksOnceAcrossProcesses.
func TestLocalBundleHelperProcess(t *testing.T) {
	project := os.Getenv("BUNDLE_HELPER_PROJECT")
	if project == "" {
		t.Skip("run by TestLocalBundle_PacksOnceAcrossProcesses")
	}
	pack = func(project string, dir string) error {
		f, err := os.OpenFile(os.Getenv("BUNDLE_HELPER_PACKS"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		require.NoError(t, err)
		_, err = f.WriteString("packed\n")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		// Gives the other processes time to wait for the pack lock.
		time.Sleep(500 * time.Millisecond)
		return packBundle(project, dir)
	}
	dir, err := localBundle(project, os.Getenv("BUNDLE_HELPER_ROOT"), "forge-artifacts")
	require.NoError(t, err)
	fmt.Printf("bundle=%s\n", dir)
}

func TestLocalBundle_MarksUse(t *testing.T) {
	project := writeProject(t)
	root := t.TempDir()
	dir, err := localBundle(project, root, "forge-artifacts")
	require.NoError(t, err)
	old := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(dir, old, old))
	again, err := localBundle(project, root, "forge-artifacts")
	require.NoError(t, err)
	require.Equal(t, dir, again)
	info, err := os.Stat(dir)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), info.ModTime(), time.Minute)
}

func TestOpenLocal_WithoutHomeUsesPrivateDirectory(t *testing.T) {
	t.Setenv("HOME", "")
	project := writeProject(t)
	targetDir := t.TempDir()
	fsys := openLocal(project, targetDir)
	require.NotContains(t, readArtifact(t, fsys), `"ast"`)
	roots, err := filepath.Glob(filepath.Join(targetDir, "op-deployer-bundles-*"))
	require.NoError(t, err)
	require.Len(t, roots, 1)
	info, err := os.Stat(roots[0])
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o700), info.Mode().Perm())
}

func TestEvictBundles(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	dirs := map[string]time.Time{
		"keep":              now.Add(-time.Hour),
		"used-1":            now,
		"used-2":            now.Add(-time.Minute),
		"used-3":            now.Add(-2 * time.Minute),
		"used-4":            now.Add(-3 * time.Minute),
		"in-use":            now.Add(-4 * time.Minute),
		"used-5":            now.Add(-5 * time.Minute),
		packingPrefix + "1": now.Add(-2 * staleAfter),
		packingPrefix + "2": now,
	}
	for name, used := range dirs {
		p := filepath.Join(root, name)
		require.NoError(t, os.Mkdir(p, 0o700))
		require.NoError(t, os.Chtimes(p, used, used))
	}
	inUse, err := openFileLock(filepath.Join(root, "in-use") + useLockSuffix)
	require.NoError(t, err)
	require.NoError(t, inUse.lock(false))
	defer inUse.release()

	evictBundles(root, filepath.Join(root, "keep"))
	var left []string
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() {
			left = append(left, entry.Name())
		}
	}
	// keep and the three others most recently used stay, and so does the bundle in use and the pack
	// that runs.
	require.ElementsMatch(t, []string{"keep", "used-1", "used-2", "used-3", "in-use", packingPrefix + "2"}, left)
}

func TestOpenLocal_UsesOtherDirectoryAsItIs(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		dir   string
	}{
		{
			name:  "no compile cache",
			files: map[string]string{"forge-artifacts/A.sol/A.json": testArtifact},
			dir:   "forge-artifacts",
		},
		{
			name:  "not the artifacts directory of the build",
			files: map[string]string{bundle.CompileCacheFile: testCompileCache, "out/A.sol/A.json": testArtifact},
			dir:   "out",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			project := t.TempDir()
			writeFiles(t, project, tt.files)
			fsys := openLocal(filepath.Join(project, tt.dir), t.TempDir())
			require.Contains(t, readArtifact(t, fsys), `"ast"`)
			require.Empty(t, bundlesIn(t, home))
		})
	}
}

func TestDownload_FileLocatorPacksBuiltProject(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	project := writeProject(t)
	loc, err := NewFileLocator(project)
	require.NoError(t, err)
	fsys, err := Download(context.Background(), loc, ioutil.NoopProgressor(), t.TempDir())
	require.NoError(t, err)
	require.NotContains(t, readArtifact(t, fsys), `"ast"`)
}

func TestOpenLocal_ResolvesLinks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := writeProject(t)
	link := filepath.Join(t.TempDir(), "contracts")
	require.NoError(t, os.Symlink(project, link))

	for _, dir := range []string{link, filepath.Join(link, "forge-artifacts"), project + "/forge-artifacts/"} {
		fsys := openLocal(dir, t.TempDir())
		require.NotContains(t, readArtifact(t, fsys), `"ast"`, dir)
	}
	require.Len(t, bundlesIn(t, home), 1, "the link and the project share the bundle")
}

func TestLocalBundle_RefusesBuildThatChangedDuringPack(t *testing.T) {
	project := writeProject(t)
	root := t.TempDir()
	pack = func(project string, dir string) error {
		writeFiles(t, project, map[string]string{bundle.CompileCacheFile: testCompileCache + " "})
		return packBundle(project, dir)
	}
	t.Cleanup(func() { pack = packBundle })
	_, err := localBundle(project, root, "forge-artifacts")
	require.ErrorContains(t, err, "changed while op-deployer copied it")
	packs, err := filepath.Glob(filepath.Join(root, packingPrefix+"*"))
	require.NoError(t, err)
	require.Empty(t, packs)
}

func TestDownload_RepacksHTTPBundleInPrivateDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := writeProject(t)
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, filepath.WalkDir(project, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(project, path)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{Name: filepath.ToSlash(rel), Mode: 0o644, Size: int64(len(data))}); err != nil {
			return err
		}
		_, err = tw.Write(data)
		return err
	}))
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(buf.Bytes())
	}))
	defer ts.Close()

	loc, err := NewLocatorFromURL(ts.URL + "/artifacts.tar.gz")
	require.NoError(t, err)
	targetDir := t.TempDir()
	fsys, err := Download(context.Background(), loc, ioutil.NoopProgressor(), targetDir)
	require.NoError(t, err)
	require.NotContains(t, readArtifact(t, fsys), `"ast"`)
	require.Empty(t, bundlesIn(t, home), "a downloaded bundle is not shared")
	private, err := filepath.Glob(filepath.Join(targetDir, "op-deployer-bundles-*"))
	require.NoError(t, err)
	require.Len(t, private, 1)
}

func TestLocalBundle_KeepsBundleInUse(t *testing.T) {
	project := writeProject(t)
	root := t.TempDir()
	dir, err := localBundle(project, root, "forge-artifacts")
	require.NoError(t, err)
	evictBundle(dir)
	require.DirExists(t, dir)
}

func TestLocalBundle_RefusesIncompleteBundle(t *testing.T) {
	project := writeProject(t)
	root := t.TempDir()
	pack = func(string, string) error { return nil }
	t.Cleanup(func() { pack = packBundle })
	_, err := localBundle(project, root, "forge-artifacts")
	require.ErrorContains(t, err, "incomplete")
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	for _, entry := range entries {
		require.False(t, entry.IsDir(), "no bundle or pack is left: %s", entry.Name())
	}
}

func TestOpenLocal_LinksInTheProject(t *testing.T) {
	t.Run("linked artifacts directory", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		project := writeProject(t)
		moved := filepath.Join(t.TempDir(), "forge-artifacts")
		require.NoError(t, os.Rename(filepath.Join(project, "forge-artifacts"), moved))
		require.NoError(t, os.Symlink(moved, filepath.Join(project, "forge-artifacts")))
		fsys := openLocal(project, t.TempDir())
		require.Contains(t, readArtifact(t, fsys), `"ast"`, "a project that cannot be bundled is used as it is")
		require.Empty(t, bundlesIn(t, home))
	})
	t.Run("linked source", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		project := writeProject(t)
		source := filepath.Join(t.TempDir(), "A.sol")
		require.NoError(t, os.Rename(filepath.Join(project, "src", "A.sol"), source))
		require.NoError(t, os.Symlink(source, filepath.Join(project, "src", "A.sol")))
		fsys := openLocal(project, t.TempDir())
		require.NotContains(t, readArtifact(t, fsys), `"ast"`)
		// The bundle has a copy of the linked file.
		data, err := os.ReadFile(filepath.Join(fmt.Sprint(fsys), "..", "src", "A.sol"))
		require.NoError(t, err)
		require.Equal(t, "contract A {}", string(data))
	})
}
