package minimal

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-test/test"
)

func TestMain(m *testing.M) {
	test.Main(m)
}

func TestMinimal(t *testing.T) {
	test.Plan(t, func(t test.Planner) {
		t.Select("example-param", []string{"foo", "bar"}, func(t test.Planner) {
			t.Plan("sub-plan", func(t test.Planner) {
				t.Run("run-a", func(t test.Executor) {
					v, ok := t.Parameter("example-param")
					t.Log("running A!", v, ok)
				})
				t.Run("run-b", func(t test.Executor) {
					v, ok := t.Parameter("example-param")
					t.Log("running B!", v, ok)
				})
			})
		})
	})
}
