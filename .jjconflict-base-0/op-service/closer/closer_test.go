package closer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloser(t *testing.T) {
	setup := func() (CloseFn, *string) {
		data := ""
		fn := CloseFn(func() {
			data += "world"
		})
		fn.Stack(func() {
			data += "hello "
		})
		fn.Stack(func() {
			data += "1234 "
		})
		return fn, &data
	}

	t.Run("closed", func(t *testing.T) {
		fn, data := setup()
		cancelClose, closeMaybe := fn.Maybe()
		closeMaybe()
		require.Equal(t, "1234 hello world", *data)
		cancelClose() // cancel is no-op after already closed
	})

	t.Run("canceled", func(t *testing.T) {
		fn, data := setup()
		cancelClose, closeMaybe := fn.Maybe()
		cancelClose()
		closeMaybe()
		require.Equal(t, "", *data)
	})
}
