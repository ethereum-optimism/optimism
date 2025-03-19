package system2

import (
	"fmt"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

// T is a minimal subset of testing.T
// This can be implemented by tooling, or a *testing.T can be used directly.
// The T interface is only used in the system2 package for sanity-checks,
// where local failure there and then is preferable over bubbling up the error.
type T interface {
	Errorf(format string, args ...interface{})
	FailNow()
}

// This testing subset is sufficient for the require.Assertions to work.
var _ require.TestingT = T(nil)

// ToolingT is a T implementation that can be used in tooling,
// when the devnet-SDK is not used in a regular Go test.
type ToolingT struct {
	// Errors will be logged here.
	Log log.Logger
	// Fail will be called to register a critical failure.
	// The implementer can choose to panic, crit-log, exit, etc. as preferred.
	Fail func()
}

var _ T = (*ToolingT)(nil)

func (t *ToolingT) Errorf(format string, args ...interface{}) {
	t.Log.Error(fmt.Sprintf(format, args...))
}

func (t *ToolingT) FailNow() {
	t.Fail()
}
