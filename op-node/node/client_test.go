package node

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHTTPHeader(t *testing.T) {
	for _, test := range []struct {
		desc   string
		str    string
		expHdr http.Header
		expErr bool
	}{
		{
			desc:   "err-empty",
			expErr: true,
		},
		{
			desc:   "err-no-colon",
			str:    "Key",
			expErr: true,
		},
		{
			desc:   "err-only-key",
			str:    "Key:",
			expErr: true,
		},
		{
			desc:   "err-no-space",
			str:    "Key:value",
			expErr: true,
		},
		{
			desc:   "valid",
			str:    "Key: value",
			expHdr: http.Header{"Key": []string{"value"}},
		},
		{
			desc:   "valid-small",
			str:    "key: value",
			expHdr: http.Header{"Key": []string{"value"}},
		},
		{
			desc:   "valid-spaces-colons",
			str:    "X-Key: a long value with spaces: and: colons",
			expHdr: http.Header{"X-Key": []string{"a long value with spaces: and: colons"}},
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			h, err := parseHTTPHeader(test.str)
			if test.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expHdr, h)
			}
		})
	}
}

func TestL1BeaconEndpointConfig_Setup(t *testing.T) {
	for _, test := range []struct {
		desc string
		baa  []string
		len  int
	}{
		{
			desc: "empty",
		},
		{
			desc: "one",
			baa:  []string{"http://foo.bar"},
			len:  1,
		},
		{
			desc: "three",
			baa:  []string{"http://foo.bar", "http://op.ti", "http://ba.se"},
			len:  3,
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			cfg := L1BeaconEndpointConfig{BeaconFallbackAddrs: test.baa}
			_, fb, err := cfg.Setup(context.Background(), nil)
			require.NoError(t, err)
			require.Len(t, fb, test.len)
		})
	}
}
