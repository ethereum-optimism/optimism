package event

type ErrEvent struct {
	Err error
}

func (e ErrEvent) String() string {
	return "err-event"
}

var _ Event = ErrEvent{}
