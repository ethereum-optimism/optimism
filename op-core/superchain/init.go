package superchain

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"

	gethsuperchain "github.com/ethereum/go-ethereum/superchain"
)

// VerifyEmbeddedCommit asserts the embedded bundle agrees with op-geth's
// embedded copy of the superchain-registry.
//
// The single source of truth for which SR commit this build uses is the
// submodule at packages/contracts-bedrock/lib/superchain-registry; that SHA
// is resolved at build time by `just sync-superchain` and baked into the
// `COMMIT` entry of the embedded zip. At runtime we cross-check that op-geth
// bundles the same SR commit -- until the registry is decoupled from op-geth
// (issue #20257), both copies must agree.
//
// A missing zip is caught at compile time by //go:embed (in chain.go), not
// here.
func VerifyEmbeddedCommit() error {
	embeddedCommit, err := readEmbeddedCommit()
	if err != nil {
		return err
	}
	if gethCommit := gethsuperchain.EmbeddedRegistryCommit(); gethCommit != embeddedCommit {
		return fmt.Errorf(
			"op-core/superchain bundle is at commit %s but op-geth bundles commit %s.\n"+
				"Bump the superchain-registry submodule (`git -C packages/contracts-bedrock/lib/superchain-registry checkout <commit>`)\n"+
				"and run `just sync-superchain`, or bump the op-geth replace directive in go.mod -- whichever is the stale side.",
			embeddedCommit, gethCommit)
	}
	return nil
}

func readEmbeddedCommit() (string, error) {
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

func init() {
	if err := VerifyEmbeddedCommit(); err != nil {
		panic("op-core/superchain: " + err.Error())
	}
}
