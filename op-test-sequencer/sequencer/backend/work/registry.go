package work

import (
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type JobRegistry struct {
	jobs locks.RWMap[seqtypes.BuildJobID, BuildJob]
}

var _ Jobs = (*JobRegistry)(nil)

func NewJobRegistry() *JobRegistry {
	return &JobRegistry{}
}

func (ba *JobRegistry) RegisterJob(job BuildJob) error {
	if !ba.jobs.SetIfMissing(job.ID(), job) {
		return seqtypes.ErrConflictingJob
	}
	return nil
}

// GetJob returns nil if the job isn't known.
func (ba *JobRegistry) GetJob(id seqtypes.BuildJobID) BuildJob {
	job, _ := ba.jobs.Get(id)
	return job
}

func (ba *JobRegistry) UnregisterJob(id seqtypes.BuildJobID) {
	ba.jobs.Delete(id)
}

func (ba *JobRegistry) Clear() {
	ba.jobs.Clear()
}

func (ba *JobRegistry) Len() int {
	return ba.jobs.Len()
}
