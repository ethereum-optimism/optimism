package sequencing

import "errors"

var (
	ErrSequencerAlreadyStarted = errors.New("sequencer already running")
	ErrSequencerAlreadyStopped = errors.New("sequencer not running")
)
