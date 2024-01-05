package alphabet

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAlphabetPrestateProvider_AbsolutePreStateCommitment_Succeeds(t *testing.T) {
	provider := AlphabetPrestateProvider{}
	hash, err := provider.AbsolutePreStateCommitment(context.Background())
	require.NoError(t, err)
	expected := common.HexToHash("0x03c7ae758795765c6664a5d39bf63841c71ff191e9189522bad8ebff5d4eca98")
	require.Equal(t, expected, hash)
}
