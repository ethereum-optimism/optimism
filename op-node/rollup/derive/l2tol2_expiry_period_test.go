package derive

import (
	"bytes"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
)

// productionL2ToL2MessageExpiryPeriod is the L2ToL2CrossDomainMessenger's expiry period on
// production networks: the 7-day interop message expiry window plus a day of margin.
const productionL2ToL2MessageExpiryPeriod = 8 * 24 * 60 * 60

// l2ToL2MessengerExpiryPeriod returns the expiry period a NUT bundle deploys the
// L2ToL2CrossDomainMessenger implementation with: the deployment calls
// ConditionalDeployer.deploy(bytes32 salt, bytes code), and the code ends with the constructor's
// one argument.
func l2ToL2MessengerExpiryPeriod(t *testing.T, bundle *nutBundle) *big.Int {
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	require.NoError(t, err)
	bytesType, err := abi.NewType("bytes", "", nil)
	require.NoError(t, err)
	deployArgs := abi.Arguments{{Type: bytes32Type}, {Type: bytesType}}

	for _, tx := range bundle.Transactions {
		if tx.Intent != "Deploy L2ToL2CrossDomainMessenger Implementation" {
			continue
		}
		require.Greater(t, len(tx.Data), 4, "the deployment must carry calldata")
		values, err := deployArgs.Unpack(tx.Data[4:])
		require.NoError(t, err)
		code := values[1].([]byte)
		require.GreaterOrEqual(t, len(code), 32, "the deployment code must end with the constructor argument")
		return new(big.Int).SetBytes(code[len(code)-32:])
	}
	t.Fatal("the bundle does not deploy the L2ToL2CrossDomainMessenger implementation")
	return nil
}

// TestCurrentNUTBundleL2ToL2MessengerExpiryPeriod checks that the current NUT bundle deploys the
// L2ToL2CrossDomainMessenger with the production expiry period.
func TestCurrentNUTBundleL2ToL2MessengerExpiryPeriod(t *testing.T) {
	bundleJSON, err := os.ReadFile("../../../packages/contracts-bedrock/snapshots/upgrades/current-upgrade-bundle.json")
	require.NoError(t, err)
	bundle, err := readNUTBundle("current", bytes.NewReader(bundleJSON))
	require.NoError(t, err)
	require.Equal(t, big.NewInt(productionL2ToL2MessageExpiryPeriod), l2ToL2MessengerExpiryPeriod(t, bundle))
}
