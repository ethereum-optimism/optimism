package artifacts

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts/bundle"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

const (
	// bundleFormat names the bundles that bundle.Write writes, so that a change to it repacks.
	bundleFormat = "op-deployer-bundle-v1"
	// keptBundles is the number of the most recently used bundles that eviction keeps.
	keptBundles = 4
	// packingPrefix starts the name of a bundle that a process is still packing.
	packingPrefix = "packing-"
	// staleAfter is the age after which a pack that has not finished has stopped.
	staleAfter = 24 * time.Hour
	// packLockSuffix and useLockSuffix end the names of the locks of a bundle.
	packLockSuffix = ".pack.lock"
	useLockSuffix  = ".use.lock"
)

// openLocal returns the artifacts at dir: a forge project, or the artifacts directory of one. It
// packs a built forge project into a bundle, as bundle.Write packs it, so that forge runs on the
// bundle: without the AST, a run takes a fraction of the memory and time. The bundles are in
// op-deployer's cache directory, shared by the processes of the user, and keyed by the project and
// its build: a rebuild of the project gets a new bundle, and a bundle in use stays as it was. A
// directory that is not a built forge project, or that op-deployer cannot bundle, is used as it is.
func openLocal(dir string, targetDir string) fs.FS {
	return openProject(dir, targetDir, true)
}

// openProject returns the artifacts at dir as openLocal does. With shared false, the bundles are in
// a new directory in targetDir, not in the shared one.
func openProject(dir string, targetDir string, shared bool) fs.FS {
	dir = filepath.Clean(dir)
	project, artifactsDir := dir, filepath.Join(dir, "forge-artifacts")
	if _, err := os.Stat(artifactsDir); err != nil {
		// TODO(#18346): Accept this for now but in the future we should error
		project, artifactsDir = filepath.Dir(dir), dir
	}
	// A bundle is keyed by the project's path, and its pack walks the project.
	resolved, err := filepath.EvalSymlinks(project)
	if err != nil {
		return os.DirFS(artifactsDir)
	}
	rel, err := filepath.Rel(project, artifactsDir)
	if err != nil {
		return os.DirFS(artifactsDir)
	}
	if buildArtifactsDir, err := bundle.ArtifactsDir(resolved); err != nil || filepath.ToSlash(rel) != buildArtifactsDir {
		return os.DirFS(artifactsDir)
	}
	if _, err := os.Stat(filepath.Join(resolved, "foundry.toml")); err != nil {
		return os.DirFS(artifactsDir)
	}

	// The shared directory can fail where a private one works, as without file locks on a network home.
	if shared {
		if root := sharedBundlesDir(); root != "" {
			if bundleDir, err := localBundle(resolved, root, rel); err == nil {
				return os.DirFS(filepath.Join(bundleDir, rel))
			}
		}
	}
	root, err := privateBundlesDir(targetDir)
	if err != nil {
		return os.DirFS(artifactsDir)
	}
	bundleDir, err := localBundle(resolved, root, rel)
	if err != nil {
		return os.DirFS(artifactsDir)
	}
	return os.DirFS(filepath.Join(bundleDir, rel))
}

// sharedBundlesDir returns the directory of the bundles that the processes of the user share, in
// op-deployer's cache directory, or "" when the user has none.
func sharedBundlesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".op-deployer", "cache", "bundles")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return ""
	}
	return dir
}

// privateBundlesDir returns a new directory for the bundles of this process in targetDir, or in the
// temporary directory when targetDir is empty. Its name is not predictable, so that no other user
// can plant a bundle in it.
func privateBundlesDir(targetDir string) (string, error) {
	if targetDir != "" {
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return "", fmt.Errorf("failed to create the artifacts directory: %w", err)
		}
	}
	dir, err := os.MkdirTemp(targetDir, "op-deployer-bundles-*")
	if err != nil {
		return "", fmt.Errorf("failed to create the bundle directory: %w", err)
	}
	RegisterForCleanup(dir)
	return dir, nil
}

// pack writes the bundle of a forge project into a directory. Tests replace it to count the packs.
var pack = packBundle

// usedBundles holds the use locks of the bundles that this process uses, by directory.
var usedBundles = struct {
	sync.Mutex
	locks map[string]*fileLock
}{locks: make(map[string]*fileLock)}

// localBundle returns the bundle of the build of the forge project at project in root, and packs it
// when root has none. A bundle has two locks. Its pack lock is exclusive while a process packs it, so
// that a process that finds another one packing the same build waits and uses its bundle. Its use
// lock is shared while a process uses it, until the process exits, and evictBundles removes only a
// bundle whose use lock it can take exclusive. Every op-deployer that shares a bundle directory
// follows these locks, so a change to them needs another directory.
func localBundle(project string, root string, artifactsDir string) (string, error) {
	key, err := buildKey(project)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, key)
	usedBundles.Lock()
	defer usedBundles.Unlock()
	if _, ok := usedBundles.locks[dir]; ok {
		touchBundle(dir)
		return dir, nil
	}
	use, err := openFileLock(dir + useLockSuffix)
	if err != nil {
		return "", err
	}
	if err := use.lock(false); err != nil {
		use.release()
		return "", err
	}
	// An eviction can have removed the bundle before this process took the use lock.
	if _, err := os.Stat(dir); err != nil {
		if err := packOnce(project, root, key, artifactsDir); err != nil {
			use.release()
			return "", err
		}
	}
	touchBundle(dir)
	usedBundles.locks[dir] = use
	return dir, nil
}

// packOnce packs the build key of the forge project at project into root, unless another process
// packed it while this one waited for the pack lock, and then evicts the bundles beyond keptBundles.
func packOnce(project string, root string, key string, artifactsDir string) error {
	dir := filepath.Join(root, key)
	packLock, err := openFileLock(dir + packLockSuffix)
	if err != nil {
		return err
	}
	defer packLock.release()
	if err := packLock.lock(true); err != nil {
		return err
	}
	if _, err := os.Stat(dir); err == nil {
		return nil
	}
	// The pack goes to a new directory and is renamed into place, so that a pack that stops leaves no
	// partial bundle.
	packing, err := os.MkdirTemp(root, packingPrefix+"*")
	if err != nil {
		return fmt.Errorf("failed to create the bundle directory: %w", err)
	}
	if err := pack(project, packing); err != nil {
		_ = os.RemoveAll(packing)
		return err
	}
	// A build that changed during the pack can have mixed the two builds.
	if after, err := buildKey(project); err != nil || after != key {
		_ = os.RemoveAll(packing)
		return fmt.Errorf("the forge project %s changed while op-deployer copied it; run op-deployer again after the build", project)
	}
	if err := checkBundle(packing, artifactsDir); err != nil {
		_ = os.RemoveAll(packing)
		return fmt.Errorf("the bundle of the forge project %s is incomplete: %w", project, err)
	}
	if err := os.Rename(packing, dir); err != nil {
		_ = os.RemoveAll(packing)
		// Without file locks, another process can have stored the same bundle first.
		if _, statErr := os.Stat(dir); statErr != nil {
			return fmt.Errorf("failed to store the bundle: %w", err)
		}
	}
	evictBundles(root, dir)
	return nil
}

// checkBundle checks that the bundle at dir has the files that forge needs.
func checkBundle(dir string, artifactsDir string) error {
	for name, wantDir := range map[string]bool{bundle.CompileCacheFile: false, "foundry.toml": false, artifactsDir: true} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if info.IsDir() != wantDir {
			return fmt.Errorf("%s has the wrong type", name)
		}
	}
	return nil
}

// touchBundle marks the bundle at dir as used now, for evictBundles.
func touchBundle(dir string) {
	now := time.Now()
	_ = os.Chtimes(dir, now, now)
}

// buildKey returns the key of the bundle of the forge project at project. The path keeps projects
// apart. The compile cache identifies the build: it holds the hash, time and settings of each source
// and the ids of the build infos. The forge config adds the settings of a run.
func buildKey(project string) (string, error) {
	h := sha256.New()
	h.Write([]byte(bundleFormat))
	h.Write([]byte{0})
	h.Write([]byte(project))
	for _, name := range []string{bundle.CompileCacheFile, "foundry.toml"} {
		data, err := os.ReadFile(filepath.Join(project, name))
		if err != nil {
			return "", fmt.Errorf("failed to read the build of the forge project: %w", err)
		}
		h.Write([]byte{0})
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// packBundle writes the bundle of the forge project at project into dir.
func packBundle(project string, dir string) error {
	pr, pw := io.Pipe()
	written := make(chan error, 1)
	go func() {
		err := bundle.Write(pw, project, nil, "")
		pw.CloseWithError(err)
		written <- err
	}()
	// When bundle.Write fails, Untar fails with its error.
	err := ioutil.Untar(dir, tar.NewReader(pr))
	// Unblocks bundle.Write when Untar stopped early.
	pr.CloseWithError(errors.New("bundle extraction stopped"))
	if writeErr := <-written; err == nil {
		err = writeErr
	}
	if err != nil {
		return fmt.Errorf("failed to bundle the forge project %s: %w", project, err)
	}
	return nil
}

// evictBundles removes the bundles in root, other than keep, beyond the keptBundles most recently
// used, unless a process uses them, and the packs that stopped more than staleAfter ago. The empty
// lock files stay, so that processes that wait for a lock still exclude each other.
func evictBundles(root string, keep string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	type usedBundle struct {
		path string
		used time.Time
	}
	var bundles []usedBundle
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || !entry.IsDir() {
			continue
		}
		p := filepath.Join(root, entry.Name())
		if strings.HasPrefix(entry.Name(), packingPrefix) {
			if time.Since(info.ModTime()) > staleAfter {
				_ = os.RemoveAll(p)
			}
			continue
		}
		if p != keep {
			bundles = append(bundles, usedBundle{path: p, used: info.ModTime()})
		}
	}
	slices.SortFunc(bundles, func(a, b usedBundle) int { return b.used.Compare(a.used) })
	// keep is one of the keptBundles.
	for _, b := range bundles[min(len(bundles), keptBundles-1):] {
		evictBundle(b.path)
	}
}

// evictBundle removes the bundle at dir unless a process uses it. It moves the bundle out of the way
// first, so that a removal that stops leaves no partial bundle at dir.
func evictBundle(dir string) {
	use, err := openFileLock(dir + useLockSuffix)
	if err != nil {
		return
	}
	defer use.release()
	if ok, err := use.tryLock(); err != nil || !ok {
		return
	}
	trash, err := os.MkdirTemp(filepath.Dir(dir), packingPrefix+"evicted-*")
	if err != nil {
		return
	}
	_ = os.Rename(dir, filepath.Join(trash, "bundle"))
	_ = os.RemoveAll(trash)
}
