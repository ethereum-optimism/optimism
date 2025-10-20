package main

import (
	"archive/tar"
	"flag"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
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
// The output file will be a zstd-compressed tarball of the given base directory.
// Do not confuse this script with the ops/publish-artifacts.sh script, which is
// used to publish the tarball to GCS.
func main() {
	flag.Parse()

	if *baseDir == "" || *outFile == "" {
		log.Fatalf("usage: mktar -base DIR -out FILE")
	}

	absBase, err := filepath.Abs(*baseDir)
	if err != nil {
		log.Fatalf("resolve base: %v", err)
	}

	info, err := os.Stat(absBase)
	if err != nil {
		log.Fatalf("stat base: %v", err)
	}
	if !info.IsDir() {
		log.Fatalf("base must be a directory: %s", absBase)
	}

	if err := os.MkdirAll(filepath.Dir(*outFile), 0o755); err != nil {
		log.Fatalf("create output directory: %v", err)
	}

	f, err := os.Create(*outFile)
	if err != nil {
		log.Fatalf("create output file: %v", err)
	}
	defer f.Close()

	gz, err := zstd.NewWriter(f, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		log.Fatalf("create zstd writer: %v", err)
	}
	defer func() {
		if err := gz.Close(); err != nil {
			log.Fatalf("close zstd: %v", err)
		}
	}()

	tw := tar.NewWriter(gz)
	defer func() {
		if err := tw.Close(); err != nil {
			log.Fatalf("close tar: %v", err)
		}
	}()

	if err := filepath.WalkDir(absBase, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(absBase, path)
		if err != nil {
			return err
		}

		if shouldExclude(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if rel == "." {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(info, linkTarget(path, info))
		if err != nil {
			return err
		}

		hdr.Name = filepath.ToSlash(rel)
		hdr.ModTime = info.ModTime()
		hdr.AccessTime = info.ModTime()

		// tar-like progress output
		log.Printf("a %s", hdr.Name)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		log.Fatalf("walk: %v", err)
	}

	if err := tw.Flush(); err != nil {
		log.Fatalf("flush tar: %v", err)
	}

	log.Printf("wrote %s", *outFile)
}

func shouldExclude(rel string, d os.DirEntry) bool {
	if rel == "." {
		return false
	}

	rel = filepath.ToSlash(rel)

	for _, pattern := range exclude {
		pattern = filepath.ToSlash(pattern)
		if matchPattern(pattern, rel) {
			return true
		}
	}

	if strings.HasPrefix(rel, "book/") || rel == "book" {
		return true
	}
	if strings.HasPrefix(rel, "snapshots/") || rel == "snapshots" {
		return true
	}

	if !d.IsDir() {
		if strings.HasSuffix(d.Name(), ".t.sol") {
			return true
		}
	}

	return false
}

func matchPattern(pattern, rel string) bool {
	matched, err := path.Match(pattern, rel)
	if err != nil {
		log.Fatalf("invalid --exclude pattern %q: %v", pattern, err)
	}
	if matched {
		return true
	}

	if !strings.HasSuffix(pattern, "/") {
		pattern = pattern + "/"
	}
	matched, err = path.Match(pattern+"*", rel+"/")
	if err != nil {
		log.Fatalf("invalid --exclude pattern %q: %v", pattern, err)
	}
	return matched
}

func linkTarget(path string, info os.FileInfo) string {
	if info.Mode()&os.ModeSymlink == 0 {
		return ""
	}
	target, err := os.Readlink(path)
	if err != nil {
		log.Fatalf("readlink %s: %v", path, err)
	}
	return target
}
