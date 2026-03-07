package model

import "time"

// Target is the universal representation of something testable.
// It could be a Go package, a Solidity test file, a Rust crate, or a
// future language's equivalent. The core engine never looks inside it —
// it operates on targets through the adapter interfaces.
type Target struct {
	// Unique identifier within a language (e.g., Go import path, Sol file path, Rust crate name).
	ID string `json:"id"`

	// Which language adapter owns this target.
	Language string `json:"language"`

	// Build configurations this target should be tested under.
	Configurations []Configuration `json:"configurations,omitempty"`

	// Scoping classification.
	// "affected": run only when dependency graph says so.
	// "always": run on every PR regardless (safety-critical or ungraphable).
	// The default for new targets is "always" until proven safe to scope.
	Scope string `json:"scope"`

	// How confident the graph is that this target's dependencies are fully tracked.
	// 1.0 = perfect. 0.0 = unknown.
	// Targets below the confidence threshold are treated as "always".
	Confidence float64 `json:"confidence"`

	// Estimated runtime from historical data. Used for parallelism planning.
	EstimatedDuration time.Duration `json:"estimated_duration"`
}

// Configuration represents a build/test configuration variant.
type Configuration struct {
	// Human-readable name (e.g., "default", "CUSTOM_GAS_TOKEN", "interop").
	Name string `json:"name"`

	// Environment variables to set when running under this configuration.
	Env map[string]string `json:"env,omitempty"`
}

// ScopeAlways is the scope value for targets that always run.
const ScopeAlways = "always"

// ScopeAffected is the scope value for targets that run based on the dependency graph.
const ScopeAffected = "affected"
