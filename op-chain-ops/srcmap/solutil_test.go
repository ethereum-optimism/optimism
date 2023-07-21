package srcmap

import (
	"path"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
)

func TestSourcemap(t *testing.T) {
	contractsDir := "../../packages/contracts-bedrock"
	sources := []string{path.Join(contractsDir, "contracts/cannon/MIPS.sol")}
	sources = append(sources, bindings.Sources...)
	for i, source := range sources {
		// Add relative path to contracts directory if the source is not
		// already relativized.
		if !strings.HasPrefix(source, "..") {
			sources[i] = path.Join(contractsDir, source)
		}
	}

	deployedByteCode := hexutil.MustDecode(bindings.MIPSDeployedBin)
	srcMap, err := ParseSourceMap(
		sources,
		deployedByteCode,
		bindings.MIPSDeployedSourceMap)
	require.NoError(t, err)

	for i := 0; i < len(deployedByteCode); i++ {
		info := srcMap.FormattedInfo(uint64(i))
		if strings.HasPrefix(info, "unexpected") {
			t.Fatalf("unexpected info: %q", info)
		}
	}
}
