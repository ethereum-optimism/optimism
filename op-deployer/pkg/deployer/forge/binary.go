package forge

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

//go:embed version.json
var versionJSON []byte

type VersionConfig struct {
	Forge     string            `json:"forge"`
	Checksums map[string]string `json:"checksums"`
}

// StandardVersion is the Foundry version that op-deployer will download if it's not found on PATH.
var StandardVersion string

// checksums map the OS/architecture to the expected checksum of the binary.
var checksums = map[string]string{}

func init() {
	var versionConfig VersionConfig
	if err := json.Unmarshal(versionJSON, &versionConfig); err != nil {
		panic(err)
	}
	StandardVersion = versionConfig.Forge
	checksums = versionConfig.Checksums
}

// maxDownloadSize bounds downloads while accommodating the supported Foundry release archives.
const maxDownloadSize = 150 * 1024 * 1024

// getOS returns the OS of the forge release archive: alpine on musl libc, or with FORGE_ENV=alpine.
func getOS() string {
	if os.Getenv("FORGE_ENV") == "alpine" || (runtime.GOOS == "linux" && isMusl()) {
		return "alpine"
	}
	return runtime.GOOS
}

// isMusl reports whether the system has the musl dynamic loader.
func isMusl() bool {
	loaders, err := filepath.Glob("/lib/ld-musl-*.so.1")
	return err == nil && len(loaders) > 0
}

func binaryURL(sysOS, sysArch string) string {
	return fmt.Sprintf("https://github.com/foundry-rs/foundry/releases/download/%s/foundry_%s_%s_%s.tar.gz", StandardVersion, StandardVersion, sysOS, sysArch)
}

type Binary interface {
	Ensure(ctx context.Context) error
	Path() string
}

type Bin struct {
	path string
}

func StaticBinary(path string) Binary {
	return &Bin{path: path}
}

func (b *Bin) Ensure(ctx context.Context) error {
	return nil
}

func (b *Bin) Path() string {
	return b.path
}

type PathBin struct {
	path string
}

func PathBinary() Binary {
	return new(PathBin)
}

func (b *PathBin) Ensure(ctx context.Context) error {
	var err error
	b.path, err = exec.LookPath("forge")
	if err != nil {
		return fmt.Errorf("could not find binary: %w", err)
	}
	return nil
}

func (b *PathBin) Path() string {
	return b.path
}

// foundryTools are the tools that op-deployer runs from the foundry archive.
var foundryTools = []string{"forge", "anvil"}

// StandardBin forces the use of the standard version of a foundry tool, forge by default, by
// first checking for the version locally, then downloading from github
// if needed
type StandardBin struct {
	tool       string
	progressor ioutil.Progressor

	cachePather func() (string, error)
	checksummer func(r io.Reader) error
	url         string
	path        string
}

type StandardBinOpt func(s *StandardBin)

// WithTool selects the foundry tool, one of foundryTools.
func WithTool(tool string) StandardBinOpt {
	return func(s *StandardBin) {
		s.tool = tool
	}
}

func WithProgressor(p ioutil.Progressor) StandardBinOpt {
	return func(s *StandardBin) {
		s.progressor = p
	}
}

func WithURL(url string) StandardBinOpt {
	return func(s *StandardBin) {
		s.url = url
	}
}

func WithCachePather(pather func() (string, error)) StandardBinOpt {
	return func(s *StandardBin) {
		s.cachePather = pather
	}
}

func WithChecksummer(checksummer func(r io.Reader) error) StandardBinOpt {
	return func(s *StandardBin) {
		s.checksummer = checksummer
	}
}

func homedirCachePather() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %w", err)
	}
	return path.Join(homeDir, ".op-deployer", "cache"), nil
}

func staticChecksummer(expChecksum string) func(r io.Reader) error {
	return func(r io.Reader) error {
		h := sha256.New()
		if _, err := io.Copy(h, r); err != nil {
			return fmt.Errorf("could not calculate checksum: %w", err)
		}
		gotChecksum := fmt.Sprintf("%x", h.Sum(nil))
		if gotChecksum != expChecksum {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", expChecksum, gotChecksum)
		}
		return nil
	}
}

func githubChecksummer(r io.Reader) error {
	expChecksum := checksums[getOS()+"_"+runtime.GOARCH]
	if expChecksum == "" {
		return fmt.Errorf("could not find checksum for %s_%s", getOS(), runtime.GOARCH)
	}
	return staticChecksummer(expChecksum)(r)
}

func NewStandardBinary(opts ...StandardBinOpt) (*StandardBin, error) {
	bin := &StandardBin{
		tool:        "forge",
		url:         binaryURL(getOS(), runtime.GOARCH),
		cachePather: homedirCachePather,
		checksummer: githubChecksummer,
	}
	for _, opt := range opts {
		opt(bin)
	}
	return bin, nil
}

func (b *StandardBin) Ensure(ctx context.Context) error {
	// 1) Exit early if b.path already set (via previous Ensure call)
	if b.path != "" {
		return nil
	}

	// 2) PATH: use if version matches the pinned Version
	if toolPath, err := exec.LookPath(b.tool); err == nil {
		if ver, err := getForgeVersion(ctx, toolPath); err == nil && ver == StandardVersion {
			b.path = toolPath
			return nil
		}
	}

	// 3) Cache: use if version matches; otherwise replace it
	binDir, err := b.cachePather()
	if err != nil {
		return fmt.Errorf("could not provide cache dir: %w", err)
	}
	binPath := path.Join(binDir, b.tool)
	if st, err := os.Stat(binPath); err == nil && !st.IsDir() {
		// tool binary exists in cache; check version
		if ver, err := getForgeVersion(ctx, binPath); err == nil && ver == StandardVersion {
			b.path = binPath
			return nil
		}
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not stat %s: %w", binPath, err)
	}

	// 4) Download expected version for this OS/arch and verify checksum
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		return fmt.Errorf("could not create forge cache directory %s: %w", binDir, err)
	}
	if err := b.downloadBinary(ctx, binDir); err != nil {
		return fmt.Errorf("could not download %s %s from %s; without network access, put %s %s on PATH: %w", b.tool, StandardVersion, b.url, b.tool, StandardVersion, err)
	}
	b.path = binPath
	return nil
}

func (b *StandardBin) Path() string {
	return b.path
}

func (b *StandardBin) downloadBinary(ctx context.Context, dest string) error {
	tmpDir, err := os.MkdirTemp("", "op-deployer-forge-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	downloader := &httputil.Downloader{
		Progressor: b.progressor,
		MaxSize:    maxDownloadSize,
	}
	buf := new(bytes.Buffer)
	if err := downloader.Download(ctx, b.url, buf); err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	data := buf.Bytes()
	if err := b.checksummer(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("checksum mismatch: %w", err)
	}
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	tr := tar.NewReader(gzr)
	if err := ioutil.Untar(tmpDir, tr); err != nil {
		return fmt.Errorf("failed to untar: %w", err)
	}
	// Install every tool, so that no tool needs a download of its own.
	for _, tool := range foundryTools {
		if err := ioutil.SafeRename(path.Join(tmpDir, tool), path.Join(dest, tool)); err != nil {
			return fmt.Errorf("failed to move %s binary: %w", tool, err)
		}
		if err := os.Chmod(path.Join(dest, tool), 0o700); err != nil {
			return fmt.Errorf("failed to set executable bit of %s: %w", tool, err)
		}
	}
	return nil
}

func getForgeVersion(ctx context.Context, forgePath string) (string, error) {
	cmd := exec.CommandContext(ctx, forgePath, "--version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("exec %s --version failed: %w", forgePath, err)
	}
	// Example output: "forge Version: 1.3.1-v1.3.1" -> capture initial "1.3.1"
	re := regexp.MustCompile(`(\d+\.\d+\.\d+)`)
	m := re.FindStringSubmatch(string(out))
	if len(m) < 2 {
		return "", fmt.Errorf("could not parse version tag from: %q", out)
	}
	return "v" + m[1], nil
}
