package testutils

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func currentBranch() (string, error) {
	// CircleCI sometimes checks out the branch then changes it to something else (I've seen it show
	// up as "ranch" in some cases). This is probably a bug in CircleCI, but we can work around it
	// by using the CIRCLE_BRANCH env var if it's available. This is always set to the branch that
	// CircleCI is currently checking out.
	circleBranch := os.Getenv("CIRCLE_BRANCH")
	if circleBranch != "" {
		return circleBranch, nil
	}

	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RunOnBranch skips the current test unless the current Git branch matches the provided regex.
// This method shells out to git. Any failures running git are considered test failures.
func RunOnBranch(t *testing.T, re *regexp.Regexp) {
	t.Helper()

	branch, err := currentBranch()
	require.NoError(t, err, "could not get current branch")

	if !re.MatchString(branch) {
		t.Skipf("branch %s does not match %s, skipping", branch, re.String())
	}
}
