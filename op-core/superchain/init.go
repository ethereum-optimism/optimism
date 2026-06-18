package superchain

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	gethsuperchain "github.com/ethereum/go-ethereum/superchain"
)

// pinnedZipSHA is the sha256sum-format file pinning the expected SHA256 of the
// embedded superchain-configs.zip. Committed to git (the zip itself isn't), so
// any drift between built and approved bundles surfaces as a .sha256 diff.
//
//go:embed superchain-configs.zip.sha256
var pinnedZipSHA string

// VerifyEmbeddedCommit asserts the embedded bundle is consistent with what was
// approved in review. Both init() and TestSyncSuperchain call this so the same
// check runs at process startup and in CI tests.
//
// Two checks:
//  1. SHA256 of the embedded zip matches superchain-configs.zip.sha256 — catches
//     any drift (submodule bump without sync, non-deterministic build, tampering).
//  2. op-geth bundles the same SR commit as the embedded zip's COMMIT entry. The
//     COMMIT entry is written from the superchain-registry submodule at
//     `just sync-superchain` time and is the canonical commit this build was
//     generated from. Until the registry is decoupled from op-geth (#20257), both
//     copies must agree.
//
// A missing zip is caught at compile time by //go:embed, not here.
func VerifyEmbeddedCommit() error {
	// (1) SHA256 of embedded zip == committed .sha256 file.
	gotSHA := sha256.Sum256(builtInConfigData)
	gotSHAHex := hex.EncodeToString(gotSHA[:])
	expectedSHA := parseSHA256SumLine(pinnedZipSHA)
	if gotSHAHex != expectedSHA {
		return fmt.Errorf(
			"embedded superchain-configs.zip SHA256 is %s, but superchain-configs.zip.sha256 pins %s.\n"+
				"Run `just sync-superchain` and commit the updated .sha256 file.",
			gotSHAHex, expectedSHA)
	}

	// The commit the bundle was generated from, recorded inside the zip by
	// sync-superchain.sh from the superchain-registry submodule HEAD.
	embeddedCommit, err := readEmbeddedZipCommit()
	if err != nil {
		return err
	}

	// (2) Cross-check: op-geth still bundles its own copy of the superchain-registry
	// (until the decoupling is complete in #20257). The two commits must match.
	if gethCommit := gethsuperchain.EmbeddedRegistryCommit(); gethCommit != embeddedCommit {
		return fmt.Errorf(
			"op-core/superchain bundles superchain-registry commit %s but op-geth bundles commit %s.\n"+
				"Bump the superchain-registry submodule and run `just sync-superchain`,\n"+
				"or bump the op-geth replace directive in go.mod — whichever is the stale side.",
			embeddedCommit, gethCommit)
	}
	return nil
}

// readEmbeddedZipCommit returns the COMMIT entry recorded inside the embedded
// superchain-configs.zip (written from the superchain-registry submodule HEAD by
// sync-superchain.sh).
func readEmbeddedZipCommit() (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(builtInConfigData), int64(len(builtInConfigData)))
	if err != nil {
		return "", fmt.Errorf("opening embedded superchain-configs.zip: %w", err)
	}
	f, err := zr.Open("COMMIT")
	if err != nil {
		return "", fmt.Errorf("reading COMMIT entry from embedded zip: %w", err)
	}
	defer f.Close()
	raw, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("reading COMMIT entry contents: %w", err)
	}
	return strings.TrimSpace(string(raw)), nil
}

// parseSHA256SumLine extracts the hex digest from a sha256sum-format line:
// "<64-hex>  <filename>". Returns "" if the format is unexpected.
func parseSHA256SumLine(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func init() {
	if err := VerifyEmbeddedCommit(); err != nil {
		panic("op-core/superchain: " + err.Error())
	}
}
