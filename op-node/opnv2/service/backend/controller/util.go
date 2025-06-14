package controller

import "iter"

// Predicate is a function that returns true/false for a given value.
type Predicate[V any] func(V) bool

// Filter wraps the given sequence, and filters it with the given predicates.
// The outer iterator will hide the inner iterated values that do not match the predicates.
func Filter[V any](seq iter.Seq[V], predicates ...Predicate[V]) iter.Seq[V] {
	return func(yield func(V) bool) {
	mainLoop:
		for v := range seq {
			for _, predicate := range predicates {
				if !predicate(v) {
					continue mainLoop
				}
			}
			if !yield(v) {
				return
			}
		}
	}
}

// First is like a single iter.Pull with close.
// It returns the first entry of the sequence, if any.
func First[V any](seq iter.Seq[V]) (v V, ok bool) {
	for entry := range seq {
		return entry, true
	}
	return
}
