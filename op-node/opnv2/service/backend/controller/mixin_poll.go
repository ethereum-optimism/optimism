package controller

import (
	"time"
)

type PollProvider interface {
	NeedPoll(interval time.Duration, now time.Time) bool
}

type pollState struct {
	forcePoll bool
	lastPoll  time.Time
}

func (p *pollState) ForcePoll() {
	p.forcePoll = true
}

func (p *pollState) RegisterPoll(now time.Time) {
	p.lastPoll = now
	p.forcePoll = false
}

func (p *pollState) NeedPoll(interval time.Duration, now time.Time) bool {
	return p.forcePoll || p.lastPoll.Add(interval).Before(now)
}

var _ PollProvider = (*pollState)(nil)

func NeedsPoll[V PollProvider](interval time.Duration, now time.Time) Predicate[V] {
	return func(v V) bool {
		return v.NeedPoll(interval, now)
	}
}
