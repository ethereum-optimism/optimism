package derive

import (
	"fmt"
)

// Level is the severity level of the error.
type Level uint

// There are three levels currently, out of which only 2 are being used
// to classify error by severity. LevelTemporary
const (
	// LevelTemporary is a temporary error for example due to an RPC or
	// connection issue, and can be safely ignored and retried by the caller
	LevelTemporary Level = iota
	// LevelReset is a pipeline reset error.
	LevelReset
	// LevelCritical is a critical error.
	LevelCritical
)

// Error is a wrapper for error, description and a severity level.
type Error struct {
	err   error
	desc  string
	level Level
}

// Error satisfiees the error interface.
func (e Error) Error() string {
	if e.err != nil {
		return fmt.Errorf("%w: %s", e.err, e.desc).Error()
	}
	return e.desc
}

// Unwrap satisfies the Is/As interface.
func (e Error) Unwrap() error {
	return e.err
}

// Is satisfies the error Unwrap interface.
func (e Error) Is(target error) bool {
	if target == nil {
		return e == target
	}
	err, ok := target.(Error)
	if !ok {
		return false
	}
	return e.level == err.level
}

// NewError returns a custom Error.
func NewError(err error, desc string, level Level) error {
	return Error{
		err:   err,
		desc:  desc,
		level: level,
	}
}

// NewTemporaryError returns a temporary error.
func NewTemporaryError(err error, desc string) error {
	return NewError(
		err,
		desc,
		LevelTemporary,
	)
}

// NewResetError returns a pipeline reset error.
func NewResetError(err error, desc string) error {
	return NewError(
		err,
		desc,
		LevelReset,
	)
}

// NewCrititalError returns a critical error.
func NewCrititalError(err error, desc string) error {
	return NewError(
		err,
		desc,
		LevelCritical,
	)
}

// Sentinel errors, use these to get the severity of errors by calling
// errors.Is(err, ErrTemporary) for example.
var ErrTemporary = NewTemporaryError(nil, "temporary error")
var ErrReset = NewResetError(nil, "pipeline reset error")
var ErrCritical = NewCrititalError(nil, "critical error")
