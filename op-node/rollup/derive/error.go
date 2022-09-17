package derive

import (
	"errors"
	"fmt"
)

// Level is the severity level of the error.
type Level uint

func (lvl Level) String() string {
	switch lvl {
	case LevelTemporary:
		return "temp"
	case LevelReset:
		return "reset"
	case LevelCritical:
		return "crit"
	default:
		return fmt.Sprintf("unknown(%d)", lvl)
	}
}

// There are three levels currently, out of which only 2 are being used
// to classify error by severity. LevelTemporary
const (
	// LevelTemporary is a temporary error for example due to an RPC or
	// connection issue, and can be safely ignored and retried by the caller
	LevelTemporary Level = iota
	// LevelReset is a pipeline reset error. It must be treated like a reorg.
	LevelReset
	// LevelCritical is a critical error.
	LevelCritical
)

// Error is a wrapper for error, description and a severity level.
type Error struct {
	err   error
	level Level
}

// Error satisfies the error interface.
func (e Error) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.level, e.err)
	}
	return e.level.String()
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
func NewError(err error, level Level) error {
	return Error{
		err:   err,
		level: level,
	}
}

// NewTemporaryError returns a temporary error.
func NewTemporaryError(err error) error {
	return NewError(err, LevelTemporary)
}

// NewResetError returns a pipeline reset error.
func NewResetError(err error) error {
	return NewError(err, LevelReset)
}

// NewCriticalError returns a critical error.
func NewCriticalError(err error) error {
	return NewError(err, LevelCritical)
}

// Sentinel errors, use these to get the severity of errors by calling
// errors.Is(err, ErrTemporary) for example.
var ErrTemporary = NewTemporaryError(nil)
var ErrReset = NewResetError(nil)
var ErrCritical = NewCriticalError(nil)

// NotEnoughData implies that the function currently does not have enough data to progress
// but if it is retried enough times, it will eventually return a real value or io.EOF
var NotEnoughData = errors.New("not enough data")
