package superchain

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"strings"
)

// pinnedZipSHA is the sha256sum-format file pinning the expected SHA256 of the
// embedded superchain-configs.zip. Committed to git (the zip itself isn't), so
// any drift between built and approved bundles surfaces as a .sha256 diff.
//
//go:embed superchain-configs.zip.sha256
var pinnedZipSHA string

// VerifyEmbeddedBundle asserts the embedded superchain-configs.zip matches what
// was approved in review: its SHA256 must equal the committed
// superchain-configs.zip.sha256. This catches any drift — a submodule bump
// without a re-sync, a non-deterministic build, or tampering. Both init() and
// TestSyncSuperchain call it so the same check runs at process startup and in CI
// tests.
//
// A missing zip is caught at compile time by //go:embed, not here.
func VerifyEmbeddedBundle() error {
	gotSHA := sha256.Sum256(builtInConfigData)
	gotSHAHex := hex.EncodeToString(gotSHA[:])
	expectedSHA := parseSHA256SumLine(pinnedZipSHA)
	if gotSHAHex != expectedSHA {
		return fmt.Errorf(
			"embedded superchain-configs.zip SHA256 is %s, but superchain-configs.zip.sha256 pins %s.\n"+
				"Run `just sync-superchain` and commit the updated .sha256 file.",
			gotSHAHex, expectedSHA)
	}
	return nil
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
	if err := VerifyEmbeddedBundle(); err != nil {
		panic("op-core/superchain: " + err.Error())
	}
}
