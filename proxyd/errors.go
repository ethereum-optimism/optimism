package proxyd

import "fmt"

func wrapErr(err error, msg string) error {
	return fmt.Errorf("%s %v", msg, err)
}
