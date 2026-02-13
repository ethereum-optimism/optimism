package engine

import "time"

const (
	buildSealTimeout      = time.Second * 10
	buildStartTimeout     = time.Second * 10
	buildCancelTimeout    = time.Second * 10
	payloadProcessTimeout = time.Second * 10
)
