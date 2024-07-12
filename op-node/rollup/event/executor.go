package event

type Executable interface {
	RunEvent(ev AnnotatedEvent)
}

// ExecutableFunc implements the Executable interface as a function,
// similar to how the std-lib http HandlerFunc implements a Handler.
// This can be used for small in-place executables, test helpers, etc.
type ExecutableFunc func(ev AnnotatedEvent)

func (fn ExecutableFunc) RunEvent(ev AnnotatedEvent) {
	fn(ev)
}

type Executor interface {
	Add(d Executable, opts *ExecutorOpts) (leaveExecutor func())
	Enqueue(ev AnnotatedEvent) error
}
