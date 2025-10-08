package sync

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringToSafeSource(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    SafeSource
		wantErr bool
	}{
		{
			name:  "l1",
			input: "l1",
			want:  SafeSourceL1,
		},
		{
			name:  "l2",
			input: "l2",
			want:  SafeSourceL2,
		},
		{
			name:  "L1 uppercase",
			input: "L1",
			want:  SafeSourceL1,
		},
		{
			name:  "L2 uppercase",
			input: "L2",
			want:  SafeSourceL2,
		},
		{
			name:    "invalid",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StringToSafeSource(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSafeSourceString(t *testing.T) {
	tests := []struct {
		name   string
		source SafeSource
		want   string
	}{
		{
			name:   "L1",
			source: SafeSourceL1,
			want:   "l1",
		},
		{
			name:   "L2",
			source: SafeSourceL2,
			want:   "l2",
		},
		{
			name:   "unknown",
			source: SafeSource(999),
			want:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.source.String()
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSafeSourceSet(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    SafeSource
		wantErr bool
	}{
		{
			name:  "set l1",
			value: "l1",
			want:  SafeSourceL1,
		},
		{
			name:  "set l2",
			value: "l2",
			want:  SafeSourceL2,
		},
		{
			name:    "set invalid",
			value:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s SafeSource
			err := s.Set(tt.value)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, s)
			}
		})
	}
}

func TestSafeSourceClone(t *testing.T) {
	tests := []struct {
		name   string
		source SafeSource
	}{
		{
			name:   "clone L1",
			source: SafeSourceL1,
		},
		{
			name:   "clone L2",
			source: SafeSourceL2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloned := tt.source.Clone()
			require.NotNil(t, cloned)
			clonedPtr, ok := cloned.(*SafeSource)
			require.True(t, ok)
			require.Equal(t, tt.source, *clonedPtr)
			// Verify it's a different pointer
			require.NotSame(t, &tt.source, clonedPtr)
		})
	}
}
