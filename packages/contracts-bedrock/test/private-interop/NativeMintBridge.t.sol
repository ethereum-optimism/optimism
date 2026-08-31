// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { MockHelper } from "test/utils/MockHelper.sol";
import { stdStorage, StdStorage } from "forge-std/StdStorage.sol";

// Contracts
import { NativeMintBridge } from "src/private-interop/NativeMintBridge.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";
import { IETHLockVault } from "interfaces/private-interop/IETHLockVault.sol";
import { INativeMintBridge } from "interfaces/private-interop/INativeMintBridge.sol";

/// @title NativeMintBridge_TestInit
/// @notice Reusable test initialization for `NativeMintBridge` tests.
abstract contract NativeMintBridge_TestInit is Test, MockHelper {
    using stdStorage for StdStorage;

    /// @notice Emitted when the native asset is minted against ETH locked on the counterparty
    ///         chain.
    event MintRelayed(address indexed to, uint256 amount);

    /// @notice Emitted when the native asset is burned and an unlock is requested.
    event BurnAndUnlockSent(address indexed from, address indexed to, uint256 amount, bytes32 msgHash);

    /// @notice Chain ID of the counterparty chain that holds the lock vault.
    uint256 internal constant COUNTERPARTY_CHAIN_ID = 10;

    /// @notice Bridge under test.
    NativeMintBridge internal bridge;

    /// @notice Address standing in for the counterparty chain's `ETHLockVault`.
    address internal lockVault;

    /// @notice Test setup.
    function setUp() public virtual {
        lockVault = makeAddr("lockVault");
        bridge = new NativeMintBridge(COUNTERPARTY_CHAIN_ID, lockVault);

        vm.etch(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            vm.getDeployedCode("L2ToL2CrossDomainMessenger.sol:L2ToL2CrossDomainMessenger")
        );
        vm.etch(Predeploys.LIQUIDITY_CONTROLLER, vm.getDeployedCode("LiquidityController.sol:LiquidityController"));
        vm.etch(Predeploys.NATIVE_ASSET_LIQUIDITY, vm.getDeployedCode("NativeAssetLiquidity.sol:NativeAssetLiquidity"));
        vm.deal(Predeploys.NATIVE_ASSET_LIQUIDITY, 1000 ether);

        // Authorize the bridge as a minter, which is how it is wired into the private chain's
        // genesis.
        stdstore.target(Predeploys.LIQUIDITY_CONTROLLER).sig(ILiquidityController.minters.selector).with_key(
            address(bridge)
        ).checked_write(true);
    }

    /// @notice Mocks the cross domain message context the messenger reports during a relay.
    function _mockContext(address _sender, uint256 _source) internal {
        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(_sender, _source)
        );
    }
}

/// @title NativeMintBridge_RelayMint_Test
/// @notice Tests the `relayMint` function of the `NativeMintBridge` contract.
contract NativeMintBridge_RelayMint_Test is NativeMintBridge_TestInit {
    /// @notice Tests that a relay from the configured lock vault on the configured counterparty
    ///         chain mints native asset to the recipient.
    function testFuzz_relayMint_succeeds(address _to, uint256 _amount) external {
        vm.assume(_to != address(0));
        assumeNotForgeAddress(_to);
        vm.assume(_to != Predeploys.NATIVE_ASSET_LIQUIDITY);
        _amount = bound(_amount, 1, 100 ether);

        _mockContext(lockVault, COUNTERPARTY_CHAIN_ID);

        uint256 toBalanceBefore = _to.balance;

        vm.expectCall(Predeploys.LIQUIDITY_CONTROLLER, abi.encodeCall(ILiquidityController.mint, (_to, _amount)), 1);
        vm.expectEmit(address(bridge));
        emit MintRelayed(_to, _amount);

        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        bridge.relayMint(_to, _amount);

        assertEq(_to.balance, toBalanceBefore + _amount);
    }

    /// @notice Tests that a caller other than the messenger cannot mint.
    function testFuzz_relayMint_notMessenger_reverts(address _caller) external {
        vm.assume(_caller != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        vm.expectRevert(INativeMintBridge.NativeMintBridge_Unauthorized.selector);
        vm.prank(_caller);
        bridge.relayMint(address(0xbeef), 1);
    }

    /// @notice Tests that a relay of a message from anyone other than the lock vault cannot mint.
    function testFuzz_relayMint_wrongCrossDomainSender_reverts(address _sender) external {
        vm.assume(_sender != lockVault);

        _mockContext(_sender, COUNTERPARTY_CHAIN_ID);

        vm.expectRevert(INativeMintBridge.NativeMintBridge_InvalidCrossDomainSender.selector);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        bridge.relayMint(address(0xbeef), 1);
    }

    /// @notice Tests that a relay of a message from the right address on the wrong chain cannot
    ///         mint.
    function testFuzz_relayMint_wrongCrossDomainSource_reverts(uint256 _source) external {
        vm.assume(_source != COUNTERPARTY_CHAIN_ID);

        _mockContext(lockVault, _source);

        vm.expectRevert(INativeMintBridge.NativeMintBridge_InvalidCrossDomainSource.selector);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        bridge.relayMint(address(0xbeef), 1);
    }

    /// @notice Tests that minting to the zero address reverts.
    function test_relayMint_zeroRecipient_reverts() external {
        _mockContext(lockVault, COUNTERPARTY_CHAIN_ID);

        vm.expectRevert(INativeMintBridge.NativeMintBridge_ZeroAddress.selector);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        bridge.relayMint(address(0), 1);
    }

    /// @notice Tests that minting nothing reverts.
    function test_relayMint_zeroAmount_reverts() external {
        _mockContext(lockVault, COUNTERPARTY_CHAIN_ID);

        vm.expectRevert(INativeMintBridge.NativeMintBridge_ZeroAmount.selector);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        bridge.relayMint(address(0xbeef), 0);
    }
}

/// @title NativeMintBridge_BurnAndUnlock_Test
/// @notice Tests the `burnAndUnlock` function of the `NativeMintBridge` contract.
contract NativeMintBridge_BurnAndUnlock_Test is NativeMintBridge_TestInit {
    using stdStorage for StdStorage;

    /// @notice Tests that burning the native asset sends an unlock message to the lock vault on
    ///         the counterparty chain.
    function testFuzz_burnAndUnlock_succeeds(address _to, uint256 _amount, bytes32 _msgHash) external {
        vm.assume(_to != address(0));
        _amount = bound(_amount, 1, 100 ether);

        address burner = makeAddr("burner");
        vm.deal(burner, _amount);

        uint256 liquidityBalanceBefore = Predeploys.NATIVE_ASSET_LIQUIDITY.balance;

        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(
                IL2ToL2CrossDomainMessenger.sendMessage,
                (COUNTERPARTY_CHAIN_ID, lockVault, abi.encodeCall(IETHLockVault.unlock, (_to, _amount)))
            ),
            abi.encode(_msgHash)
        );

        vm.expectEmit(address(bridge));
        emit BurnAndUnlockSent(burner, _to, _amount, _msgHash);

        vm.prank(burner);
        bytes32 msgHash = bridge.burnAndUnlock{ value: _amount }(_to);

        assertEq(msgHash, _msgHash);
        assertEq(address(bridge).balance, 0);
        assertEq(Predeploys.NATIVE_ASSET_LIQUIDITY.balance, liquidityBalanceBefore + _amount);
    }

    /// @notice Tests that burning to the zero address reverts.
    function test_burnAndUnlock_zeroRecipient_reverts() external {
        vm.deal(address(this), 1 ether);

        vm.expectRevert(INativeMintBridge.NativeMintBridge_ZeroAddress.selector);
        bridge.burnAndUnlock{ value: 1 ether }(address(0));
    }

    /// @notice Tests that burning nothing reverts.
    function test_burnAndUnlock_zeroAmount_reverts() external {
        vm.expectRevert(INativeMintBridge.NativeMintBridge_ZeroAmount.selector);
        bridge.burnAndUnlock{ value: 0 }(address(0xbeef));
    }

    /// @notice Tests that burning reverts when the bridge is not an authorized minter, which is
    ///         the state of a genesis that forgot to wire it into the `LiquidityController`.
    function test_burnAndUnlock_notAuthorizedMinter_reverts() external {
        stdstore.target(Predeploys.LIQUIDITY_CONTROLLER).sig(ILiquidityController.minters.selector).with_key(
            address(bridge)
        ).checked_write(false);

        vm.deal(address(this), 1 ether);

        vm.expectRevert(ILiquidityController.LiquidityController_Unauthorized.selector);
        bridge.burnAndUnlock{ value: 1 ether }(address(0xbeef));
    }
}

/// @title NativeMintBridge_Uncategorized_Test
/// @notice Tests the configuration getters of the `NativeMintBridge` contract.
contract NativeMintBridge_Uncategorized_Test is NativeMintBridge_TestInit {
    /// @notice Tests that the configuration immutables are exposed.
    function test_configuration_succeeds() external view {
        assertEq(bridge.counterpartyChainId(), COUNTERPARTY_CHAIN_ID);
        assertEq(bridge.lockVault(), lockVault);
        assertTrue(bytes(bridge.version()).length > 0);
    }
}
