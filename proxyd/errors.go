package proxyd

import "fmt"

func wrapErr(err error, msg string) error {
	return fmt.Errorf("%s %w", msg, err)
}
