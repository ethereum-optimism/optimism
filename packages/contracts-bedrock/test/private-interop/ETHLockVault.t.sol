// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { MockHelper } from "test/utils/MockHelper.sol";
import { stdStorage, StdStorage } from "forge-std/StdStorage.sol";

// Contracts
import { ETHLockVault } from "src/private-interop/ETHLockVault.sol";
import { NativeMintBridge } from "src/private-interop/NativeMintBridge.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";
import { IETHLockVault } from "interfaces/private-interop/IETHLockVault.sol";
import { INativeMintBridge } from "interfaces/private-interop/INativeMintBridge.sol";

/// @title ETHLockVault_TestInit
/// @notice Reusable test initialization for `ETHLockVault` tests.
abstract contract ETHLockVault_TestInit is Test, MockHelper {
    /// @notice Emitted when ETH is locked and a mint is requested on the private chain.
    event ETHLocked(address indexed from, address indexed recipient, uint256 amount, bytes32 msgHash);

    /// @notice Emitted when ETH is unlocked on behalf of the private chain.
    event ETHUnlocked(address indexed to, uint256 amount);

    /// @notice Chain ID of the private chain the vault bridges to.
    uint256 internal constant PRIVATE_CHAIN_ID = 424_243;

    /// @notice Vault under test.
    ETHLockVault internal vault;

    /// @notice Address standing in for the private chain's `NativeMintBridge`.
    address internal privateBridge;

    /// @notice Test setup.
    function setUp() public virtual {
        privateBridge = makeAddr("privateBridge");
        vault = new ETHLockVault(PRIVATE_CHAIN_ID, privateBridge);

        // Give the messenger predeploy real code so that calls into it are not calls into an empty
        // account. Individual behaviors are mocked per test.
        vm.etch(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            vm.getDeployedCode("L2ToL2CrossDomainMessenger.sol:L2ToL2CrossDomainMessenger")
        );
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

/// @title ETHLockVault_Lock_Test
/// @notice Tests the `lock` function of the `ETHLockVault` contract.
contract ETHLockVault_Lock_Test is ETHLockVault_TestInit {
    /// @notice Tests that locking holds the ETH and sends a mint message to the private-side
    ///         bridge encoding the recipient and the amount.
    function testFuzz_lock_succeeds(address _recipient, uint256 _amount, bytes32 _msgHash) external {
        vm.assume(_recipient != address(0));
        address _from = makeAddr("locker");
        _amount = bound(_amount, 1, type(uint128).max);
        vm.deal(_from, _amount);

        bytes memory message = abi.encodeCall(INativeMintBridge.relayMint, (_recipient, _amount));
        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.sendMessage, (PRIVATE_CHAIN_ID, privateBridge, message)),
            abi.encode(_msgHash)
        );

        vm.expectEmit(address(vault));
        emit ETHLocked(_from, _recipient, _amount, _msgHash);

        uint256 vaultBalanceBefore = address(vault).balance;

        vm.prank(_from);
        bytes32 msgHash = vault.lock{ value: _amount }(_recipient);

        assertEq(msgHash, _msgHash);
        assertEq(address(vault).balance, vaultBalanceBefore + _amount);
        assertEq(vault.totalLocked(), _amount);
    }

    /// @notice Tests that locking to the zero address reverts.
    function test_lock_zeroRecipient_reverts() external {
        vm.deal(address(this), 1 ether);
        vm.expectRevert(IETHLockVault.ETHLockVault_ZeroAddress.selector);
        vault.lock{ value: 1 ether }(address(0));
    }

    /// @notice Tests that locking nothing reverts.
    function test_lock_zeroAmount_reverts() external {
        vm.expectRevert(IETHLockVault.ETHLockVault_ZeroAmount.selector);
        vault.lock{ value: 0 }(address(0xbeef));
    }
}

/// @title ETHLockVault_Unlock_Test
/// @notice Tests the `unlock` function of the `ETHLockVault` contract.
contract ETHLockVault_Unlock_Test is ETHLockVault_TestInit {
    /// @notice Tests that a relay from the configured private-side bridge on the configured
    ///         private chain releases the ETH.
    function testFuzz_unlock_succeeds(address _to, uint256 _amount) external {
        vm.assume(_to != address(0));
        assumeNotForgeAddress(_to);
        vm.assume(_to != address(vault));
        _amount = bound(_amount, 1, type(uint128).max);
        address locker = makeAddr("locker");
        vm.deal(locker, _amount);
        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(
                IL2ToL2CrossDomainMessenger.sendMessage,
                (PRIVATE_CHAIN_ID, privateBridge, abi.encodeCall(INativeMintBridge.relayMint, (locker, _amount)))
            ),
            abi.encode(bytes32(uint256(1)))
        );
        vm.prank(locker);
        vault.lock{ value: _amount }(locker);

        _mockContext(privateBridge, PRIVATE_CHAIN_ID);

        uint256 toBalanceBefore = _to.balance;

        vm.expectEmit(address(vault));
        emit ETHUnlocked(_to, _amount);

        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        vault.unlock(_to, _amount);

        assertEq(_to.balance, toBalanceBefore + _amount);
        assertEq(address(vault).balance, 0);
        assertEq(vault.totalLocked(), 0);
    }

    /// @notice Tests that a caller other than the messenger cannot unlock.
    function testFuzz_unlock_notMessenger_reverts(address _caller) external {
        vm.assume(_caller != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        vm.expectRevert(IETHLockVault.ETHLockVault_Unauthorized.selector);
        vm.prank(_caller);
        vault.unlock(address(0xbeef), 1);
    }

    /// @notice Tests that a relay of a message from anyone other than the private-side bridge
    ///         cannot unlock.
    function testFuzz_unlock_wrongCrossDomainSender_reverts(address _sender) external {
        vm.assume(_sender != privateBridge);

        _mockContext(_sender, PRIVATE_CHAIN_ID);

        vm.expectRevert(IETHLockVault.ETHLockVault_InvalidCrossDomainSender.selector);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        vault.unlock(address(0xbeef), 1);
    }

    /// @notice Tests that a relay of a message from the right address on the wrong chain cannot
    ///         unlock.
    function testFuzz_unlock_wrongCrossDomainSource_reverts(uint256 _source) external {
        vm.assume(_source != PRIVATE_CHAIN_ID);

        _mockContext(privateBridge, _source);

        vm.expectRevert(IETHLockVault.ETHLockVault_InvalidCrossDomainSource.selector);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        vault.unlock(address(0xbeef), 1);
    }

    /// @notice Tests that unlocking to the zero address reverts.
    function test_unlock_zeroRecipient_reverts() external {
        _mockContext(privateBridge, PRIVATE_CHAIN_ID);

        vm.expectRevert(IETHLockVault.ETHLockVault_ZeroAddress.selector);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        vault.unlock(address(0), 1);
    }

    /// @notice Tests that unlocking nothing reverts.
    function test_unlock_zeroAmount_reverts() external {
        _mockContext(privateBridge, PRIVATE_CHAIN_ID);

        vm.expectRevert(IETHLockVault.ETHLockVault_ZeroAmount.selector);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        vault.unlock(address(0xbeef), 0);
    }

    /// @notice Tests that forced ETH does not let the private chain withdraw more than entered
    ///         through the authenticated lock path.
    function test_unlock_insufficientLocked_reverts() external {
        vm.deal(address(vault), 1 ether);
        _mockContext(privateBridge, PRIVATE_CHAIN_ID);

        vm.expectRevert(IETHLockVault.ETHLockVault_InsufficientLocked.selector);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        vault.unlock(address(0xbeef), 1 ether);
    }
}

/// @title ETHLockVault_Integration_Test
/// @notice Tests the full lock -> mint -> burn -> unlock round trip across both halves of the
///         application-level ETH bridge, with the cross domain message context mocked in place of
///         a real relay.
contract ETHLockVault_Integration_Test is ETHLockVault_TestInit {
    using stdStorage for StdStorage;

    /// @notice Private-side bridge under test.
    NativeMintBridge internal bridge;

    /// @notice Test setup. Replaces the placeholder private-side bridge with a real one and stands
    ///         up the custom gas token liquidity predeploys it mints and burns through.
    function setUp() public override {
        super.setUp();

        // Each half names the other in an immutable, so the second deployment's address is
        // predicted before the first is built.
        address predictedBridge = vm.computeCreateAddress(address(this), vm.getNonce(address(this)) + 1);
        vault = new ETHLockVault(PRIVATE_CHAIN_ID, predictedBridge);
        bridge = new NativeMintBridge(block.chainid, address(vault));
        assertEq(address(bridge), predictedBridge);

        vm.etch(Predeploys.LIQUIDITY_CONTROLLER, vm.getDeployedCode("LiquidityController.sol:LiquidityController"));
        vm.etch(Predeploys.NATIVE_ASSET_LIQUIDITY, vm.getDeployedCode("NativeAssetLiquidity.sol:NativeAssetLiquidity"));
        vm.deal(Predeploys.NATIVE_ASSET_LIQUIDITY, 1000 ether);

        stdstore.target(Predeploys.LIQUIDITY_CONTROLLER).sig(ILiquidityController.minters.selector).with_key(
            address(bridge)
        ).checked_write(true);
    }

    /// @notice Tests that ETH locked on the counterparty chain becomes native asset on the private
    ///         chain, and that burning it there releases exactly that ETH again.
    function test_lockMintBurnUnlock_succeeds() external {
        address alice = makeAddr("alice");
        address bob = makeAddr("bob");
        uint256 amount = 3 ether;
        vm.deal(alice, amount);

        // 1. Alice locks ETH on the counterparty chain. The vault asks the private chain to mint.
        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(
                IL2ToL2CrossDomainMessenger.sendMessage,
                (PRIVATE_CHAIN_ID, address(bridge), abi.encodeCall(INativeMintBridge.relayMint, (alice, amount)))
            ),
            abi.encode(bytes32(uint256(1)))
        );
        vm.prank(alice);
        vault.lock{ value: amount }(alice);

        assertEq(address(vault).balance, amount);
        assertEq(vault.totalLocked(), amount);
        assertEq(alice.balance, 0);

        // 2. The message is relayed on the private chain and mints native asset to Alice.
        _mockContext(address(vault), block.chainid);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        bridge.relayMint(alice, amount);

        assertEq(alice.balance, amount);

        // 3. Alice burns the native asset on the private chain to release ETH to Bob.
        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(
                IL2ToL2CrossDomainMessenger.sendMessage,
                (block.chainid, address(vault), abi.encodeCall(IETHLockVault.unlock, (bob, amount)))
            ),
            abi.encode(bytes32(uint256(2)))
        );
        vm.prank(alice);
        bridge.burnAndUnlock{ value: amount }(bob);

        assertEq(alice.balance, 0);

        // 4. The unlock message is relayed on the counterparty chain and releases the ETH.
        _mockContext(address(bridge), PRIVATE_CHAIN_ID);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        vault.unlock(bob, amount);

        assertEq(bob.balance, amount);
        assertEq(address(vault).balance, 0);
        assertEq(vault.totalLocked(), 0);
    }
}
