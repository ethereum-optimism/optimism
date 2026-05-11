package runner

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestNamedPrestateFetcherRequiresBaseURL(t *testing.T) {
	fetcher := &NamedPrestateFetcher{filename: "develop.bin.gz"}

	_, err := fetcher.getPrestate(context.Background(), log.New(), nil, "", t.TempDir(), nil)

	require.ErrorIs(t, err, config.ErrMissingPrestateBaseURL)
}
