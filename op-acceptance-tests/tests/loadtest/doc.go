// Package loadtest provides helpers for acceptance tests that put transaction load on a chain:
// funding many spammer EOAs (FundEOAs), rotating between them (RoundRobin), and schedules that
// decide how often to invoke a Spammer (Burst, Constant).
package loadtest
