package sequencerrecovery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/log"
)

// ErrNoCanonicalSequencer means no reachable voting sequencer is on the
// supernode's chain, so there is nothing safe to restart from.
var ErrNoCanonicalSequencer = errors.New("no reachable voting sequencer matches the supernode")

// Step is one action of a recovery plan.
type Step struct {
	Desc string
	Run  func(ctx context.Context) error
	// Cleanup steps still run after an earlier step fails, so a failed
	// recovery never leaves the conductors paused.
	Cleanup bool
}

// RecoverConfig sets how long recovery waits for each condition.
type RecoverConfig struct {
	// PollInterval is how often waits re-read the cluster.
	PollInterval time.Duration
	// Timeout bounds each wait.
	Timeout time.Duration
	// AdvanceBlocks is how many blocks the restarted sequencer must build, and
	// commit to raft, before the conductors resume.
	AdvanceBlocks uint64
	// CaughtUpWithin is how close to wall-clock time the restarted sequencer's
	// unsafe head must be before the conductors resume. A sequencer restarted
	// behind the clock builds catch-up blocks with old timestamps, and a resumed
	// conductor would judge it unhealthy and hand leadership to a standby that
	// lacks those blocks, which restarts the stall.
	CaughtUpWithin time.Duration
}

// Plan returns the steps that bring the cluster back to one active sequencer on
// the canonical chain, led by raft. It returns no steps when sequencing is
// healthy, even if some members have problems of their own.
func (c *Cluster) Plan(st *Status, rc RecoverConfig) ([]Step, error) {
	if len(st.SequencingProblems()) == 0 {
		return nil, nil
	}
	target, err := pickTarget(c.Members, st)
	if err != nil {
		return nil, err
	}
	statusOf := func(id string) MemberStatus {
		for _, ms := range st.Members {
			if ms.ID == id {
				return ms
			}
		}
		return MemberStatus{ID: id}
	}

	var steps []Step
	for _, m := range c.Members {
		if ms := statusOf(m.ID); ms.Err == nil && ms.Paused {
			continue
		}
		steps = append(steps, Step{
			Desc: fmt.Sprintf("pause conductor %s", m.ID),
			Run:  m.Conductor.Pause,
		})
	}
	for _, m := range c.Members {
		if m.ID == target.ID || !statusOf(m.ID).SequencerActive {
			continue
		}
		steps = append(steps, Step{
			Desc: fmt.Sprintf("stop sequencer %s", m.ID),
			Run: func(ctx context.Context) error {
				_, err := m.Node.StopSequencer(ctx)
				if err != nil && strings.Contains(err.Error(), "sequencer not running") {
					return nil // it stopped on its own (e.g. the conductor handed off leadership)
				}
				return err
			},
		})
	}
	// Leadership can move until the conductors are paused (an unhealthy leader
	// hands it off), so check it after pausing rather than trusting st.
	steps = append(steps, Step{
		Desc: fmt.Sprintf("make %s the raft leader (transfer if leadership moved)", target.ID),
		Run: func(ctx context.Context) error {
			var leaderID string
			err := c.waitFor(ctx, rc, "a raft leader", func(ctx context.Context) (bool, error) {
				info, err := target.Conductor.LeaderWithID(ctx)
				if err != nil || info == nil || info.ID == "" {
					return false, nil
				}
				leaderID = info.ID
				return true, nil
			})
			if err != nil {
				return err
			}
			if leaderID == target.ID {
				return nil
			}
			leader, ok := memberByID(c.Members, leaderID)
			if !ok {
				return fmt.Errorf("raft leader %q is not a configured sequencer", leaderID)
			}
			if err := leader.Conductor.TransferLeaderToServer(ctx, target.ID, target.RaftAddr); err != nil {
				return fmt.Errorf("transfer leadership from %s: %w", leader.ID, err)
			}
			return c.waitFor(ctx, rc, "raft leader "+target.ID, func(ctx context.Context) (bool, error) {
				info, err := target.Conductor.LeaderWithID(ctx)
				return err == nil && info != nil && info.ID == target.ID, nil
			})
		},
	})
	if !statusOf(target.ID).SequencerActive {
		steps = append(steps, Step{
			Desc: fmt.Sprintf("start sequencer %s from its own unsafe head", target.ID),
			Run: func(ctx context.Context) error {
				ss, err := target.Node.SyncStatus(ctx)
				if err != nil {
					return err
				}
				return target.Node.StartSequencer(ctx, ss.UnsafeL2.Hash)
			},
		})
	}
	steps = append(steps, Step{
		Desc: fmt.Sprintf("wait for %s to build %d blocks and catch up to within %s of now", target.ID, rc.AdvanceBlocks, rc.CaughtUpWithin),
		Run: func(ctx context.Context) error {
			ss, err := target.Node.SyncStatus(ctx)
			if err != nil {
				return err
			}
			want := ss.UnsafeL2.Number + rc.AdvanceBlocks
			return c.waitFor(ctx, rc, fmt.Sprintf("%s unsafe head %d, caught up", target.ID, want), func(ctx context.Context) (bool, error) {
				ss, err := target.Node.SyncStatus(ctx)
				if err != nil {
					return false, nil
				}
				age := c.Now().Sub(time.Unix(int64(ss.UnsafeL2.Time), 0))
				return ss.UnsafeL2.Number >= want && age <= rc.CaughtUpWithin, nil
			})
		},
	})
	for _, m := range c.Members {
		steps = append(steps, Step{
			Desc:    fmt.Sprintf("resume conductor %s", m.ID),
			Run:     m.Conductor.Resume,
			Cleanup: true,
		})
	}
	steps = append(steps, Step{
		Desc: "wait for sequencing to report healthy",
		Run: func(ctx context.Context) error {
			return c.waitFor(ctx, rc, "healthy sequencing", func(ctx context.Context) (bool, error) {
				st, err := c.Status(ctx)
				if err != nil {
					return false, nil
				}
				return len(st.SequencingProblems()) == 0, nil
			})
		},
	})
	return steps, nil
}

// Run executes the steps in order. After a failure it stops, but still runs
// the remaining cleanup steps (resuming the conductors), then returns the error.
func Run(ctx context.Context, lgr log.Logger, steps []Step) error {
	for i, s := range steps {
		lgr.Info("Running step", "step", i+1, "of", len(steps), "desc", s.Desc)
		if err := s.Run(ctx); err != nil {
			failed := fmt.Errorf("step %d (%s): %w", i+1, s.Desc, err)
			for _, c := range steps[i+1:] {
				if !c.Cleanup {
					continue
				}
				lgr.Warn("Running cleanup after failure", "desc", c.Desc)
				if cerr := c.Run(ctx); cerr != nil {
					failed = errors.Join(failed, fmt.Errorf("cleanup (%s): %w", c.Desc, cerr))
				}
			}
			return failed
		}
	}
	return nil
}

// pickTarget prefers the raft leader, so no leadership transfer is needed, then
// the canonical voter with the highest unsafe head.
func pickTarget(members []Member, st *Status) (Member, error) {
	var best *MemberStatus
	for i := range st.Members {
		ms := &st.Members[i]
		m, ok := memberByID(members, ms.ID)
		if !ok || !m.Voting || ms.Err != nil || !ms.Canonical {
			continue
		}
		if ms.ID == st.LeaderID {
			return m, nil
		}
		if best == nil || ms.Sync.UnsafeL2.Number > best.Sync.UnsafeL2.Number {
			best = ms
		}
	}
	if best == nil {
		return Member{}, ErrNoCanonicalSequencer
	}
	m, _ := memberByID(members, best.ID)
	return m, nil
}

func memberByID(members []Member, id string) (Member, bool) {
	for _, m := range members {
		if m.ID == id {
			return m, true
		}
	}
	return Member{}, false
}

func (c *Cluster) waitFor(ctx context.Context, rc RecoverConfig, what string, cond func(context.Context) (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, rc.Timeout)
	defer cancel()
	for {
		ok, err := cond(ctx)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for %s: %w", what, ctx.Err())
		case <-time.After(rc.PollInterval):
		}
	}
}
