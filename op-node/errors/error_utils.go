// Package errors provides enhanced error handling utilities for op-node.
package errors

import (
	"fmt"
	"runtime"

	"github.com/ethereum/go-ethereum/log"
)

// ErrorWithContext wraps an error with additional context information.
type ErrorWithContext struct {
	Err         error
	Component   string
	Operation   string
	File        string
	Line        int
	Context     map[string]interface{}
}

// Error implements the error interface.
func (e *ErrorWithContext) Error() string {
	if e.Context == nil || len(e.Context) == 0 {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Component, e.Operation, e.File, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s:%d: %v (context: %+v)", e.Component, e.Operation, e.File, e.Line, e.Err, e.Context)
}

// Unwrap returns the underlying error.
func (e *ErrorWithContext) Unwrap() error {
	return e.Err
}

// NewErrorWithContext creates a new error with context information.
func NewErrorWithContext(err error, component, operation string, context map[string]interface{}) *ErrorWithContext {
	if err == nil {
		return nil
	}
	
	_, file, line, _ := runtime.Caller(1)
	return &ErrorWithContext{
		Err:       err,
		Component: component,
		Operation: operation,
		File:      file,
		Line:      line,
		Context:   context,
	}
}

// LogError logs an error with structured logging and context.
func LogError(logger log.Logger, err error, component, operation string, context map[string]interface{}) {
	if err == nil {
		return
	}
	
	errWithCtx := NewErrorWithContext(err, component, operation, context)
	
	// Build log fields
	fields := []interface{}{
		"error", errWithCtx.Err.Error(),
		"component", component,
		"operation", operation,
	}
	
	if errWithCtx.Context != nil {
		for k, v := range errWithCtx.Context {
			fields = append(fields, k, v)
		}
	}
	
	logger.Error("Operation failed", fields...)
}

// WrapError wraps an error with additional context and returns a new error.
func WrapError(err error, component, operation, message string, context map[string]interface{}) error {
	if err == nil {
		return nil
	}
	
	wrappedErr := fmt.Errorf("%s: %w", message, err)
	return NewErrorWithContext(wrappedErr, component, operation, context)
}

