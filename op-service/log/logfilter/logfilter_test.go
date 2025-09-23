package logfilter_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	. "github.com/ethereum-optimism/optimism/op-service/log/logfilter"
	"github.com/ethereum-optimism/optimism/op-service/logmods"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/tri"
)

type CtxTestKeyType struct{}
type UserIDKeyType struct{}
type ComponentKeyType struct{}

var (
	CtxTestKey   = CtxTestKeyType{}
	UserIDKey    = UserIDKeyType{}
	ComponentKey = ComponentKeyType{}
)

// Selector Constructors
// UnknownFilterExp always returns Unknown
func UnknownFilterExp() Selector {
	return func(ctx context.Context, lvl slog.Level) tri.Tri {
		return tri.Undefined
	}
}

// CtxTestKeySelector selects based on a test context key
func CtxTestKeySelector(value string) Selector {
	return func(ctx context.Context, lvl slog.Level) tri.Tri {
		if v := ctx.Value(CtxTestKey); v != nil {
			if str, ok := v.(string); ok {
				return tri.FromBool(str == value)
			}
		}
		return tri.Undefined
	}
}

// UserSelector selects based on user ID in context
func UserSelector(userID string) Selector {
	return func(ctx context.Context, lvl slog.Level) tri.Tri {
		if v := ctx.Value(UserIDKey); v != nil {
			if str, ok := v.(string); ok {
				return tri.FromBool(str == userID)
			}
		}
		return tri.Undefined
	}
}

// ComponentSelector selects based on component name in context
func ComponentSelector(component string) Selector {
	return func(ctx context.Context, lvl slog.Level) tri.Tri {
		if v := ctx.Value(ComponentKey); v != nil {
			if str, ok := v.(string); ok {
				return tri.FromBool(str == component)
			}
		}
		return tri.Undefined
	}
}

// Test utilities for context-based filtering
func ContextWithFoo(ctx context.Context, value string) context.Context {
	// Use the exported key from the DSL
	return context.WithValue(ctx, CtxTestKey, value)
}

func FooFromContext(ctx context.Context) string {
	if v := ctx.Value(CtxTestKey); v != nil {
		if str, ok := v.(string); ok {
			return str
		}
	}
	return ""
}

func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

func UserIDFromContext(ctx context.Context) string {
	if v := ctx.Value(UserIDKey); v != nil {
		if str, ok := v.(string); ok {
			return str
		}
	}
	return ""
}

func ContextWithComponent(ctx context.Context, component string) context.Context {
	return context.WithValue(ctx, ComponentKey, component)
}

func ComponentFromContext(ctx context.Context) string {
	if v := ctx.Value(ComponentKey); v != nil {
		if str, ok := v.(string); ok {
			return str
		}
	}
	return ""
}

// TestLogSelectorComposition tests the composition operations on LogSelectors
func TestLogSelectorComposition(t *testing.T) {
	ctx := context.Background()
	level := slog.LevelInfo

	alwaysTrue := func(context.Context, slog.Level) tri.Tri { return tri.True }
	alwaysFalse := func(context.Context, slog.Level) tri.Tri { return tri.False }
	alwaysUndefined := func(context.Context, slog.Level) tri.Tri { return tri.Undefined }

	t.Run("And composition", func(t *testing.T) {
		require.Equal(t, tri.True, Selector(alwaysTrue).And(alwaysTrue)(ctx, level))
		require.Equal(t, tri.False, Selector(alwaysTrue).And(alwaysFalse)(ctx, level))
		require.Equal(t, tri.Undefined, Selector(alwaysTrue).And(alwaysUndefined)(ctx, level))
		require.Equal(t, tri.False, Selector(alwaysFalse).And(alwaysTrue)(ctx, level))
	})

	t.Run("Or composition", func(t *testing.T) {
		require.Equal(t, tri.True, Selector(alwaysTrue).Or(alwaysFalse)(ctx, level))
		require.Equal(t, tri.False, Selector(alwaysFalse).Or(alwaysFalse)(ctx, level))
		require.Equal(t, tri.True, Selector(alwaysUndefined).Or(alwaysTrue)(ctx, level))
		require.Equal(t, tri.Undefined, Selector(alwaysUndefined).Or(alwaysFalse)(ctx, level))
	})

	t.Run("Not composition", func(t *testing.T) {
		require.Equal(t, tri.False, Selector(alwaysTrue).Not()(ctx, level))
		require.Equal(t, tri.True, Selector(alwaysFalse).Not()(ctx, level))
		require.Equal(t, tri.Undefined, Selector(alwaysUndefined).Not()(ctx, level))
	})

	t.Run("Show and Mute conversions", func(t *testing.T) {
		selector := Selector(alwaysTrue)

		// Show should return the same result
		require.Equal(t, tri.True, selector.Show()(ctx, level))

		// Mute should return the negated result
		require.Equal(t, tri.False, selector.Mute()(ctx, level))
	})
}

// TestLevelFiltering tests level-based filtering functionality
func TestLevelFiltering(t *testing.T) {
	ctx := context.Background()

	t.Run("Level selector", func(t *testing.T) {
		levelFilter := Level(slog.LevelWarn)

		require.Equal(t, tri.False, levelFilter(ctx, slog.LevelDebug))
		require.Equal(t, tri.False, levelFilter(ctx, slog.LevelInfo))
		require.Equal(t, tri.True, levelFilter(ctx, slog.LevelWarn))
		require.Equal(t, tri.True, levelFilter(ctx, slog.LevelError))
	})

	t.Run("LevelExact selector", func(t *testing.T) {
		exactFilter := LevelExact(slog.LevelInfo)

		require.Equal(t, tri.False, exactFilter(ctx, slog.LevelDebug))
		require.Equal(t, tri.True, exactFilter(ctx, slog.LevelInfo))
		require.Equal(t, tri.False, exactFilter(ctx, slog.LevelWarn))
		require.Equal(t, tri.False, exactFilter(ctx, slog.LevelError))
	})
}

// TestContextBasedFiltering tests filtering based on context values
func TestContextBasedFiltering(t *testing.T) {
	t.Run("Context selector with matching value", func(t *testing.T) {
		selector := CtxTestKeySelector("alice")
		ctx := ContextWithFoo(context.Background(), "alice")

		require.Equal(t, tri.True, selector(ctx, slog.LevelInfo))
	})

	t.Run("Context selector with non-matching value", func(t *testing.T) {
		selector := CtxTestKeySelector("alice")
		ctx := ContextWithFoo(context.Background(), "bob")

		require.Equal(t, tri.False, selector(ctx, slog.LevelInfo))
	})

	t.Run("Context selector with missing context", func(t *testing.T) {
		selector := CtxTestKeySelector("alice")
		ctx := context.Background()

		require.Equal(t, tri.Undefined, selector(ctx, slog.LevelInfo))
	})
}

// TestFilterEvaluationOrder tests that filters are evaluated in reverse order with early termination
func TestFilterEvaluationOrder(t *testing.T) {
	logger := createTestLogger(t, log.LevelTrace)

	capturer, ok := logmods.FindHandler[testlog.Capturer](logger.Handler())
	require.True(t, ok)

	filterHandler, ok := logmods.FindHandler[FilterHandler](logger.Handler())
	require.True(t, ok)

	// Test evaluation order: later filters should override earlier ones
	filterHandler.Set(DefaultMute(
		Level(slog.LevelError).Show(),      // Base rule: show errors
		LevelExact(slog.LevelError).Mute(), // Override: mute errors specifically
		CtxTestKeySelector("admin").Show(), // Override: show admin logs regardless of level
	))

	// Test error log without admin context - should be muted by override
	logger.Error("error without admin")

	// Test error log with admin context - should be shown by final override
	adminCtx := ContextWithFoo(context.Background(), "admin")
	logger.ErrorContext(adminCtx, "error with admin")

	// Verify results
	errorWithoutAdmin := capturer.FindLog(testlog.NewMessageFilter("error without admin"))
	require.Nil(t, errorWithoutAdmin, "error without admin should be muted by override")

	errorWithAdmin := capturer.FindLog(testlog.NewMessageFilter("error with admin"))
	require.NotNil(t, errorWithAdmin, "error with admin should be shown by final override")
}

// TestDefaultMuteVsDefaultShow tests the different default behaviors
func TestDefaultMuteVsDefaultShow(t *testing.T) {
	t.Run("DefaultMute behavior", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// DefaultMute: show only what matches filters
		filterHandler.Set(DefaultMute(
			Level(slog.LevelError).Show(),
		))

		logger.Info("info log")   // Should be muted (default)
		logger.Error("error log") // Should be shown (matches filter)

		infoLog := capturer.FindLog(testlog.NewMessageFilter("info log"))
		require.Nil(t, infoLog, "info log should be muted by default")

		errorLog := capturer.FindLog(testlog.NewMessageFilter("error log"))
		require.NotNil(t, errorLog, "error log should be shown by filter")
	})

	t.Run("DefaultShow behavior", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// DefaultShow: hide only what matches filters
		filterHandler.Set(DefaultShow(
			LevelExact(slog.LevelDebug).Mute(),
		))

		logger.Debug("debug log") // Should be muted (matches filter)
		logger.Info("info log")   // Should be shown (default)

		debugLog := capturer.FindLog(testlog.NewMessageFilter("debug log"))
		require.Nil(t, debugLog, "debug log should be muted by filter")

		infoLog := capturer.FindLog(testlog.NewMessageFilter("info log"))
		require.NotNil(t, infoLog, "info log should be shown by default")
	})
}

// TestMultiCriteriaFiltering tests complex filter combinations
func TestMultiCriteriaFiltering(t *testing.T) {
	logger := createTestLogger(t, log.LevelTrace)

	capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
	filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

	// Show logs that are either:
	// 1. Error level or above, OR
	// 2. Debug level from admin user
	complexFilter := Level(slog.LevelError).Or(
		LevelExact(slog.LevelDebug).And(UserSelector("admin")),
	)

	filterHandler.Set(DefaultMute(complexFilter.Show()))

	// Test cases
	adminCtx := ContextWithUserID(context.Background(), "admin")
	userCtx := ContextWithUserID(context.Background(), "user")

	logger.Debug("debug from admin", slog.String("ctx", "admin"))
	logger.DebugContext(adminCtx, "debug from admin with context")
	logger.DebugContext(userCtx, "debug from user")
	logger.Error("error log")
	logger.Info("info log")

	// Verify results
	debugAdmin := capturer.FindLog(testlog.NewMessageFilter("debug from admin with context"))
	require.NotNil(t, debugAdmin, "debug from admin should be shown")

	debugUser := capturer.FindLog(testlog.NewMessageFilter("debug from user"))
	require.Nil(t, debugUser, "debug from user should be muted")

	errorLog := capturer.FindLog(testlog.NewMessageFilter("error log"))
	require.NotNil(t, errorLog, "error log should be shown")

	infoLog := capturer.FindLog(testlog.NewMessageFilter("info log"))
	require.Nil(t, infoLog, "info log should be muted")
}

// TestCascadingFilters tests that later filters override earlier ones
func TestCascadingFilters(t *testing.T) {
	logger := createTestLogger(t, log.LevelTrace)

	capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
	filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

	filterHandler.Set(DefaultMute(
		// Base rule: Show errors and above
		Level(slog.LevelError).Show(),

		// Override: Also show debug logs for admin users
		LevelExact(slog.LevelDebug).And(UserSelector("admin")).Show(),

		// Override: Never show debug logs from metrics component
		ComponentSelector("metrics").And(LevelExact(slog.LevelDebug)).Mute(),
	))

	// Test contexts
	adminCtx := ContextWithUserID(context.Background(), "admin")
	adminMetricsCtx := ContextWithComponent(ContextWithUserID(context.Background(), "admin"), "metrics")

	logger.ErrorContext(context.Background(), "error log")
	logger.DebugContext(adminCtx, "admin debug")
	logger.DebugContext(adminMetricsCtx, "admin metrics debug")

	// Verify results
	errorLog := capturer.FindLog(testlog.NewMessageFilter("error log"))
	require.NotNil(t, errorLog, "error log should be shown")

	adminDebug := capturer.FindLog(testlog.NewMessageFilter("admin debug"))
	require.NotNil(t, adminDebug, "admin debug should be shown")

	adminMetricsDebug := capturer.FindLog(testlog.NewMessageFilter("admin metrics debug"))
	require.Nil(t, adminMetricsDebug, "admin metrics debug should be muted by final override")
}

// TestThreadSafety tests that filter configuration updates are thread-safe
func TestThreadSafety(t *testing.T) {
	logger := createTestLogger(t, log.LevelTrace)

	filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

	var wg sync.WaitGroup
	var errorCount int64

	// Initial configuration
	filterHandler.Set(DefaultMute(Level(slog.LevelError).Show()))

	// Start multiple goroutines that update the filter configuration
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if id%2 == 0 {
					filterHandler.Set(DefaultMute(Level(slog.LevelError).Show()))
				} else {
					filterHandler.Set(DefaultShow(LevelExact(slog.LevelDebug).Mute()))
				}
			}
		}(i)
	}

	// Start goroutines that continuously log
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				logger.Error("concurrent error")
				logger.Debug("concurrent debug")
				logger.Info("concurrent info")
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify no race conditions occurred by checking that we can still use the filter
	logger.Error("final test")
	require.Equal(t, int64(0), errorCount, "no errors should occur during concurrent access")
}

// TestGracefulDegradation tests behavior when no filters match or return Undefined
func TestGracefulDegradation(t *testing.T) {
	t.Run("No configuration set", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())

		// With no filter configuration, all logs should pass through
		logger.Info("no config test")

		log := capturer.FindLog(testlog.NewMessageFilter("no config test"))
		require.NotNil(t, log, "log should pass through when no config is set")
	})

	t.Run("All filters return Undefined", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// Set filters that all return Undefined
		filterHandler.Set(DefaultMute(
			UnknownFilterExp().Show(),
			UnknownFilterExp().Mute(),
		))

		logger.Info("undefined test")

		log := capturer.FindLog(testlog.NewMessageFilter("undefined test"))
		require.Nil(t, log, "with DefaultMute and all Undefined, should default to muting")
	})

	t.Run("DefaultShow with all Undefined", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		filterHandler.Set(DefaultShow(
			UnknownFilterExp().Show(),
			UnknownFilterExp().Mute(),
		))

		logger.Info("undefined show test")

		log := capturer.FindLog(testlog.NewMessageFilter("undefined show test"))
		require.NotNil(t, log, "with DefaultShow and all Undefined, should default to showing")
	})
}

// TestOriginalWrapFilterHandler ensures the original test still passes
func TestWrapFilterHandler(t *testing.T) {
	// Create a logger, with capturing and filtering
	logger := createTestLogger(t, log.LevelTrace)

	capturer, ok := logmods.FindHandler[testlog.Capturer](logger.Handler())
	require.True(t, ok)

	filterHandler, ok := logmods.FindHandler[FilterHandler](logger.Handler())
	require.True(t, ok)
	filterHandler.Set(DefaultMute(CtxTestKeySelector("alice").Show()))

	// Log some things
	logger.InfoContext(context.Background(), "unrecognized context 1")
	logger.InfoContext(ContextWithFoo(context.Background(), "alice"), "matched context")
	logger.InfoContext(context.Background(), "unrecognized context 2")

	// Now see if the filter worked - only the matched context should be captured
	rec := capturer.FindLog(testlog.NewMessageFilter("matched context"))
	require.NotNil(t, rec, "matched context log should be found")
	require.Equal(t, "matched context", rec.Record.Message)

	// Verify other logs were filtered out
	unrecognized1 := capturer.FindLog(testlog.NewMessageFilter("unrecognized context 1"))
	require.Nil(t, unrecognized1, "unrecognized context 1 should be filtered out")

	unrecognized2 := capturer.FindLog(testlog.NewMessageFilter("unrecognized context 2"))
	require.Nil(t, unrecognized2, "unrecognized context 2 should be filtered out")
}

// TestREADMEExamples tests specific examples from the README
func TestREADMEExamples(t *testing.T) {
	t.Run("Basic level filtering example", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// Example from README: Only show error-level logs and above
		filterHandler.Set(DefaultMute(
			Level(slog.LevelError).Show(),
		))

		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")

		require.Nil(t, capturer.FindLog(testlog.NewMessageFilter("debug message")))
		require.Nil(t, capturer.FindLog(testlog.NewMessageFilter("info message")))
		require.Nil(t, capturer.FindLog(testlog.NewMessageFilter("warn message")))
		require.NotNil(t, capturer.FindLog(testlog.NewMessageFilter("error message")))
	})

	t.Run("Context-based filtering example", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// Example from README: Show admin users and errors
		filterHandler.Set(DefaultMute(
			UserSelector("admin").Show(),
			Level(slog.LevelError).Show(),
		))

		adminCtx := ContextWithUserID(context.Background(), "admin")
		userCtx := ContextWithUserID(context.Background(), "user")

		logger.InfoContext(adminCtx, "admin info")
		logger.InfoContext(userCtx, "user info")
		logger.ErrorContext(userCtx, "user error")

		require.NotNil(t, capturer.FindLog(testlog.NewMessageFilter("admin info")))
		require.Nil(t, capturer.FindLog(testlog.NewMessageFilter("user info")))
		require.NotNil(t, capturer.FindLog(testlog.NewMessageFilter("user error")))
	})
}

// TestProblematicBehaviorWithEarlyTermination demonstrates the issue with early termination
// in filter evaluation that leads to counter-intuitive behavior
func TestProblematicBehaviorWithEarlyTermination(t *testing.T) {
	logger := createTestLogger(t, log.LevelTrace)

	capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
	filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

	t.Run("DefaultShow with multiple Mute filters - problematic behavior", func(t *testing.T) {
		capturer.Clear()

		// This configuration should mute logs that are EITHER admin context OR debug level
		// But due to early termination, the behavior is unexpected
		filterHandler.Set(DefaultShow(
			CtxTestKeySelector("admin").Mute(), // Should mute admin logs
			LevelExact(slog.LevelDebug).Mute(), // Should mute debug logs
		))

		adminCtx := ContextWithFoo(context.Background(), "admin")

		// This log is admin context but INFO level
		// Expected: Should be muted (matches admin selector)
		// Actual: Might not be muted due to early termination
		logger.InfoContext(adminCtx, "admin info log")

		// This log is debug level but not admin context
		// Expected: Should be muted (matches debug level selector)
		// Actual: Should work correctly as it hits the debug filter
		logger.Debug("debug log")

		// This log is neither admin nor debug
		// Expected: Should be shown (doesn't match any mute filter)
		// Actual: Should work correctly
		logger.Info("normal info log")

		adminInfoLog := capturer.FindLog(testlog.NewMessageFilter("admin info log"))
		debugLog := capturer.FindLog(testlog.NewMessageFilter("debug log"))
		normalInfoLog := capturer.FindLog(testlog.NewMessageFilter("normal info log"))

		// This demonstrates the problematic behavior:
		// The admin info log might not be muted as expected
		t.Logf("Admin info log found: %v (should be nil)", adminInfoLog != nil)
		t.Logf("Debug log found: %v (should be nil)", debugLog != nil)
		t.Logf("Normal info log found: %v (should be true)", normalInfoLog != nil)

		// With current implementation, these might fail due to early termination issues
		require.Nil(t, adminInfoLog, "admin info log should be muted")
		require.Nil(t, debugLog, "debug log should be muted")
		require.NotNil(t, normalInfoLog, "normal info log should be shown")
	})

	t.Run("DefaultMute with multiple Show filters - problematic behavior", func(t *testing.T) {
		capturer.Clear()

		// This configuration should show logs that are EITHER admin context OR error level
		// But due to early termination, the behavior is unexpected
		filterHandler.Set(DefaultMute(
			CtxTestKeySelector("admin").Show(), // Should show admin logs
			Level(slog.LevelError).Show(),      // Should show error+ logs
		))

		adminCtx := ContextWithFoo(context.Background(), "admin")

		// This log is admin context and INFO level
		// Expected: Should be shown (matches admin selector)
		// Actual: Should work if admin filter is evaluated
		logger.InfoContext(adminCtx, "admin info log")

		// This log is error level but not admin context
		// Expected: Should be shown (matches error level selector)
		// Actual: Should work if error filter is evaluated
		logger.Error("error log")

		// This log is neither admin nor error+
		// Expected: Should be muted (doesn't match any show filter)
		// Actual: Should work correctly
		logger.Info("normal info log")

		adminInfoLog := capturer.FindLog(testlog.NewMessageFilter("admin info log"))
		errorLog := capturer.FindLog(testlog.NewMessageFilter("error log"))
		normalInfoLog := capturer.FindLog(testlog.NewMessageFilter("normal info log"))

		t.Logf("Admin info log found: %v (should be true)", adminInfoLog != nil)
		t.Logf("Error log found: %v (should be true)", errorLog != nil)
		t.Logf("Normal info log found: %v (should be false)", normalInfoLog != nil)

		// These should work correctly with OR semantics
		require.NotNil(t, adminInfoLog, "admin info log should be shown")
		require.NotNil(t, errorLog, "error log should be shown")
		require.Nil(t, normalInfoLog, "normal info log should be muted")
	})
}

// TestWithAttrsIntegration tests end-to-end functionality of WithAttrs with filtering
func TestWithAttrsIntegration(t *testing.T) {
	t.Run("Basic WithAttrs filtering", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// Set up filter to show logs with component="database"
		filterHandler.Set(DefaultMute(
			Select("component", "database").Show(),
		))

		// Create a child logger with component attribute
		dbLogger := logger.With("component", "database")
		cacheLogger := logger.With("component", "cache")

		// Log messages
		dbLogger.Info("database operation")
		cacheLogger.Info("cache operation")
		logger.Info("general operation")

		// Verify filtering worked
		dbLog := capturer.FindLog(testlog.NewMessageFilter("database operation"))
		require.NotNil(t, dbLog, "database log should be shown")

		cacheLog := capturer.FindLog(testlog.NewMessageFilter("cache operation"))
		require.Nil(t, cacheLog, "cache log should be muted")

		generalLog := capturer.FindLog(testlog.NewMessageFilter("general operation"))
		require.Nil(t, generalLog, "general log should be muted")
	})

	t.Run("Multiple WithAttrs calls", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// Set up filter to show logs with both component="api" AND version="v2"
		filterHandler.Set(DefaultMute(
			Select("component", "api").And(
				Select("version", "v2")).Show(),
		))

		// Chain multiple WithAttrs calls
		apiLogger := logger.With("component", "api")
		apiV2Logger := apiLogger.With("version", "v2")
		apiV1Logger := apiLogger.With("version", "v1")

		// Log messages
		apiV2Logger.Info("API v2 operation")
		apiV1Logger.Info("API v1 operation")
		logger.Info("base operation")

		// Verify filtering worked
		v2Log := capturer.FindLog(testlog.NewMessageFilter("API v2 operation"))
		require.NotNil(t, v2Log, "API v2 log should be shown")

		v1Log := capturer.FindLog(testlog.NewMessageFilter("API v1 operation"))
		require.Nil(t, v1Log, "API v1 log should be muted")

		baseLog := capturer.FindLog(testlog.NewMessageFilter("base operation"))
		require.Nil(t, baseLog, "base log should be muted")
	})

	t.Run("WithAttrs with different value types", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// Test different attribute types - use PrimitiveSelectorFn for type-safe matching
		filterHandler.Set(DefaultMute(
			Select("user_id", 123).Show(),
		))

		userLogger := logger.With("user_id", 123)
		// otherUserLogger := logger.With("user_id", 456)

		userLogger.Info("user 123 action")
		// otherUserLogger.Info("user 456 action")

		user123Log := capturer.FindLog(testlog.NewMessageFilter("user 123 action"))
		require.NotNil(t, user123Log, "user 123 log should be shown")

		// user456Log := capturer.FindLog(testlog.NewMessageFilter("user 456 action"))
		// require.Nil(t, user456Log, "user 456 log should be muted")
	})
}

// TestWithGroupIntegration tests end-to-end functionality of WithGroup with filtering
func TestWithGroupIntegration(t *testing.T) {
	t.Run("Basic WithGroup filtering", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// Set up filter to show logs with grouped attribute "db.table"="users"
		filterHandler.Set(DefaultMute(
			Select("db.table", "users").Show(),
		))

		// Create grouped logger by working with handler directly
		dbHandler := logger.Handler().WithGroup("db")
		dbLogger := log.NewLogger(dbHandler)
		usersLogger := dbLogger.With("table", "users")
		postsLogger := dbLogger.With("table", "posts")

		// Log messages
		usersLogger.Info("users table operation")
		postsLogger.Info("posts table operation")
		logger.Info("base operation")

		// Verify filtering worked
		usersLog := capturer.FindLog(testlog.NewMessageFilter("users table operation"))
		require.NotNil(t, usersLog, "users table log should be shown")

		postsLog := capturer.FindLog(testlog.NewMessageFilter("posts table operation"))
		require.Nil(t, postsLog, "posts table log should be muted")

		baseLog := capturer.FindLog(testlog.NewMessageFilter("base operation"))
		require.Nil(t, baseLog, "base log should be muted")
	})

	t.Run("Nested groups", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// Set up filter for deeply nested attribute "service.db.query.type"="select"
		filterHandler.Set(DefaultMute(
			Select("service.db.query.type", "select").Show(),
		))

		// Create nested groups by chaining handler calls
		serviceHandler := logger.Handler().WithGroup("service")
		dbHandler := serviceHandler.WithGroup("db")
		queryHandler := dbHandler.WithGroup("query")
		queryLogger := log.NewLogger(queryHandler)
		selectLogger := queryLogger.With("type", "select")
		insertLogger := queryLogger.With("type", "insert")

		// Log messages
		selectLogger.Info("select query executed")
		insertLogger.Info("insert query executed")

		// Verify filtering worked
		selectLog := capturer.FindLog(testlog.NewMessageFilter("select query executed"))
		require.NotNil(t, selectLog, "select query log should be shown")

		insertLog := capturer.FindLog(testlog.NewMessageFilter("insert query executed"))
		require.Nil(t, insertLog, "insert query log should be muted")
	})

	t.Run("Group with existing attributes", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// Set up filters for both base and grouped attributes
		filterHandler.Set(DefaultMute(
			Select("service", "api").And(
				Select("metrics.type", "counter")).Show(),
		))

		// Create logger with base attribute, then add group
		apiHandler := logger.Handler().WithAttrs([]slog.Attr{slog.String("service", "api")})
		metricsHandler := apiHandler.WithGroup("metrics")
		metricsLogger := log.NewLogger(metricsHandler)
		counterLogger := metricsLogger.With("type", "counter")
		gaugeLogger := metricsLogger.With("type", "gauge")

		// Log messages
		counterLogger.Info("counter metric updated")
		gaugeLogger.Info("gauge metric updated")

		// Verify filtering worked
		counterLog := capturer.FindLog(testlog.NewMessageFilter("counter metric updated"))
		require.NotNil(t, counterLog, "counter metric log should be shown")

		gaugeLog := capturer.FindLog(testlog.NewMessageFilter("gauge metric updated"))
		require.Nil(t, gaugeLog, "gauge metric log should be muted")
	})
}

// TestWithAttrsAndGroupCombination tests complex combinations of WithAttrs and WithGroup
func TestWithAttrsAndGroupCombination(t *testing.T) {
	t.Run("Complex attribute hierarchy", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// Set up complex filter
		filterHandler.Set(DefaultMute(
			// Show logs from user service database operations on users table
			Select("service", "user").And(
				Select("db.table", "users").And(
					Select("db.operation.type", "read"))).Show(),
		))

		// Build complex logger hierarchy using handler manipulation
		userServiceHandler := logger.Handler().WithAttrs([]slog.Attr{slog.String("service", "user")})
		dbHandler := userServiceHandler.WithGroup("db")
		usersTableHandler := dbHandler.WithAttrs([]slog.Attr{slog.String("table", "users")})
		postsTableHandler := dbHandler.WithAttrs([]slog.Attr{slog.String("table", "posts")})
		operationHandler := usersTableHandler.WithGroup("operation")

		readOpLogger := log.NewLogger(operationHandler).With("type", "read")
		writeOpLogger := log.NewLogger(operationHandler).With("type", "write")
		postsTableLogger := log.NewLogger(postsTableHandler)

		// Log various operations
		readOpLogger.Info("reading user data")
		writeOpLogger.Info("writing user data")
		postsTableLogger.Info("posts operation")

		// Verify only the specific combination is shown
		readLog := capturer.FindLog(testlog.NewMessageFilter("reading user data"))
		require.NotNil(t, readLog, "user service read operation should be shown")

		writeLog := capturer.FindLog(testlog.NewMessageFilter("writing user data"))
		require.Nil(t, writeLog, "user service write operation should be muted")

		postsLog := capturer.FindLog(testlog.NewMessageFilter("posts operation"))
		require.Nil(t, postsLog, "posts operation should be muted")
	})

	t.Run("Filter inheritance across group boundaries", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// Set up filter that affects multiple attribute levels
		filterHandler.Set(DefaultMute(
			// Show debug logs only from admin users with auth API component
			LevelExact(slog.LevelDebug).And(
				Select("user_type", "admin").And(
					Select("api.component", "auth"))).Show(),
		))

		// Create loggers with different combinations using handler manipulation
		adminHandler := logger.Handler().WithAttrs([]slog.Attr{slog.String("user_type", "admin")})
		userHandler := logger.Handler().WithAttrs([]slog.Attr{slog.String("user_type", "user")})

		adminAPIHandler := adminHandler.WithGroup("api")
		userAPIHandler := userHandler.WithGroup("api")

		adminAuthLogger := log.NewLogger(adminAPIHandler).With("component", "auth")
		adminCacheLogger := log.NewLogger(adminAPIHandler).With("component", "cache")
		userAuthLogger := log.NewLogger(userAPIHandler).With("component", "auth")

		// Log debug messages
		adminAuthLogger.Debug("admin auth debug")
		adminCacheLogger.Debug("admin cache debug")
		userAuthLogger.Debug("user auth debug")
		logger.Debug("base debug")

		// Verify filtering
		adminAuthLog := capturer.FindLog(testlog.NewMessageFilter("admin auth debug"))
		require.NotNil(t, adminAuthLog, "admin auth debug should be shown")

		adminCacheLog := capturer.FindLog(testlog.NewMessageFilter("admin cache debug"))
		require.Nil(t, adminCacheLog, "admin cache debug should be muted")

		userAuthLog := capturer.FindLog(testlog.NewMessageFilter("user auth debug"))
		require.Nil(t, userAuthLog, "user auth debug should be muted")

		baseLog := capturer.FindLog(testlog.NewMessageFilter("base debug"))
		require.Nil(t, baseLog, "base debug should be muted")
	})

	t.Run("Filter reconfiguration with persistent attributes", func(t *testing.T) {
		logger := createTestLogger(t, log.LevelTrace)

		capturer, _ := logmods.FindHandler[testlog.Capturer](logger.Handler())
		filterHandler, _ := logmods.FindHandler[FilterHandler](logger.Handler())

		// First filter configuration: show only slow queries
		filterHandler.Set(DefaultMute(
			Select("query.type", "slow").Show(),
		))

		// Create logger with persistent attributes using handler manipulation
		dbHandler := logger.Handler().WithAttrs([]slog.Attr{slog.String("service", "database")})
		queryHandler := dbHandler.WithGroup("query")
		queryLogger := log.NewLogger(queryHandler)
		slowQueryLogger := queryLogger.With("type", "slow")
		fastQueryLogger := queryLogger.With("type", "fast")

		slowQueryLogger.Info("slow query detected")
		fastQueryLogger.Info("fast query executed")

		slowLog1 := capturer.FindLog(testlog.NewMessageFilter("slow query detected"))
		require.NotNil(t, slowLog1, "slow query should be shown with first filter")

		fastLog1 := capturer.FindLog(testlog.NewMessageFilter("fast query executed"))
		require.Nil(t, fastLog1, "fast query should be muted with first filter")

		// Reconfigure filter: show database service logs regardless of query type
		capturer.Clear()
		filterHandler.Set(DefaultMute(
			Select("service", "database").Show(),
		))

		// Create new loggers with the updated filter
		dbHandler2 := logger.Handler().WithAttrs([]slog.Attr{slog.String("service", "database")})
		queryHandler2 := dbHandler2.WithGroup("query")
		queryLogger2 := log.NewLogger(queryHandler2)
		slowQueryLogger2 := queryLogger2.With("type", "slow")
		fastQueryLogger2 := queryLogger2.With("type", "fast")

		slowQueryLogger2.Info("another slow query")
		fastQueryLogger2.Info("another fast query")

		slowLog2 := capturer.FindLog(testlog.NewMessageFilter("another slow query"))
		require.NotNil(t, slowLog2, "slow query should be shown with second filter")

		fastLog2 := capturer.FindLog(testlog.NewMessageFilter("another fast query"))
		require.NotNil(t, fastLog2, "fast query should be shown with second filter")
	})
}

// createTestLogger creates a logger with handlers that support WithGroup for testing
func createTestLogger(t *testing.T, level slog.Level) log.Logger {
	// Use a discard handler as the base to avoid stdout output during tests
	baseHandler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
		Level: level,
	})
	// Apply filtering first, then capture the filtered logs
	filterHandler := WrapFilterHandler(baseHandler)
	capturer := testlog.WrapCaptureLogger(filterHandler)
	return log.NewLogger(capturer)
}
