package batcher

import (
	"sync"
)

type JobFactory func() (func(), error)

type JobRunner struct {
	concurrency uint64
	started     func(uint64)
	finished    func(uint64)
	cond        *sync.Cond
	wg          sync.WaitGroup
	running     uint64
}

// NewJobRunner creates a new JobRunner, with the following parameters:
//   - concurrency: max number of jobs to run at once (0 == no limit)
//   - started / finished: called whenever a job starts or finishes. The
//     number of currently running jobs is passed as a parameter.
func NewJobRunner(concurrency uint64, started func(uint64), finished func(uint64)) *JobRunner {
	return &JobRunner{
		concurrency: concurrency,
		started:     started,
		finished:    finished,
		cond:        sync.NewCond(&sync.Mutex{}),
	}
}

// Wait waits on all running jobs to stop.
func (s *JobRunner) Wait() {
	s.wg.Wait()
}

// Run will wait until the number of running jobs is below the max concurrency,
// and then run the next job. The JobFactory should return `nil` if the next
// job does not exist. Returns the error returned from the JobFactory (if any).
func (s *JobRunner) Run(factory JobFactory) error {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	for s.full() {
		s.cond.Wait()
	}
	return s.tryRun(factory)
}

// TryRun runs the next job, but only if the number of running jobs is below the
// max concurrency, otherwise the JobFactory is not called (and nil is returned).
//
// The JobFactory should return `nil` if the next job does not exist. Returns
// the error returned from the JobFactory (if any).
func (s *JobRunner) TryRun(factory JobFactory) error {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	return s.tryRun(factory)
}

func (s *JobRunner) tryRun(factory JobFactory) error {
	if s.full() {
		return nil
	}
	job, err := factory()
	if err != nil {
		return err
	}
	if job == nil {
		return nil
	}

	s.running++
	s.started(s.running)
	s.wg.Add(1)
	go func() {
		defer func() {
			s.cond.L.Lock()
			s.running--
			s.finished(s.running)
			s.wg.Done()
			s.cond.L.Unlock()
			s.cond.Broadcast()
		}()
		job()
	}()
	return nil
}

func (s *JobRunner) full() bool {
	return s.concurrency > 0 && s.running >= s.concurrency
}
