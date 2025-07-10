package event

import (
	"reflect"
)

// HandlerKey identifies what a handler processes.
// A handler-key may be catch-all; all applicable keys are yielded by iterating the HandledBy.
type HandlerKey struct {
	// EventType returns the type of event to handle.
	// If the handler is not type-specific to one event,
	// then the general Event interface type itself is returned as type.
	//
	// The EventType is used as compile-time scope for a handler.
	//
	// The reflect.Type interface is safe to use as map-key
	// (reflect package type-for/of functions return global type pointers specific per type).
	EventType reflect.Type

	// Domain identifies the runtime scope for a handler.
	// If the handler is not domain-specific, UndefinedDomain is used.
	Domain Domain

	// Task is not UndefinedTask (zero) when we are handling a task response,
	// i.e. this is a short-lived handler.
	Task TaskID
}

var genericEventType = reflect.TypeFor[Event]()

func (k HandlerKey) taskMods(yield func(id TaskID) bool) {
	if k.Task != UndefinedTask {
		if !yield(k.Task) {
			return
		}
	}
	yield(UndefinedTask)
}

func (k HandlerKey) eventTypeMods(yield func(evType reflect.Type) bool) {
	if k.EventType != genericEventType {
		if !yield(k.EventType) {
			return
		}
	}
	yield(genericEventType)
}

func (k HandlerKey) domainMods(yield func(domain Domain) bool) {
	if k.Domain != UndefinedDomain {
		if !yield(k.Domain) {
			return
		}
	}
	yield(UndefinedDomain)
}

// HandledBy iterates over aver all versions of the key that it may be accepted as.
// I.e. this widens the key to more generic keys, to find applicable handlers to run.
func (k HandlerKey) HandledBy(yield func(key HandlerKey) bool) {
	for taskMod := range k.taskMods {
		for eventTypeMod := range k.eventTypeMods {
			for domainMod := range k.domainMods {
				if !yield(HandlerKey{
					EventType: eventTypeMod,
					Domain:    domainMod,
					Task:      taskMod,
				}) {
					return
				}
			}
		}
	}
}
