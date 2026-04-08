package selector

import "fmt"

// Stage represents a point in the development lifecycle.
type Stage struct {
	Name     string
	MissCost float64 // seconds of engineer time wasted by a failure at this stage
}

// Miss costs in seconds:
//   save:        10s — you see the error immediately, trivial to fix
//   commit:     300s — 5 min to notice, fix, re-commit
//   pr:        1800s — 30 min to notice CI failure, context-switch, fix, re-push
//   merge_queue: 7200s — 2 hours: blocks the queue, everyone notices, high-pressure fix
//   develop:   86400s — essentially infinite: run everything, any miss is unacceptable
var (
	StageOnSave     = Stage{Name: "save", MissCost: 10}
	StageOnCommit   = Stage{Name: "commit", MissCost: 300}
	StageOnPR       = Stage{Name: "pr", MissCost: 1800}
	StageMergeQueue = Stage{Name: "merge_queue", MissCost: 7200}
	StageDevelop    = Stage{Name: "develop", MissCost: 86400}
)

var stages = map[string]Stage{
	"save":        StageOnSave,
	"commit":      StageOnCommit,
	"pr":          StageOnPR,
	"merge_queue": StageMergeQueue,
	"develop":     StageDevelop,
}

// StageByName returns a stage by name.
func StageByName(name string) (Stage, error) {
	s, ok := stages[name]
	if !ok {
		return Stage{}, fmt.Errorf("unknown stage %q (valid: save, commit, pr, merge_queue, develop)", name)
	}
	return s, nil
}

// AllStages returns all available stages in lifecycle order.
func AllStages() []Stage {
	return []Stage{StageOnSave, StageOnCommit, StageOnPR, StageMergeQueue, StageDevelop}
}
