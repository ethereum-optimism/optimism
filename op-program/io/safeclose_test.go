package io

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOnlyCallsCloseOnce(t *testing.T) {
	delegate := new(mockCloser)
	defer delegate.AssertExpectations(t)

	safeClose := NewSafeClose(delegate)
	// Only expects one close call
	delegate.ExpectClose(nil)

	require.NoError(t, safeClose.Close())
	require.NoError(t, safeClose.Close())
}

func TestReturnsErrorFromFirstCall(t *testing.T) {
	delegate := new(mockCloser)
	defer delegate.AssertExpectations(t)

	safeClose := NewSafeClose(delegate)
	err := errors.New("expected")
	// Only expects one close call
	delegate.ExpectClose(err)

	require.ErrorIs(t, safeClose.Close(), err)
	// Later calls should not return an error as they didn't need to call Close
	require.NoError(t, safeClose.Close())
}

type mockCloser struct {
	mock.Mock
}

func (t *mockCloser) Close() error {
	err := t.Mock.MethodCalled("Close").Get(0)
	if err != nil {
		return err.(error)
	}
	return nil
}

func (t *mockCloser) ExpectClose(err error) {
	t.Mock.On("Close").Return(err)
}
