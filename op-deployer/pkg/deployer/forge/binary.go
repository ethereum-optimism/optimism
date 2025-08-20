package forge

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

// Version is the Foundry version that op-deployer will download if it's not found on PATH.
const Version = "v1.3.1"

// maxDownloadSize is the maximum size of the Foundry tarball that will be downloaded. It's typically ~60MB so
// this should be more than enough.
const maxDownloadSize = 100 * 1024 * 1024

// checksums map the OS/architecture to the expected checksum of the binary.
var checksums = map[string]string{
	"darwin_amd64": "0b74d7efa2fe020c58dafbec5377617c1830f4ce8de26c0bbe8b57334984aab6",
	"darwin_arm64": "e3c880d28eae2a150f3f01a674b3cd6a130d6d3f16685740cd1e16538b58a4a5",
	"linux_amd64":  "baad3e1b06d6f310d210c93e95258a03d923fe610f8d0742138f2245f94abd7c",
	"linux_arm64":  "ac5f88c0f6c1e5ed09c035a9f4405f74b996ea8a701d7150dc0184c18dd09f11",
}

func getOS() string {
	sysOS := runtime.GOOS
	if runtime.GOOS == "windows" {
		sysOS = "win32"
	}
	return sysOS
}

func binaryURL(sysOS, sysArch string) string {
	return fmt.Sprintf("https://github.com/foundry-rs/foundry/releases/download/%s/foundry_%s_%s_%s.tar.gz", Version, Version, sysOS, sysArch)
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

type AutodetectBin struct {
	progressor ioutil.Progressor

	cachePather func() (string, error)
	checksummer func(r io.Reader) error
	url         string
	path        string
}

type AutodetectBinaryOpt func(s *AutodetectBin)

func WithProgressor(p ioutil.Progressor) AutodetectBinaryOpt {
	return func(s *AutodetectBin) {
		s.progressor = p
	}
}

func WithURL(url string) AutodetectBinaryOpt {
	return func(s *AutodetectBin) {
		s.url = url
	}
}

func WithCachePather(pather func() (string, error)) AutodetectBinaryOpt {
	return func(s *AutodetectBin) {
		s.cachePather = pather
	}
}

func WithChecksummer(checksummer func(r io.Reader) error) AutodetectBinaryOpt {
	return func(s *AutodetectBin) {
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

func AutodetectBinary(opts ...AutodetectBinaryOpt) (*AutodetectBin, error) {
	bin := &AutodetectBin{
		url:         binaryURL(getOS(), runtime.GOARCH),
		cachePather: homedirCachePather,
		checksummer: githubChecksummer,
	}
	for _, opt := range opts {
		opt(bin)
	}
	return bin, nil
}

func (b *AutodetectBin) Ensure(ctx context.Context) error {
	if b.path != "" {
		return nil
	}

	forgePath, err := exec.LookPath("forge")
	if err == nil {
		b.path = forgePath
		return nil
	}

	binDir, err := b.cachePather()
	if err != nil {
		return fmt.Errorf("could not provide cache dir: %w", err)
	}
	binPath := path.Join(binDir, "forge")
	_, err = os.Stat(binPath)
	if err == nil {
		b.path = binPath
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("could not stat %s: %w", binPath, err)
	}

	if err := b.downloadBinary(ctx, binDir); err != nil {
		return fmt.Errorf("could not download binary: %w", err)
	}
	b.path = binPath
	return nil
}

func (b *AutodetectBin) Path() string {
	return b.path
}

func (b *AutodetectBin) downloadBinary(ctx context.Context, dest string) error {
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
	if err := os.Rename(path.Join(tmpDir, "forge"), path.Join(dest, "forge")); err != nil {
		return fmt.Errorf("failed to move binary: %w", err)
	}
	return nil
}
