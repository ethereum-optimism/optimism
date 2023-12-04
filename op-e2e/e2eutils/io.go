package e2eutils

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TempDir(t *testing.T) string {
	// Drop unusual characters (such as path separators or
	// characters interacting with globs) from the directory name to
	// avoid surprising os.MkdirTemp behavior.
	// Taken from the t.TempDir() implementation in the standard library.
	mapper := func(r rune) rune {
		if r < utf8.RuneSelf {
			const allowed = "!#$%&()+,-.=@^_{}~ "
			if '0' <= r && r <= '9' ||
				'a' <= r && r <= 'z' ||
				'A' <= r && r <= 'Z' {
				return r
			}
			if strings.ContainsRune(allowed, r) {
				return r
			}
		} else if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return -1
	}

	dir, err := os.MkdirTemp("", strings.Map(mapper, fmt.Sprintf("op-e2e-%s", t.Name())))
	require.NoError(t, err)

	t.Cleanup(func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Logf("Error removing temp dir %s: %s", dir, err)
		}
	})

	return dir
}
