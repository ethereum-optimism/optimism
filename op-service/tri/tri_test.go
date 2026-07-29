package tri

import (
	"testing"
)

func TestTriConstants(t *testing.T) {
	// Test the ordering: False < Undefined < True
	if False >= Undefined {
		t.Errorf("Expected False < Undefined, but False = %d, Undefined = %d", False, Undefined)
	}
	if Undefined >= True {
		t.Errorf("Expected Undefined < True, but Undefined = %d, True = %d", Undefined, True)
	}
	if False >= True {
		t.Errorf("Expected False < True, but False = %d, True = %d", False, True)
	}
}

func TestTriString(t *testing.T) {
	tests := []struct {
		name     string
		tri      Tri
		expected string
	}{
		{"True", True, "true"},
		{"False", False, "false"},
		{"Undefined", Undefined, "undefined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tri.String(); got != tt.expected {
				t.Errorf("Tri.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTriValid(t *testing.T) {
	tests := []struct {
		name     string
		tri      Tri
		expected bool
	}{
		{"True is valid", True, true},
		{"False is valid", False, true},
		{"Undefined is not valid", Undefined, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tri.Valid(); got != tt.expected {
				t.Errorf("Tri.Valid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTriBool(t *testing.T) {
	tests := []struct {
		name     string
		tri      Tri
		failOpen bool
		expected bool
	}{
		{"True with failOpen=false", True, false, true},
		{"True with failOpen=true", True, true, true},
		{"False with failOpen=false", False, false, false},
		{"False with failOpen=true", False, true, false},
		{"Undefined with failOpen=false", Undefined, false, false},
		{"Undefined with failOpen=true", Undefined, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tri.Bool(tt.failOpen); got != tt.expected {
				t.Errorf("Tri.Bool(%v) = %v, want %v", tt.failOpen, got, tt.expected)
			}
		})
	}
}

func TestTriAnd(t *testing.T) {
	// Test all combinations of And operation
	tests := []struct {
		name     string
		a, b     Tri
		expected Tri
	}{
		// Classical logic cases
		{"True AND True", True, True, True},
		{"True AND False", True, False, False},
		{"False AND True", False, True, False},
		{"False AND False", False, False, False},

		// Three-valued logic cases with Undefined
		{"True AND Undefined", True, Undefined, Undefined},
		{"Undefined AND True", Undefined, True, Undefined},
		{"False AND Undefined", False, Undefined, False},
		{"Undefined AND False", Undefined, False, False},
		{"Undefined AND Undefined", Undefined, Undefined, Undefined},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.And(tt.b); got != tt.expected {
				t.Errorf("(%v).And(%v) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestTriOr(t *testing.T) {
	// Test all combinations of Or operation
	tests := []struct {
		name     string
		a, b     Tri
		expected Tri
	}{
		// Classical logic cases
		{"True OR True", True, True, True},
		{"True OR False", True, False, True},
		{"False OR True", False, True, True},
		{"False OR False", False, False, False},

		// Three-valued logic cases with Undefined
		{"True OR Undefined", True, Undefined, True},
		{"Undefined OR True", Undefined, True, True},
		{"False OR Undefined", False, Undefined, Undefined},
		{"Undefined OR False", Undefined, False, Undefined},
		{"Undefined OR Undefined", Undefined, Undefined, Undefined},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Or(tt.b); got != tt.expected {
				t.Errorf("(%v).Or(%v) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestTriNot(t *testing.T) {
	tests := []struct {
		name     string
		tri      Tri
		expected Tri
	}{
		{"NOT True", True, False},
		{"NOT False", False, True},
		{"NOT Undefined", Undefined, Undefined},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tri.Not(); got != tt.expected {
				t.Errorf("(%v).Not() = %v, want %v", tt.tri, got, tt.expected)
			}
		})
	}
}

func TestFromBool(t *testing.T) {
	tests := []struct {
		name     string
		b        bool
		expected Tri
	}{
		{"true to True", true, True},
		{"false to False", false, False},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromBool(tt.b); got != tt.expected {
				t.Errorf("FromBool(%v) = %v, want %v", tt.b, got, tt.expected)
			}
		})
	}
}

// Test De Morgan's laws and other logical properties
func TestTriLogicalProperties(t *testing.T) {
	values := []Tri{True, False, Undefined}

	t.Run("Double negation", func(t *testing.T) {
		for _, v := range values {
			if got := v.Not().Not(); got != v {
				t.Errorf("Double negation failed: (%v).Not().Not() = %v, want %v", v, got, v)
			}
		}
	})

	t.Run("Idempotency of And", func(t *testing.T) {
		for _, v := range values {
			if got := v.And(v); got != v {
				t.Errorf("And idempotency failed: (%v).And(%v) = %v, want %v", v, v, got, v)
			}
		}
	})

	t.Run("Idempotency of Or", func(t *testing.T) {
		for _, v := range values {
			if got := v.Or(v); got != v {
				t.Errorf("Or idempotency failed: (%v).Or(%v) = %v, want %v", v, v, got, v)
			}
		}
	})

	t.Run("Commutativity of And", func(t *testing.T) {
		for _, a := range values {
			for _, b := range values {
				if got1, got2 := a.And(b), b.And(a); got1 != got2 {
					t.Errorf("And commutativity failed: (%v).And(%v) = %v, but (%v).And(%v) = %v", a, b, got1, b, a, got2)
				}
			}
		}
	})

	t.Run("Commutativity of Or", func(t *testing.T) {
		for _, a := range values {
			for _, b := range values {
				if got1, got2 := a.Or(b), b.Or(a); got1 != got2 {
					t.Errorf("Or commutativity failed: (%v).Or(%v) = %v, but (%v).Or(%v) = %v", a, b, got1, b, a, got2)
				}
			}
		}
	})

	t.Run("Identity elements", func(t *testing.T) {
		for _, v := range values {
			// True is identity for And
			if got := v.And(True); got != v {
				t.Errorf("And identity failed: (%v).And(True) = %v, want %v", v, got, v)
			}
			if got := True.And(v); got != v {
				t.Errorf("And identity failed: True.And(%v) = %v, want %v", v, got, v)
			}

			// False is identity for Or
			if got := v.Or(False); got != v {
				t.Errorf("Or identity failed: (%v).Or(False) = %v, want %v", v, got, v)
			}
			if got := False.Or(v); got != v {
				t.Errorf("Or identity failed: False.Or(%v) = %v, want %v", v, got, v)
			}
		}
	})

	t.Run("Absorption elements", func(t *testing.T) {
		for _, v := range values {
			// False absorbs in And
			if got := v.And(False); got != False {
				t.Errorf("And absorption failed: (%v).And(False) = %v, want False", v, got)
			}
			if got := False.And(v); got != False {
				t.Errorf("And absorption failed: False.And(%v) = %v, want False", v, got)
			}

			// True absorbs in Or
			if got := v.Or(True); got != True {
				t.Errorf("Or absorption failed: (%v).Or(True) = %v, want True", v, got)
			}
			if got := True.Or(v); got != True {
				t.Errorf("Or absorption failed: True.Or(%v) = %v, want True", v, got)
			}
		}
	})
}
