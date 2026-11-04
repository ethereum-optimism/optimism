package main

import (
	"archive/tar"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

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

// archiveTime is the modification time of every archived file, so that the archive's metadata
// doesn't depend on when or by whom it was built.
var archiveTime = time.Unix(0, 0)

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

	if err := writeArchive(f, absBase, getGitCommit(absBase)); err != nil {
		log.Fatalf("write archive: %v", err)
	}

	log.Printf("wrote %s", *outFile)
}

// writeArchive writes base as a zstd-compressed tarball to w. If commit is non-empty, it is added
// as a COMMIT file. On error the tarball is left unterminated, so that readers don't mistake it for
// a complete archive.
func writeArchive(w io.Writer, base string, commit string) error {
	gz, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(22)))
	if err != nil {
		return fmt.Errorf("create zstd writer: %w", err)
	}

	tw := tar.NewWriter(gz)

	if err := filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(base, path)
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
		hdr.ModTime = archiveTime
		hdr.Uid, hdr.Gid = 0, 0
		hdr.Uname, hdr.Gname = "", ""
		hdr.Mode = archiveMode(info)

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
		return fmt.Errorf("walk: %w", err)
	}

	if commit != "" {
		if err := addCommitFile(tw, commit); err != nil {
			return fmt.Errorf("add COMMIT file: %w", err)
		}
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("close tar: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("close zstd: %w", err)
	}
	return nil
}

// archiveMode returns the file mode to archive for info, independent of the builder's umask.
func archiveMode(info os.FileInfo) int64 {
	if info.IsDir() || info.Mode()&0o100 != 0 {
		return 0o755
	}
	return 0o644
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

func getGitCommit(baseDir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = baseDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func addCommitFile(tw *tar.Writer, commit string) error {
	hdr := &tar.Header{
		Name:     "COMMIT",
		Size:     int64(len(commit)),
		Mode:     0644,
		ModTime:  archiveTime,
		Typeflag: tar.TypeReg,
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	if _, err := tw.Write([]byte(commit)); err != nil {
		return err
	}

	log.Printf("a COMMIT")
	return nil
}
