package devtest

import "testing"

func TestMarkFlakySkipsRequireFailure(t *testing.T) {
	t.Setenv(FailFlakyTests, "false")
	ok := t.Run("flaky", func(gt *testing.T) {
		dt := SerialT(gt)
		dt.MarkFlaky("tracked as flaky")
		dt.Require().Equal("expected", "actual")
		gt.Fatal("flaky assertion should skip before reaching this line")
	})
	if !ok {
		t.Fatal("expected flaky subtest to be skipped instead of failed")
	}
}

func TestMarkFlakySkipsExplicitFailNow(t *testing.T) {
	t.Setenv(FailFlakyTests, "false")
	ok := t.Run("flaky", func(gt *testing.T) {
		dt := SerialT(gt)
		dt.MarkFlaky("tracked as flaky")
		dt.FailNow()
		gt.Fatal("flaky FailNow should skip before reaching this line")
	})
	if !ok {
		t.Fatal("expected flaky subtest to be skipped instead of failed")
	}
}

func TestMarkFlakyPropagatesToSubtests(t *testing.T) {
	t.Setenv(FailFlakyTests, "false")
	ok := t.Run("flaky-parent", func(gt *testing.T) {
		dt := SerialT(gt)
		dt.MarkFlaky("tracked as flaky")
		dt.Run("child", func(dt T) {
			dt.Require().Equal(1, 2)
			gt.Fatal("flaky child assertion should skip before reaching this line")
		})
	})
	if !ok {
		t.Fatal("expected flaky child subtest to be skipped instead of failed")
	}
}

func TestMarkFlakyFailProducesAnnotatedSkip(t *testing.T) {
	t.Setenv(FailFlakyTests, "false")
	ok := t.Run("flaky", func(gt *testing.T) {
		dt := SerialT(gt)
		dt.MarkFlaky("test-reason")
		dt.Require().Equal("expected", "actual")
		gt.Fatal("should not reach here")
	})
	if !ok {
		t.Fatal("expected flaky subtest to be skipped instead of failed")
	}
}

func TestMarkFlakyPassProducesAnnotatedSkip(t *testing.T) {
	t.Setenv(FailFlakyTests, "false")
	ok := t.Run("flaky-pass", func(gt *testing.T) {
		dt := SerialT(gt)
		dt.MarkFlaky("test-reason")
		// Test passes — no assertions fail
	})
	if !ok {
		t.Fatal("expected flaky passing subtest to be skipped (not failed)")
	}
}

func TestMarkFlakyAnnotationsDisabledWhenForceFail(t *testing.T) {
	t.Setenv(FailFlakyTests, "true")
	// Verify that mustFailFlakyTests returns true, which means
	// shouldSkipFlakyFailure returns false and FailNow will not be
	// converted to a skip.  We cannot call FailNow in a subtest here
	// because Go propagates subtest failures to the parent, which would
	// cause this test to fail unconditionally.
	if !mustFailFlakyTests() {
		t.Fatal("expected mustFailFlakyTests to return true when DEVNET_FAIL_FLAKY_TESTS=true")
	}
}

func TestMarkFlakyPassNoAnnotationWhenForceFail(t *testing.T) {
	t.Setenv(FailFlakyTests, "true")
	ok := t.Run("flaky-pass-forced", func(gt *testing.T) {
		dt := SerialT(gt)
		dt.MarkFlaky("test-reason")
		// Test passes — no assertions fail
		// Should NOT be skipped when DEVNET_FAIL_FLAKY_TESTS=true
	})
	if !ok {
		t.Fatal("expected passing test to pass normally when DEVNET_FAIL_FLAKY_TESTS=true")
	}
}

func TestMarkFlakySubtestPassProducesAnnotatedSkip(t *testing.T) {
	t.Setenv(FailFlakyTests, "false")
	ok := t.Run("flaky-parent", func(gt *testing.T) {
		dt := SerialT(gt)
		dt.MarkFlaky("test-reason")
		dt.Run("passing-child", func(dt T) {
			// Child passes — should be skipped with FLAKY_PASS
		})
	})
	if !ok {
		t.Fatal("expected flaky passing subtest to be skipped (not failed)")
	}
}
