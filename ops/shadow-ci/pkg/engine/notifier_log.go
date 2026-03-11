package engine

import (
	"fmt"
	"strings"
)

// LogNotifier is a fallback notifier that logs revert decisions to stdout.
type LogNotifier struct{}

// NotifyRevert logs the revert decision.
func (ln *LogNotifier) NotifyRevert(decision *RevertDecision) error {
	fmt.Printf("[auto-revert] ShouldRevert=%v Commit=%s PR=#%d Reason=%q Tests=[%s]\n",
		decision.ShouldRevert,
		decision.CulpritCommit,
		decision.CulpritPR,
		decision.Reason,
		strings.Join(decision.FailedTests, ", "),
	)
	return nil
}
