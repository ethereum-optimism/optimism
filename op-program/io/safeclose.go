package io

import (
	"io"
	"sync"
)

type safeClose struct {
	c    io.Closer
	once sync.Once
}

func (s *safeClose) Close() error {
	var err error
	s.once.Do(func() {
		err = s.c.Close()
	})
	return err
}

func NewSafeClose(c io.Closer) io.Closer {
	return &safeClose{
		c: c,
	}
}
