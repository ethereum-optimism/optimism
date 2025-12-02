package ioutil

import (
	"errors"
	"io"
)

// MultiCloser is a simple type that supports combining multiple io.Closer instances into a single io.Closer
type MultiCloser []io.Closer

func (m MultiCloser) Close() error {
	var err error
	for _, c := range m {
		if e := c.Close(); e != nil {
			err = errors.Join(err, e)
		}
	}
	return err
}
