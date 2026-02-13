package derive

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
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
		{
			source:       deploySuperchainETHBridgeSource,
			expectedHash: "0x53eccc738e298d613b3c3dcc8ad1d9e9626945a2f7b005252c2b57837176d960",
		},
		{
			source:       updateSuperchainETHBridgeProxySource,
			expectedHash: "0x50684989256294e3c64949ea1cf5bad586c7e6b91b8b7f21ee9ef7086efe60db",
		},
		{
			source:       deployETHLiquiditySource,
			expectedHash: "0xceec4ed75501efd5830d25045e10014464155345d91a8c78dba77aed02d5b08b",
		},
		{
			source:       updateETHLiquidityProxySource,
			expectedHash: "0x8c6c281c65cba9a9286233c61c3a1b4d606b899b1aee3b3a7221fd5212b22822",
		},
		{
			source:       interopETHLiquidityFundingSource,
			expectedHash: "0xa9b2a45c225d10db0a0a092d024192968cef10170a82f9d67d2bf0264d0c0555",
		},
	} {
		require.Equal(t, common.HexToHash(test.expectedHash), test.source.SourceHash(), "Source hash mismatch for intent: %s", test.source.Intent)
	}
}

func TestInteropNetworkTransactions(t *testing.T) {
	upgradeTxns, err := InteropNetworkUpgradeTransactions()
	require.NoError(t, err)
	require.Len(t, upgradeTxns, 7)

	// 1. Deploy L2ToL2CrossDomainMessenger
	sender3, tx3 := toDepositTxn(t, upgradeTxns[0])
	require.Equal(t, l2ToL2MessengerDeployerAddress, sender3, "sender mismatch tx 3")
	require.Equal(t, deployL2ToL2MessengerSource.SourceHash(), tx3.SourceHash(), "source hash mismatch tx 3")
	require.Nil(t, tx3.To(), "to mismatch tx 3")
	require.Equal(t, uint64(1100000), tx3.Gas(), "gas mismatch tx 3")
	require.Equal(t, l2ToL2MessengerDeploymentBytecode, tx3.Data(), "data mismatch tx 3")

	// 2. Update L2ToL2CrossDomainMessenger Proxy
	sender4, tx4 := toDepositTxn(t, upgradeTxns[1])
	require.Equal(t, common.Address{}, sender4, "sender mismatch tx 4")
	require.Equal(t, updateL2ToL2MessengerProxySource.SourceHash(), tx4.SourceHash(), "source hash mismatch tx 4")
	require.NotNil(t, tx4.To(), "to mismatch tx 4")
	require.Equal(t, predeploys.L2toL2CrossDomainMessengerAddr, *tx4.To(), "to mismatch tx 4")
	require.Equal(t, uint64(50_000), tx4.Gas(), "gas mismatch tx 4")
	expectedData, _ := hex.DecodeString("3659cfe60000000000000000000000000d0edd0ebd0e94d218670a8de867eb5c4d37cadd")
	require.Equal(t, expectedData, tx4.Data(), "data mismatch tx 4")

	// 3. Deploy SuperchainETHBridge
	sender5, tx5 := toDepositTxn(t, upgradeTxns[2])
	require.Equal(t, superchainETHBridgeDeployerAddress, sender5, "sender mismatch tx 5")
	require.Equal(t, deploySuperchainETHBridgeSource.SourceHash(), tx5.SourceHash(), "source hash mismatch tx 5")
	require.Nil(t, tx5.To(), "to mismatch tx 5")
	require.Equal(t, uint64(500_000), tx5.Gas(), "gas mismatch tx 5")
	require.Equal(t, superchainETHBridgeDeploymentBytecode, tx5.Data(), "data mismatch tx 5")

	// 4. Update SuperchainETHBridge Proxy
	sender6, tx6 := toDepositTxn(t, upgradeTxns[3])
	require.Equal(t, common.Address{}, sender6, "sender mismatch tx 6")
	require.Equal(t, updateSuperchainETHBridgeProxySource.SourceHash(), tx6.SourceHash(), "source hash mismatch tx 6")
	require.NotNil(t, tx6.To(), "to mismatch tx 6")
	require.Equal(t, predeploys.SuperchainETHBridgeAddr, *tx6.To(), "to mismatch tx 6")
	require.Equal(t, uint64(50_000), tx6.Gas(), "gas mismatch tx 6")
	require.Equal(t, upgradeToCalldata(SuperchainETHBridgeAddress), tx6.Data(), "data mismatch tx 6")

	// 5. Deploy ETHLiquidity
	sender7, tx7 := toDepositTxn(t, upgradeTxns[4])
	require.Equal(t, ethLiquidityDeployerAddress, sender7, "sender mismatch tx 7")
	require.Equal(t, deployETHLiquiditySource.SourceHash(), tx7.SourceHash(), "source hash mismatch tx 7")
	require.Nil(t, tx7.To(), "to mismatch tx 7")
	require.Equal(t, uint64(375_000), tx7.Gas(), "gas mismatch tx 7")
	require.Equal(t, ethLiquidityDeploymentBytecode, tx7.Data(), "data mismatch tx 7")

	// 6. Update ETHLiquidity Proxy
	sender8, tx8 := toDepositTxn(t, upgradeTxns[5])
	require.Equal(t, common.Address{}, sender8, "sender mismatch tx 8")
	require.Equal(t, updateETHLiquidityProxySource.SourceHash(), tx8.SourceHash(), "source hash mismatch tx 8")
	require.NotNil(t, tx8.To(), "to mismatch tx 8")
	require.Equal(t, predeploys.ETHLiquidityAddr, *tx8.To(), "to mismatch tx 8")
	require.Equal(t, uint64(50_000), tx8.Gas(), "gas mismatch tx 8")
	require.Equal(t, upgradeToCalldata(ETHLiquidityAddress), tx8.Data(), "data mismatch tx 8")

	// 7. Fund ETHLiquidity
	sender9, tx9 := toDepositTxn(t, upgradeTxns[6])
	require.Equal(t, L1InfoDepositerAddress, sender9, "sender mismatch tx 9")
	require.Equal(t, interopETHLiquidityFundingSource.SourceHash(), tx9.SourceHash(), "source hash mismatch tx 9")
	require.NotNil(t, tx9.To(), "to mismatch tx 9")
	require.Equal(t, predeploys.ETHLiquidityAddr, *tx9.To(), "to mismatch tx 9")
	require.Equal(t, InteropETHLiquidityFundingAmount(), tx9.Mint(), "mint mismatch tx 9")
	require.Equal(t, InteropETHLiquidityFundingAmount(), tx9.Value(), "value mismatch tx 9")
	require.Equal(t, uint64(50_000), tx9.Gas(), "gas mismatch tx 9")
	expectedFundData, _ := hex.DecodeString("b60d4288")
	require.Equal(t, expectedFundData, tx9.Data(), "data mismatch tx 9")
}

func TestInteropActivateCrossL2InboxTransactions(t *testing.T) {
	upgradeTxns, err := InteropActivateCrossL2InboxTransactions()
	require.NoError(t, err)
	require.Len(t, upgradeTxns, 2)

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
}
