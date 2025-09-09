package sequencing

import "time"

const (
	buildSealTimeout      = time.Second * 10
	buildStartTimeout     = time.Second * 10
	buildCancelTimeout    = time.Second * 10
	payloadProcessTimeout = time.Second * 10
	// sealingDuration defines the expected time it takes to seal the block
	sealingDuration = time.Millisecond * 50
)
