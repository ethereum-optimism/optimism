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
