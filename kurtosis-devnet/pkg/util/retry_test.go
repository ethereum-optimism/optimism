package util

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithRetry(t *testing.T) {
	tests := []struct {
		name                string
		operation           string
		setupFunc           func() func() (string, error)
		expectedResult      string
		expectError         bool
		expectedErrorString string
		expectedAttempts    int
		timeout             time.Duration
	}{
		{
			name:      "success on first attempt",
			operation: "test-operation",
			setupFunc: func() func() (string, error) {
				return func() (string, error) {
					return "success", nil
				}
			},
			expectedResult:   "success",
			expectError:      false,
			expectedAttempts: 1,
			timeout:          5 * time.Second,
		},
		{
			name:      "success on second attempt",
			operation: "test-operation",
			setupFunc: func() func() (string, error) {
				attempts := 0
				return func() (string, error) {
					attempts++
					if attempts < 2 {
						return "", errors.New("temporary failure")
					}
					return "success", nil
				}
			},
			expectedResult:   "success",
			expectError:      false,
			expectedAttempts: 2,
			timeout:          10 * time.Second,
		},
		{
			name:      "success on third attempt",
			operation: "test-operation",
			setupFunc: func() func() (string, error) {
				attempts := 0
				return func() (string, error) {
					attempts++
					if attempts < 3 {
						return "", errors.New("temporary failure")
					}
					return "success", nil
				}
			},
			expectedResult:   "success",
			expectError:      false,
			expectedAttempts: 3,
			timeout:          15 * time.Second,
		},
		{
			name:      "failure after all attempts",
			operation: "test-operation",
			setupFunc: func() func() (string, error) {
				return func() (string, error) {
					return "", errors.New("persistent failure")
				}
			},
			expectedResult:      "",
			expectError:         true,
			expectedErrorString: "test-operation failed after 3 attempts: persistent failure",
			expectedAttempts:    3,
			timeout:             15 * time.Second,
		},
		{
			name:      "different return type - int",
			operation: "integer-operation",
			setupFunc: func() func() (string, error) {
				// Note: This test uses string but we'll test int in a separate function
				return func() (string, error) {
					return "42", nil
				}
			},
			expectedResult:   "42",
			expectError:      false,
			expectedAttempts: 1,
			timeout:          5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			fn := tt.setupFunc()
			start := time.Now()

			result, err := WithRetry(ctx, tt.operation, fn)

			duration := time.Since(start)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorString)
				assert.Equal(t, tt.expectedResult, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			// Verify timing for multi-attempt scenarios
			if tt.expectedAttempts > 1 {
				// Expected minimum time: (attempt-1) * 2 + (attempt-2) * 4 seconds
				// For 2 attempts: 2 seconds minimum
				// For 3 attempts: 2 + 4 = 6 seconds minimum
				expectedMinDuration := time.Duration(0)
				for i := 1; i < tt.expectedAttempts; i++ {
					expectedMinDuration += time.Duration(i*2) * time.Second
				}

				if tt.expectError && tt.expectedAttempts == 3 {
					// Should take at least 6 seconds for 3 failed attempts
					assert.True(t, duration >= expectedMinDuration,
						"Expected at least %v but took %v", expectedMinDuration, duration)
				} else if !tt.expectError && tt.expectedAttempts > 1 {
					// Should take at least the retry delay time
					assert.True(t, duration >= expectedMinDuration,
						"Expected at least %v but took %v", expectedMinDuration, duration)
				}
			}
		})
	}
}

func TestWithRetryContextCancellation(t *testing.T) {
	// Note: The current implementation doesn't actually respect context cancellation
	// during sleep operations, so this test verifies the current behavior
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	callCount := 0
	fn := func() (string, error) {
		callCount++
		// Check if context is cancelled at the start of each call
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Always fail to test retry behavior
		return "", errors.New("test failure")
	}

	start := time.Now()
	result, err := WithRetry(ctx, "context-cancel-test", fn)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context-cancel-test failed after 3 attempts")
	assert.Equal(t, "", result)
	// The function should complete all 3 attempts even with context timeout
	// because the current implementation doesn't respect context cancellation during sleep
	assert.True(t, duration >= 6*time.Second, "Should complete all retry attempts")
}

func TestWithRetryDifferentTypes(t *testing.T) {
	t.Run("integer return type", func(t *testing.T) {
		ctx := context.Background()

		fn := func() (int, error) {
			return 42, nil
		}

		result, err := WithRetry(ctx, "integer-test", fn)

		assert.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("struct return type", func(t *testing.T) {
		ctx := context.Background()

		type TestStruct struct {
			Name  string
			Value int
		}

		expected := TestStruct{Name: "test", Value: 123}
		fn := func() (TestStruct, error) {
			return expected, nil
		}

		result, err := WithRetry(ctx, "struct-test", fn)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("pointer return type", func(t *testing.T) {
		ctx := context.Background()

		expected := &struct{ Value string }{Value: "test"}
		fn := func() (*struct{ Value string }, error) {
			return expected, nil
		}

		result, err := WithRetry(ctx, "pointer-test", fn)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})
}

func TestWithRetryErrorPropagation(t *testing.T) {
	ctx := context.Background()

	originalErr := errors.New("original error")
	fn := func() (string, error) {
		return "", originalErr
	}

	result, err := WithRetry(ctx, "error-propagation", fn)

	assert.Error(t, err)
	assert.Equal(t, "", result)
	assert.Contains(t, err.Error(), "error-propagation failed after 3 attempts")
	assert.True(t, errors.Is(err, originalErr), "Should wrap the original error")
}

func TestWithRetryOperationName(t *testing.T) {
	ctx := context.Background()

	operationName := "custom-operation-name"
	fn := func() (string, error) {
		return "", errors.New("test error")
	}

	_, err := WithRetry(ctx, operationName, fn)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), operationName)
}
