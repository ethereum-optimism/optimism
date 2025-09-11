package node

import (
	"errors"
)

var errNodeHalt = errors.New("opted to halt, unprepared for protocol change")
