package testutil

type TestCaseVariation[C, V any] struct {
	Base      C
	Variation V
}

func TestVariations[C, V any](cases []C, variations []V) []TestCaseVariation[C, V] {
	out := make([]TestCaseVariation[C, V], 0, len(cases)*len(variations))
	for _, baseTestCase := range cases {
		for _, variation := range variations {
			out = append(out, TestCaseVariation[C, V]{Base: baseTestCase, Variation: variation})
		}
	}
	return out
}
