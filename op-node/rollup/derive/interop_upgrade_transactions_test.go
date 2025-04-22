package derive

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestInteropSourcesMatchSpec(t *testing.T) {
	for _, test := range []struct {
		source       UpgradeDepositSource
		expectedHash string
	}{
		{
			source:       deployCrossL2InboxSource,
			expectedHash: "0x6e5e214f73143df8fe6f6054a3ed7eb472d373376458a9c8aecdf23475beb616",
		},
		{
			source:       updateCrossL2InboxProxySource,
			expectedHash: "0x88c6b48354c367125a59792a93a7b60ad7cd66e516157dbba16558c68a46d3cb",
		},
		{
			source:       deployL2ToL2MessengerSource,
			expectedHash: "0xf5484697c7a9a791db32a3bf0763bf2ba686c77ae7d4c0a5ee8c222a92a8dcc2",
		},
		{
			source:       updateL2ToL2MessengerProxySource,
			expectedHash: "0xe54b4d06bbcc857f41ae00e89d820339ac5ce0034aac722c817b2873e03a7e68",
		},
	} {
		require.Equal(t, common.HexToHash(test.expectedHash), test.source.SourceHash(), "Source hash mismatch for intent: %s", test.source.Intent)
	}
}

func TestInteropNetworkTransactions(t *testing.T) {
	upgradeTxns, err := InteropNetworkUpgradeTransactions()
	require.NoError(t, err)
	require.Len(t, upgradeTxns, 4)

	// 1. Deploy CrossL2Inbox
	sender1, tx1 := toDepositTxn(t, upgradeTxns[0])
	require.Equal(t, crossL2InboxDeployerAddress, sender1, "sender mismatch tx 1")
	require.Equal(t, deployCrossL2InboxSource.SourceHash(), tx1.SourceHash(), "source hash mismatch tx 1")
	require.Nil(t, tx1.To(), "to mismatch tx 1")
	require.Equal(t, uint64(420000), tx1.Gas(), "gas mismatch tx 1")
	require.Equal(t, crossL2InboxDeploymentBytecode, tx1.Data(), "data mismatch tx 1")

	// 2. Update CrossL2Inbox Proxy
	sender2, tx2 := toDepositTxn(t, upgradeTxns[1])
	require.Equal(t, common.Address{}, sender2, "sender mismatch tx 2")
	require.Equal(t, updateCrossL2InboxProxySource.SourceHash(), tx2.SourceHash(), "source hash mismatch tx 2")
	require.NotNil(t, tx2.To(), "to mismatch tx 2")
	require.Equal(t, predeploys.CrossL2InboxAddr, *tx2.To(), "to mismatch tx 2")
	require.Equal(t, uint64(50_000), tx2.Gas(), "gas mismatch tx 2")
	expectedData, _ := hex.DecodeString("3659cfe6000000000000000000000000691300f512e48b463c2617b34eef1a9f82ee7dbf")
	require.Equal(t, expectedData, tx2.Data(), "data mismatch tx 2")

	// 3. Deploy L2ToL2CrossDomainMessenger
	sender3, tx3 := toDepositTxn(t, upgradeTxns[2])
	require.Equal(t, l2ToL2MessengerDeployerAddress, sender3, "sender mismatch tx 3")
	require.Equal(t, deployL2ToL2MessengerSource.SourceHash(), tx3.SourceHash(), "source hash mismatch tx 3")
	require.Nil(t, tx3.To(), "to mismatch tx 3")
	require.Equal(t, uint64(1100000), tx3.Gas(), "gas mismatch tx 3")
	require.Equal(t, l2ToL2MessengerDeploymentBytecode, tx3.Data(), "data mismatch tx 3")

	// 4. Update L2ToL2CrossDomainMessenger Proxy
	sender4, tx4 := toDepositTxn(t, upgradeTxns[3])
	require.Equal(t, common.Address{}, sender4, "sender mismatch tx 4")
	require.Equal(t, updateL2ToL2MessengerProxySource.SourceHash(), tx4.SourceHash(), "source hash mismatch tx 4")
	require.NotNil(t, tx4.To(), "to mismatch tx 4")
	require.Equal(t, predeploys.L2toL2CrossDomainMessengerAddr, *tx4.To(), "to mismatch tx 4")
	require.Equal(t, uint64(50_000), tx4.Gas(), "gas mismatch tx 4")
	expectedData, _ = hex.DecodeString("3659cfe60000000000000000000000000d0edd0ebd0e94d218670a8de867eb5c4d37cadd")
	require.Equal(t, expectedData, tx4.Data(), "data mismatch tx 4")
}
