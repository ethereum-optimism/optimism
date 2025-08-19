package forge

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

// Version is the Foundry version that op-deployer will download if it's not found on PATH.
const Version = "v1.3.1"

func bindirName() string {
	sysOS := runtime.GOOS
	if runtime.GOOS == "windows" {
		sysOS = "win32"
	}
	sysArch := runtime.GOARCH

	return fmt.Sprintf("foundry_%s_%s_%s", Version, sysOS, sysArch)
}

func binaryURL() string {
	return fmt.Sprintf("https://github.com/foundry-rs/foundry/releases/download/%s/%s.tar.gz", Version, bindirName())
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

	cacheDirProvider func() (string, error)
	url              string
	path             string
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

func WithCacheDirProvider(provider func() (string, error)) AutodetectBinaryOpt {
	return func(s *AutodetectBin) {
		s.cacheDirProvider = provider
	}
}

func AutodetectBinary(opts ...AutodetectBinaryOpt) (*AutodetectBin, error) {
	bin := &AutodetectBin{
		url: binaryURL(),
		cacheDirProvider: func() (string, error) {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("could not find home directory: %w", err)
			}
			return path.Join(homeDir, ".op-deployer", "cache"), nil
		},
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

	binDir, err := b.cacheDirProvider()
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
	}
	buf := new(bytes.Buffer)
	if err := downloader.Download(ctx, b.url, buf); err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	gzr, err := gzip.NewReader(buf)
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
