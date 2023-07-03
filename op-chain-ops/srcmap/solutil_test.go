package srcmap

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
)

func TestSourcemap(t *testing.T) {
	sourcePath := "../../packages/contracts-bedrock/contracts/cannon/MIPS.sol"
	deployedByteCode := hexutil.MustDecode(bindings.MIPSDeployedBin)
	srcMap, err := ParseSourceMap(
		[]string{sourcePath},
		deployedByteCode,
		bindings.MIPSDeployedSourceMap)
	require.NoError(t, err)

	for i := 0; i < len(deployedByteCode); i++ {
		info := srcMap.FormattedInfo(uint64(i))
		if !strings.HasPrefix(info, "generated:") && !strings.HasPrefix(info, sourcePath) {
			t.Fatalf("unexpected info: %q", info)
		}
	}
}
