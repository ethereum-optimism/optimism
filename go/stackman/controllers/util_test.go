package controllers

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHostify(t *testing.T) {
	tests := [][2]string{
		{
			"https://test.infura.io/v1/123456",
			"test.infura.io:443",
		},
		{
			"http://test.infura.io/v1/123456",
			"test.infura.io:80",
		},
		{
			"test.infura.io/v1/123456",
			"test.infura.io:80",
		},
		{
			"test.infura.io",
			"test.infura.io:80",
		},
		{
			"http://sequencer:8545",
			"sequencer:8545",
		},
	}
	for _, tt := range tests {
		t.Run(tt[0], func(t *testing.T) {
			require.Equal(t, tt[1], Hostify(tt[0]))
		})
	}
}
