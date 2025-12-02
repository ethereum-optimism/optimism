package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMinCheckedTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		now      time.Time
		duration time.Duration
		want     uint64
	}{
		{
			name:     "ZeroDurationZeroClock",
			now:      time.Unix(0, 0),
			duration: 0,
			want:     0,
		},
		{
			name:     "ZeroDurationPositiveClock",
			now:      time.Unix(1, 0),
			duration: 0,
			want:     0,
		},
		{
			name:     "UnderflowZeroClock",
			now:      time.Unix(0, 0),
			duration: time.Second,
			want:     0,
		},
		{
			name:     "UnderflowPositiveClock",
			now:      time.Unix(1, 0),
			duration: time.Second * 2,
			want:     0,
		},
		{
			name:     "CorrectArithmetic",
			now:      time.Unix(100, 0),
			duration: time.Second * 10,
			want:     90,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			clock := &mockClock{now: test.now}
			require.Equal(t, test.want, MinCheckedTimestamp(clock, test.duration))
		})
	}
}

type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}
