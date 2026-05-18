package stack

type Lifecycle interface {
	Start()
	Stop()
}

// OnDiskStateWiper is an optional interface implemented by components that
// persist state on disk. Test helpers call WipeOnDiskState between Stop and
// Start to force a genuine cold-start restart.
type OnDiskStateWiper interface {
	WipeOnDiskState() error
}
