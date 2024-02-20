package node

import (
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
