package selector

import "fmt"

// Stage represents a point in the development lifecycle.
type Stage struct {
	Name     string
	MissCost float64 // how expensive is a failure at this stage (relative units)
}

var (
	StageOnSave     = Stage{Name: "save", MissCost: 0.1}
	StageOnCommit   = Stage{Name: "commit", MissCost: 1.0}
	StageOnPR       = Stage{Name: "pr", MissCost: 5.0}
	StageMergeQueue = Stage{Name: "merge_queue", MissCost: 50.0}
	StageDevelop    = Stage{Name: "develop", MissCost: 1000.0}
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
