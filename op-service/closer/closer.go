package closer

// CloseFn types the given function to be used for closing of a resource.
type CloseFn func()

// Stack adds the given "stacked" function to run upon close.
// It will be called before this (method-receiver) function that it is stacked on top of.
func (fn *CloseFn) Stack(stacked func()) {
	self := *fn
	*fn = func() {
		stacked()
		self()
	}
}

// Maybe prepares a conditional close:
// it may be canceled by calling cancel, and the close call will then be a no-op.
// This helps e.g. constructors to close their resources,
// if there is some issue before constructor completion.
// But then cancel if the constructor is successful.
func (fn CloseFn) Maybe() (cancel func(), close func()) {
	do := true
	cancel = func() {
		do = false
	}
	close = func() {
		if do {
			fn()
		}
	}
	return
}
