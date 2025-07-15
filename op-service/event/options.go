package event

import "golang.org/x/time/rate"

type ExecutorConfig struct {
	// Priority. Higher = more important.
	// For synchronous executors this may help decide which deriver receives the event first.
	Priority Priority
}

// WithExecPriority sets the executor priority. Higher = more important.
// This directs which deriver first executes an event, if there is a synchronous choice.
func WithExecPriority(priority Priority) RegisterOption {
	return func(cfg *RegisterConfig) {
		cfg.Executor.Priority = priority
	}
}

type EmitterConfig struct {
	Limiting  bool
	Rate      rate.Limit
	Burst     int
	OnLimited func()

	// Priority. Higher = more important.
	// Events from more important emitters will be prioritized
	// for execution over queued up events with lower priority.
	Priority Priority
}

func WithEmitLimiter(rate rate.Limit, burst int, onLimited func()) RegisterOption {
	return func(cfg *RegisterConfig) {
		cfg.Emitter.Limiting = true
		cfg.Emitter.Rate = rate
		cfg.Emitter.Burst = burst
		cfg.Emitter.OnLimited = onLimited
	}
}

func WithNoEmitLimiter() RegisterOption {
	return func(cfg *RegisterConfig) {
		cfg.Emitter.Limiting = false
	}
}

// WithEmitPriority sets the emitter priority. Higher = more important.
// This directs when an emitted event is processed:
// it may be prioritized over other emitters of lesser importance.
func WithEmitPriority(priority Priority) RegisterOption {
	return func(cfg *RegisterConfig) {
		cfg.Emitter.Priority = priority
	}
}

// RegisterConfig represents the set of parameters to configure a
// new deriver/emitter with that is registered with an event System.
// These options may be reused for multiple registrations.
type RegisterConfig struct {
	Executor ExecutorConfig
	Emitter  EmitterConfig
}

type RegisterOption func(cfg *RegisterConfig)

// 10,000 events per second is plenty.
// If we are going through more events, the driver needs to breathe, and warn the user of a potential issue.
const eventsLimit = rate.Limit(10_000)

// 500 events of burst: the maximum amount of events to eat up
// past the rate limit before the rate limit becomes applicable.
const eventsBurst = 500

func defaultRegisterConfig() *RegisterConfig {
	return &RegisterConfig{
		Executor: ExecutorConfig{
			Priority: Normal,
		},
		Emitter: EmitterConfig{
			Limiting:  true,
			Rate:      eventsLimit,
			Burst:     eventsBurst,
			OnLimited: nil,
			Priority:  Normal,
		},
	}
}
