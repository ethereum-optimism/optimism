package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts/bundle"
)

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

var (
	baseDir = flag.String("base", "", "directory to archive")
	outFile = flag.String("out", "", "path to output tzst")
	exclude multiFlag
)

func init() {
	flag.Var(&exclude, "exclude", "glob pattern to exclude (can be specified multiple times)")
}

// mktar creates a zstd-compressed tarball of the given base directory.
// It excludes certain directories and files that are not needed for the
// forge client. Additional exclusions can be specified via --exclude.
//
// Usage: mktar -base DIR -out FILE [--exclude pattern]...
//
// Example: mktar -base ../packages/contracts-bedrock -out ./pkg/deployer/artifacts/forge-artifacts/artifacts.tzst
//
// The base directory must be a built forge project. The tarball is its bundle, as
// bundle.Write writes it.
//
// Do not confuse this script with the ops/publish-artifacts.sh script, which is
// used to publish the tarball to GCS.
func main() {
	flag.Parse()

	if *baseDir == "" || *outFile == "" {
		log.Fatalf("usage: mktar -base DIR -out FILE")
	}
	if err := run(*baseDir, *outFile, exclude); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s", *outFile)
}

func run(baseDir string, outFile string, exclude []string) (err error) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("resolve base: %w", err)
	}

	info, err := os.Stat(absBase)
	if err != nil {
		return fmt.Errorf("stat base: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("base must be a directory: %s", absBase)
	}

	if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	f, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	zw, err := zstd.NewWriter(f, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(22)))
	if err != nil {
		return fmt.Errorf("create zstd writer: %w", err)
	}
	if err := bundle.Write(zw, absBase, exclude, gitCommit(absBase)); err != nil {
		_ = zw.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("close zstd: %w", err)
	}
	return nil
}

// gitCommit returns the commit of the git checkout at dir, or "" when there is none.
func gitCommit(dir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
