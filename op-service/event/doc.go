/*
Package event implements an event system.

The executor choice is flexible,
such that components can run in a synchronous environment (like a fault-proof),
as well as a parallel environment (node).

Summary:
  - Event: interface, likely some user struct, to be communicated.
  - HandlerKey.EventType: events may be filtered by type (compile-time scope), for directed routing.
  - HandlerKey.Domain: events may be filtered by domain (run-time scope), for directed routing.
  - HandlerOption: customizes a handler.
  - Fn: most common handler option, specifying what function to run.
  - ErrFn: for event-handlers with default error return handling.
  - Handler: handlers process events.
  - Emitter: inserts an event into the system. Errors if not accepted by the system.
  - Actor: a named actor that may handle and/or emit some events.
  - Bus: an interface to handle/emit, that provides an ordering guarantee.
  - Executor: schedules and runs all events, provides actors with a Bus.
  - Tracer: monitors what is emitted and processed, for metrics and debugging.
  - System: collection of actors that share an executor.
  - Bus.Watch: sets up simple handlers, that just observe events.
  - Bus.Task: sets up handling of events with a processing-completion state.
    There can only be one task-handler per HandlerKey globally.
  - Resolve: Completes a task with an event. An event may be watched by many, but resolved only once.
  - Reject: Updates the task-completion with an error, does not emit an event.
  - Then: Emits an event, and registers short-lived handlers to follow resolution.
  - Await: returns an AwaitCase, a handler for a specific Then outcome, to block on.
*/
package event
