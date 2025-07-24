// Kleene’s three-valued logic (K3) enlarges classical logic with an “undefined”
// value ⊥ alongside true (T) and false (F) to model partial or indeterminate
// statements. It keeps the classical truth tables where both operands are T or
// F, but whenever any input is ⊥, the result is ⊥ except where truth can still
// be decided. Only T is “designated” (counted as logically valid), so the law
// of excluded middle (p ∨ ¬p) may be ⊥ rather than T, reflecting uncertainty,
// while contradictions (p ∧ ¬p) can also be ⊥ instead of always F. This makes
// K3 paracomplete – avoiding commitment to truth or falsity when information is
// incomplete—yet it reduces to classical logic once all statements are decided.
//
// This logical system is useful for building a general DSL for selecting logs
// and handling them according to a factorization of concerns.
package tri

type Tri int8

const (
	False     Tri = iota - 1 // −1 keeps the natural ordering: False < Unknown < True
	Undefined                // 0
	True                     // 1
)

// String lets fmt / log print nice names.
func (t Tri) String() string {
	switch t {
	case True:
		return "true"
	case False:
		return "false"
	default:
		return "undefined"
	}
}

// Valid reports whether the value is True or False (i.e. not Unknown).
func (t Tri) Valid() bool { return t != Undefined }

// Bool returns (value, valid) so you can drop Unknown easily.
func (t Tri) Bool(failOpen bool) bool {
	switch t {
	case True:
		return true
	case False:
		return false
	default:
		return failOpen
	}
}

func (t Tri) And(b Tri) Tri {
	if t < b {
		return t
	}
	return b
}

func (t Tri) Or(b Tri) Tri {
	if t > b {
		return t
	}
	return b
}

func (t Tri) Not() Tri {
	return -t
}

func FromBool(b bool) Tri {
	if b {
		return True
	}
	return False
}
