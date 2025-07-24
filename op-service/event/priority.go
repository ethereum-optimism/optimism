package event

import "fmt"

// Priority applies to events,
// to process the more important events first,
// when and if there is a synchronous choice.
// The default priority is 0, the normal.
type Priority int8

const (
	priorityMin            = -1
	Low           Priority = -1
	Normal        Priority = 0
	High          Priority = 1
	priorityMax            = 1
	priorityCount          = priorityMax + 1 - priorityMin
)

func (p Priority) String() string {
	switch p {
	case Low:
		return "low"
	case Normal:
		return "normal"
	case High:
		return "high"
	default:
		return fmt.Sprintf("unknown(%d)", int(p))
	}
}

func (p Priority) Valid() bool {
	return priorityMin <= p && p <= priorityMax
}
