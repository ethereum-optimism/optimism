// Package release provides helpers to download and verify binaries released on GitHub.
//
// The downloader knows how to construct a release download URL, fetch the tarball,
// validate its checksum and extract the contained name to a destination path.
package release

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

// BinaryProvider is a small abstraction around downloading a specific name
// for a given version. Implementations should return the path of the downloaded
// name on success.
type BinaryProvider interface {
	Get(ctx context.Context, version string) (string, error)
}

// BinaryVersionCheckerFactory is a factory function that creates a BinaryVersionChecker for a
// specific requested version and OS/architecture pair. This allows for customizable version
// checking logic that may vary based on the target platform or version.
type BinaryVersionCheckerFactory func(requestedVersion string, os string, arch string) (BinaryVersionChecker, error)

// BinaryVersionChecker is a function that verifies the version of a downloaded binary matches
// the requested version. It takes a context and the path to the binary as input and returns
// an error if the version check fails (e.g., version mismatch or command execution error).
type BinaryVersionChecker func(ctx context.Context, binary string) error

// BinaryVersionComparator is a function that compares a requested version with the actual
// version extracted from a binary. It returns an error if the versions do not match according
// to the comparison logic (e.g., semver equality check).
type BinaryVersionComparator func(requestedVersion string, actualVersion string) error

// GithubReleaseChecksummer reads the downloaded archive data and validates its
// checksum. It should return an error if the checksum does not match.
type GithubReleaseChecksummer func(r io.Reader) error

// GithubReleaseChecksummerFactory returns a GithubReleaseChecksummer for a
// specific OS/architecture pair. This allows providing precomputed checksums
// for different release artifacts.
type GithubReleaseChecksummerFactory func(os string, arch string) (GithubReleaseChecksummer, error)

// GithubReleaseOSGetter returns the OS and architecture string used to select
// the appropriate release artifact (e.g. "linux", "amd64"). The function may
// inspect the runtime or be overridden for testing.
type GithubReleaseOSGetter func() (string, string, error)

// GithubReleaseURLGetter constructs the download URL for a given owner/repo
// and version for the provided os/arch pair.
type GithubReleaseURLGetter func(owner string, repo string, name string, version string, os string, arch string) (string, error)

type GithubReleaseCachePather func() (string, error)

type GithubReleaseBinaryLocator func(releaseDir string, name string, version string, os string, arch string) (string, error)

type GithubReleaseDownloader struct {
	// Repository owner (e.g. "ethereum-optimism").
	owner string

	// Repository name (e.g. "optimism").
	repo string

	// Name of the binary to download (e.g. "op-deployer").
	//
	// If the binary is not located under this name directly in the tarball,
	// a custom BinaryLocator should be provided.
	name string

	// Maximum allowed download size in bytes. When zero, no size limit is
	// enforced. This helps protect against unexpectedly large artifacts.
	maxDownloadSize int64

	// checksummerFactory produces a checksum-verifier for the artifact to be
	// downloaded. It must be set before attempting a download if checksum
	// validation is required.
	checksummerFactory GithubReleaseChecksummerFactory

	// urlGetter constructs the download URL for the release asset. The
	// default implementation assumes GitHub release download URL formats.
	urlGetter GithubReleaseURLGetter

	// osGetter returns the OS/arch pair used to select the appropriate
	// release artifact. It can be overridden for testing.
	osGetter GithubReleaseOSGetter

	// progressor reports download progress. Optional; may be nil.
	progressor ioutil.Progressor

	// cachePather returns the directory used for local caching of downloaded
	// artifacts. Typically points to a user-writable cache directory.
	cachePather GithubReleaseCachePather

	// locator determines where the desired file is located inside the
	// extracted tarball (useful for artifacts with nested directories).
	locator GithubReleaseBinaryLocator

	// logger is optional and used for informational logging during the
	// download/extract process. The `WithLogger` option sets this field.
	logger *log.Logger

	// versionCheckerFactory is optional and used to verify that the downloaded
	// binary matches the requested version. If set, it will be invoked after
	// successful download and extraction. The `WithVersionCheckerFactory` option
	// sets this field.
	versionCheckerFactory BinaryVersionCheckerFactory
}

var _ BinaryProvider = (*GithubReleaseDownloader)(nil)

type GithubReleaseDownloaderOption func(*GithubReleaseDownloader)

func WithVersionCheckerFactory(f BinaryVersionCheckerFactory) GithubReleaseDownloaderOption {
	return func(d *GithubReleaseDownloader) {
		d.versionCheckerFactory = f
	}
}

func NewStaticCommandVersionCheckerFactory(args []string, outputParser func(stdout string) (string, error), comparator BinaryVersionComparator) BinaryVersionCheckerFactory {
	return func(requestedVersion string, os string, arch string) (BinaryVersionChecker, error) {
		return func(ctx context.Context, binary string) error {
			cmd := exec.CommandContext(ctx, binary, args...)
			out, err := cmd.Output()
			if err != nil {
				return fmt.Errorf("version check failed: command '%s %s' returned an error: %w", binary, strings.Join(args, " "), err)
			}

			actualVersion, err := outputParser(string(out))
			if err != nil {
				return fmt.Errorf("version check failed: could not parse version from output '%s': %w", string(out), err)
			}

			return comparator(requestedVersion, actualVersion)
		}, nil
	}
}

func NewSemverEqualityComparator() BinaryVersionComparator {
	return func(requestedVersion string, actualVersion string) error {
		requestedVersionSemver, err := semver.NewVersion(requestedVersion)
		if err != nil {
			return fmt.Errorf("failed to convert version %s to semver: %w", requestedVersion, err)
		}

		actualVersionSemver, err := semver.NewVersion(actualVersion)
		if err != nil {
			return fmt.Errorf("failed to convert version %s to semver: %w", actualVersion, err)
		}

		if requestedVersionSemver.Compare(actualVersionSemver) != 0 {
			return fmt.Errorf("requested version %s does not match the actual one %s", requestedVersion, actualVersion)
		}

		return nil
	}
}

func WithChecksummerFactory(c GithubReleaseChecksummerFactory) GithubReleaseDownloaderOption {
	return func(d *GithubReleaseDownloader) {
		d.checksummerFactory = c
	}
}

func NewStaticChecksummerFactory(checksums map[string]string) GithubReleaseChecksummerFactory {
	return func(os string, arch string) (GithubReleaseChecksummer, error) {
		checksum, ok := checksums[fmt.Sprintf("%s_%s", os, arch)]
		if !ok {
			return nil, fmt.Errorf("no checksum found for os %s arch %s", os, arch)
		}

		return NewDefaultChecksummer(checksum), nil
	}
}

func NewDefaultChecksummer(expectedChecksum string) GithubReleaseChecksummer {
	return func(r io.Reader) error {
		h := sha256.New()
		if _, err := io.Copy(h, r); err != nil {
			return fmt.Errorf("could not calculate checksum: %w", err)
		}
		gotChecksum := fmt.Sprintf("%x", h.Sum(nil))
		if gotChecksum != expectedChecksum {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, gotChecksum)
		}
		return nil
	}
}

func WithCachePather(c GithubReleaseCachePather) GithubReleaseDownloaderOption {
	return func(d *GithubReleaseDownloader) {
		d.cachePather = c
	}
}

func NewHomeDirCachePather(namespace string) GithubReleaseCachePather {
	return func() (string, error) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not find home directory: %w", err)
		}
		return path.Join(homeDir, namespace, "cache"), nil
	}
}

func NewStaticCachePather(cacheDir string) GithubReleaseCachePather {
	return func() (string, error) {
		return cacheDir, nil
	}
}

func WithOSGetter(c GithubReleaseOSGetter) GithubReleaseDownloaderOption {
	return func(d *GithubReleaseDownloader) {
		d.osGetter = c
	}
}

func NewDefaultOSGetter() GithubReleaseOSGetter {
	return func() (string, string, error) {
		return runtime.GOOS, runtime.GOARCH, nil
	}
}

func WithURLGetter(c GithubReleaseURLGetter) GithubReleaseDownloaderOption {
	return func(d *GithubReleaseDownloader) {
		d.urlGetter = c
	}
}

func NewDefaultURLGetter() GithubReleaseURLGetter {
	return func(owner string, repo string, releaseName string, version string, os string, arch string) (string, error) {
		return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s_%s_%s_%s.tar.gz", owner, repo, version, releaseName, version, os, arch), nil
	}
}

func WithProgressor(p ioutil.Progressor) GithubReleaseDownloaderOption {
	return func(d *GithubReleaseDownloader) {
		d.progressor = p
	}
}

func WithMaxDownloadSize(size int64) GithubReleaseDownloaderOption {
	return func(d *GithubReleaseDownloader) {
		d.maxDownloadSize = size
	}
}

func WithLogger(l *log.Logger) GithubReleaseDownloaderOption {
	return func(d *GithubReleaseDownloader) {
		d.logger = l
	}
}

func WithBinaryLocator(l GithubReleaseBinaryLocator) GithubReleaseDownloaderOption {
	return func(d *GithubReleaseDownloader) {
		d.locator = l
	}
}

func NewDefaultBinaryLocator() GithubReleaseBinaryLocator {
	return func(releaseDir string, name string, version string, os string, arch string) (string, error) {
		return path.Join(releaseDir, name), nil
	}
}

func NewGithubReleaseDownloader(owner string, repo string, name string, opts ...GithubReleaseDownloaderOption) *GithubReleaseDownloader {
	d := &GithubReleaseDownloader{
		owner:           owner,
		repo:            repo,
		name:            name,
		maxDownloadSize: 0,

		urlGetter:   NewDefaultURLGetter(),
		osGetter:    NewDefaultOSGetter(),
		cachePather: NewHomeDirCachePather("op-name-downloader"),
		locator:     NewDefaultBinaryLocator(),
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

func (d *GithubReleaseDownloader) Get(ctx context.Context, version string) (string, error) {
	releaseOS, releaseArch, err := d.osGetter()
	if err != nil {
		return "", fmt.Errorf("failed to get OS and Arch: %w", err)
	}

	destinationPath, err := d.getDestinationPath(version, releaseOS, releaseArch)
	if err != nil {
		return "", fmt.Errorf("failed to get destination path: %w", err)
	}

	var binary string

	binary, err = d.getCached(ctx, version, destinationPath)
	if err != nil {
		binary, err = d.download(ctx, version, releaseOS, releaseArch, destinationPath)
	}

	if err != nil {
		return "", fmt.Errorf("failed to get binary %s of version %s: %w", d.name, version, err)
	}

	if d.versionCheckerFactory != nil {
		versionChecker, err := d.versionCheckerFactory(version, releaseOS, releaseArch)
		if err != nil {
			return "", fmt.Errorf("failed to create version checker for binary %s of version %s: %w", d.name, version, err)
		}

		err = versionChecker(ctx, binary)
		if err != nil {
			return "", fmt.Errorf("version check failed for binary %s of version %s: %w", d.name, version, err)
		}
	}

	return binary, nil
}

func (d *GithubReleaseDownloader) getDestinationPath(version string, os string, arch string) (string, error) {
	cacheDir, err := d.cachePather()
	if err != nil {
		return "", fmt.Errorf("could not get cache path: %w", err)
	}

	return path.Join(cacheDir, d.owner, d.repo, fmt.Sprintf("%s-%s-%s", version, os, arch), d.name), nil
}

func (d *GithubReleaseDownloader) getCached(ctx context.Context, version string, destinationPath string) (string, error) {
	nameInfo, err := os.Stat(destinationPath)
	if err != nil {
		return "", fmt.Errorf("could not find cached name %s: %w", destinationPath, err)
	}

	if !nameInfo.Mode().IsRegular() {
		return "", fmt.Errorf("cached name %s is not a regular file", destinationPath)
	}

	// The nine least-significant bits of the FileMode represent the standard Unix rwxrwxrwx permissions.
	// We are checking for the executable bit for the owner, group, or others.
	// 0111 is the octal representation for rwx for all.
	// A non-zero result from the bitwise AND operation means at least one executable bit is set.
	if nameInfo.Mode()&0111 == 0 {
		return "", fmt.Errorf("cached name %s is not executable", destinationPath)
	}

	return destinationPath, nil
}

func (d *GithubReleaseDownloader) download(ctx context.Context, version string, releaseOS, releaseArch, destinationPath string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "github-release-downloader-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	// Ensure temporary extraction directory is removed when we're done.
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Construct the download URL for the requested release artifact.
	releaseURL, err := d.urlGetter(d.owner, d.repo, d.name, version, releaseOS, releaseArch)
	if err != nil {
		return "", fmt.Errorf("failed to get download URL: %w", err)
	}

	// Obtain a checksum-verifier for the target OS/arch. This allows the
	// downloader to validate the archive before extraction.
	var checksummer GithubReleaseChecksummer
	if d.checksummerFactory != nil {
		checksummer, err = d.checksummerFactory(releaseOS, releaseArch)
		if err != nil {
			return "", fmt.Errorf("failed to get checksummer: %w", err)
		}
	}

	// Create a simple HTTP downloader. Note: if authentication is required
	// (e.g. private GitHub assets or API rate limiting), callers can extend
	// this to provide an authenticated http.Client to httputil.Downloader.
	downloader := &httputil.Downloader{
		Progressor: d.progressor,
		MaxSize:    d.maxDownloadSize,
	}

	// Download the release tarball into memory, validate checksum, then
	// extract. The artifact is expected to be reasonably small (controlled
	// via MaxSize) so in-memory buffering is acceptable here.
	buf := new(bytes.Buffer)
	if err := downloader.Download(ctx, releaseURL, buf); err != nil {
		return "", fmt.Errorf("failed to download name: %w", err)
	}

	// Verify the checksum of the downloaded data if a checksummer is provided.
	data := buf.Bytes()
	if checksummer != nil {
		if err := checksummer(bytes.NewReader(data)); err != nil {
			return "", fmt.Errorf("checksum mismatch: %w", err)
		}
	}

	// The released artifact is expected to be a gzipped tarball; create a
	// gzip reader and unpack the tar contents into the temp dir.
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}

	tr := tar.NewReader(gzr)
	if err := ioutil.Untar(tmpDir, tr); err != nil {
		return "", fmt.Errorf("failed to untar: %w", err)
	}

	sourcePath, err := d.locator(tmpDir, d.name, version, releaseOS, releaseArch)
	if err != nil {
		return "", fmt.Errorf("failed to locate binary %s in extracted archive: %w", d.name, err)
	}

	err = os.MkdirAll(path.Dir(destinationPath), 0o755)
	if err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Move the extracted name to the destination path and ensure it is
	// executable by clearing/setting appropriate file mode bits.
	if err := ioutil.SafeRename(sourcePath, destinationPath); err != nil {
		return "", fmt.Errorf("failed to move name from %s to %s: %w", sourcePath, destinationPath, err)
	}
	if err := os.Chmod(destinationPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to set executable bit: %w", err)
	}

	return destinationPath, nil
}

func getOPStackArtifactSlug(name string, version string, os string, arch string) string {
	return fmt.Sprintf("%s-%s-%s-%s", name, version, os, arch)
}

func NewOPStackURLGetter() GithubReleaseURLGetter {
	return func(owner string, repo string, name string, version string, os string, arch string) (string, error) {
		return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s%%2Fv%s/%s.tar.gz", owner, repo, name, version, getOPStackArtifactSlug(name, version, os, arch)), nil
	}
}

func NewOPStackBinaryLocator() GithubReleaseBinaryLocator {
	return func(releaseDir string, name string, version string, os string, arch string) (string, error) {
		// The binary is always located at <name>-<version>-<os>-<arch>/<name>
		return path.Join(releaseDir, getOPStackArtifactSlug(name, version, os, arch), name), nil
	}
}
