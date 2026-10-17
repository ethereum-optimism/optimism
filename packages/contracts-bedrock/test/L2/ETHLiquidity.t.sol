// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Vm, VmSafe } from "forge-std/Vm.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Libraries
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Features } from "src/libraries/Features.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";

// Interfaces
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";

// Error imports
import { Unauthorized } from "src/libraries/errors/CommonErrors.sol";
import { InvalidAmount } from "src/libraries/errors/CommonErrors.sol";

/// @title ETHLiquidity_TestInit
/// @notice Reusable test initialization for `ETHLiquidity` tests.
abstract contract ETHLiquidity_TestInit is CommonTest {
    /// @notice Emitted when an address burns ETH liquidity.
    event LiquidityBurned(address indexed caller, uint256 value);

    /// @notice Emitted when an address mints ETH liquidity.
    event LiquidityMinted(address indexed caller, uint256 value);

    /// @notice Emitted when an address funds the contract.
    event LiquidityFunded(address indexed funder, uint256 amount);

    /// @notice Emitted when the pause state synced from L1 is applied.
    event PausedSet(bool paused, uint256 messageNonce);

    /// @notice Gas limit the L1CrossDomainMessenger gives `setPaused` on L2.
    uint32 internal constant SET_PAUSED_GAS_LIMIT = 100_000;

    /// @notice The starting balance of the ETHLiquidity contract.
    uint256 public constant STARTING_LIQUIDITY_BALANCE = type(uint248).max;

    /// @notice Nonce of the next message these tests relay from the L1CrossDomainMessenger.
    uint240 internal l1MessageNonce;

    /// @notice Test setup.
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();

        {
            // TODO: Remove this block when L2Genesis includes this contract.
            vm.etch(address(superchainETHBridge), vm.getDeployedCode("SuperchainETHBridge.sol:SuperchainETHBridge"));
            vm.etch(address(ethLiquidity), vm.getDeployedCode("ETHLiquidity.sol:ETHLiquidity"));
            vm.deal(address(ethLiquidity), type(uint248).max);
        }
    }

    /// @notice Tests that contract is set up with the correct starting balance, unpaused.
    function test_setup_succeeds() public view {
        // Assert
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE);
        assertFalse(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedMessageNonce(), 0);
    }

    /// @notice Returns an L1CrossDomainMessenger message nonce newer than any before.
    function _nextNonce() internal returns (uint256) {
        return Encoding.encodeVersionedNonce({ _nonce: l1MessageNonce++, _version: 1 });
    }

    /// @notice Relays message `_nonce` from `_sender` on L1, as the L1CrossDomainMessenger's
    ///         deposit does, with `_gas` for the relay. A relay that fails lands in the
    ///         L2CrossDomainMessenger's failed messages.
    /// @return versionedHash_ Hash of the relayed message.
    function _relayFromL1(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _minGasLimit,
        bytes memory _message,
        uint256 _gas
    )
        internal
        returns (bytes32 versionedHash_)
    {
        address aliasedL1Messenger = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));
        vm.deal(aliasedL1Messenger, _value);
        vm.prank(aliasedL1Messenger);
        l2CrossDomainMessenger.relayMessage{ value: _value, gas: _gas }(
            _nonce, _sender, _target, _value, _minGasLimit, _message
        );
        versionedHash_ = Hashing.hashCrossDomainMessageV1(_nonce, _sender, _target, _value, _minGasLimit, _message);
    }

    /// @notice Relays `setPaused(_paused, _nonce)` as message `_nonce` from this chain's
    ///         L1CrossDomainMessenger, as `L1CrossDomainMessenger.syncETHLiquidityPause` sends it.
    /// @return Hash of the relayed message.
    function _relaySetPaused(bool _paused, uint256 _nonce, uint256 _gas) internal returns (bytes32) {
        return _relayFromL1(
            _nonce,
            address(l1CrossDomainMessenger),
            Predeploys.ETH_LIQUIDITY,
            0,
            SET_PAUSED_GAS_LIMIT,
            abi.encodeCall(IETHLiquidity.setPaused, (_paused, _nonce)),
            _gas
        );
    }

    /// @notice Syncs `_paused` in a message newer than any before, and checks that it applied.
    /// @param _paused Whether to pause the contract.
    function _setPaused(bool _paused) internal {
        _relaySetPaused(_paused, _nextNonce(), 1_000_000);
        assertEq(ethLiquidity.paused(), _paused);
    }
}

/// @title ETHLiquidity_Version_Test
/// @notice Tests the `version` function of the `ETHLiquidity` contract.
contract ETHLiquidity_Version_Test is ETHLiquidity_TestInit {
    /// @notice Tests that the version returns a valid semver string.
    function test_version_validSemver_succeeds() external view {
        SemverComp.parse(ethLiquidity.version());
    }
}

/// @title ETHLiquidity_Burn_Test
/// @notice Tests the `burn` function of the `ETHLiquidity` contract.
contract ETHLiquidity_Burn_Test is ETHLiquidity_TestInit {
    /// @notice Tests that the burn function can always be called by an authorized caller.
    /// @param _amount Amount of ETH (in wei) to call the burn function with.
    function testFuzz_burn_fromAuthorizedCaller_succeeds(uint256 _amount) public {
        // Assume
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(address(superchainETHBridge), _amount);

        // Act
        vm.expectEmit(address(ethLiquidity));
        emit LiquidityBurned(address(superchainETHBridge), _amount);
        vm.prank(address(superchainETHBridge));
        ethLiquidity.burn{ value: _amount }();

        // Assert
        assertEq(address(superchainETHBridge).balance, 0);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE + _amount);
    }

    /// @notice Tests that the burn function always reverts when called by an unauthorized caller.
    /// @param _amount Amount of ETH (in wei) to call the burn function with.
    /// @param _caller Address of the caller to call the burn function with.
    function testFuzz_burn_fromUnauthorizedCaller_fails(uint256 _amount, address _caller) public {
        // Assume
        vm.assume(_caller != address(superchainETHBridge));
        vm.assume(_caller != address(ethLiquidity));
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(_caller, _amount);

        // Act
        vm.prank(_caller);
        vm.expectRevert(Unauthorized.selector);
        ethLiquidity.burn{ value: _amount }();

        // Assert
        assertEq(_caller.balance, _amount);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE);
    }

    /// @notice Tests that the burn function reverts while the contract is paused, even for the
    ///         authorized caller, and works again once it is unpaused.
    /// @param _amount Amount of ETH (in wei) to call the burn function with.
    function testFuzz_burn_whilePaused_reverts(uint256 _amount) public {
        // Assume
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(address(superchainETHBridge), _amount);
        _setPaused(true);

        // Act
        vm.prank(address(superchainETHBridge));
        vm.expectRevert(IETHLiquidity.ETHLiquidity_Paused.selector);
        ethLiquidity.burn{ value: _amount }();

        // Assert: unpausing lets the same burn through.
        _setPaused(false);
        vm.prank(address(superchainETHBridge));
        ethLiquidity.burn{ value: _amount }();
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE + _amount);
    }

    /// @notice Tests that an unauthorized caller still gets `Unauthorized`, not the pause error,
    ///         while the contract is paused.
    function test_burn_unauthorizedWhilePaused_reverts() public {
        // Arrange
        _setPaused(true);

        // Act
        vm.prank(alice);
        vm.expectRevert(Unauthorized.selector);
        ethLiquidity.burn();
    }
}

/// @title ETHLiquidity_Mint_Test
/// @notice Tests the `mint` function of the `ETHLiquidity` contract.
contract ETHLiquidity_Mint_Test is ETHLiquidity_TestInit {
    /// @notice Tests that the mint function fails when the amount
    ///         requested is greater than the available balance.
    /// @param _amount Amount of ETH exceeding the available balance.
    function testFuzz_mint_moreThanAvailableBalance_reverts(uint256 _amount) public {
        // Bound
        _amount = bound(_amount, STARTING_LIQUIDITY_BALANCE + 1, type(uint256).max);

        // Act
        vm.prank(address(superchainETHBridge));
        vm.expectRevert(); // nosemgrep: sol-safety-expectrevert-no-args
        ethLiquidity.mint(_amount);

        // Assert
        assertEq(address(superchainETHBridge).balance, 0);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE);
    }

    /// @notice Tests that the mint function can always be called by an authorized caller.
    /// @param _amount Amount of ETH (in wei) to call the mint function with.
    function testFuzz_mint_fromAuthorizedCaller_succeeds(uint256 _amount) public {
        // Assume
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Get balances before
        uint256 superchainETHBridgeBalanceBefore = address(superchainETHBridge).balance;

        // Act
        vm.expectEmit(address(ethLiquidity));
        emit LiquidityMinted(address(superchainETHBridge), _amount);
        vm.prank(address(superchainETHBridge));
        ethLiquidity.mint(_amount);

        // Assert
        assertEq(address(superchainETHBridge).balance, superchainETHBridgeBalanceBefore + _amount);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE - _amount);
    }

    /// @notice Tests that the mint function always reverts when called by an unauthorized caller.
    /// @param _amount Amount of ETH (in wei) to call the mint function with.
    /// @param _caller Address of the caller to call the mint function with.
    function testFuzz_mint_fromUnauthorizedCaller_fails(uint256 _amount, address _caller) public {
        // Assume
        vm.assume(_caller != address(superchainETHBridge));
        vm.assume(address(_caller).balance == 0);
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        // Nothing to arrange.

        // Act
        vm.prank(_caller);
        vm.expectRevert(Unauthorized.selector);
        ethLiquidity.mint(_amount);

        // Assert
        assertEq(_caller.balance, 0);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE);
        assertEq(address(superchainETHBridge).balance, 0);
    }

    /// @notice Tests that the mint function reverts while the contract is paused, even for the
    ///         authorized caller, and works again once it is unpaused.
    /// @param _amount Amount of ETH (in wei) to call the mint function with.
    function testFuzz_mint_whilePaused_reverts(uint256 _amount) public {
        // Assume
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        _setPaused(true);

        // Act
        vm.prank(address(superchainETHBridge));
        vm.expectRevert(IETHLiquidity.ETHLiquidity_Paused.selector);
        ethLiquidity.mint(_amount);

        // Assert: unpausing lets the same mint through.
        _setPaused(false);
        vm.prank(address(superchainETHBridge));
        ethLiquidity.mint(_amount);
        assertEq(address(superchainETHBridge).balance, _amount);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE - _amount);
    }
}

/// @title ETHLiquidity_Fund_Test
/// @notice Tests the `fund` function of the `ETHLiquidity` contract.
contract ETHLiquidity_Fund_Test is ETHLiquidity_TestInit {
    /// @notice Tests that the fund function succeeds when called with a non-zero value.
    /// @param _amount Amount of ETH (in wei) to call the fund function with.
    /// @param _caller Address of the caller to call the fund function with.
    function testFuzz_fund_succeeds(uint256 _amount, address _caller) public {
        // Assume
        vm.assume(_amount > 0); // Fund amount must be greater than 0
        // Bound amount reasonably, e.g., up to 1 million ETH
        _amount = bound(_amount, 1, STARTING_LIQUIDITY_BALANCE);
        vm.assume(_caller != address(0));
        vm.assume(_caller != address(ethLiquidity)); // Prevent contract from calling itself

        // Arrange
        uint256 initialContractBalance = address(ethLiquidity).balance;
        vm.deal(_caller, _amount);

        // Act
        vm.expectEmit(address(ethLiquidity));
        emit LiquidityFunded(_caller, _amount);
        vm.prank(_caller);
        ethLiquidity.fund{ value: _amount }();

        // Assert
        // Caller should have 0 balance after funding the exact amount they were dealt
        assertEq(_caller.balance, 0);
        assertEq(address(ethLiquidity).balance, initialContractBalance + _amount);
    }

    /// @notice Tests that the fund function still works while the contract is paused.
    /// @param _amount Amount of ETH (in wei) to call the fund function with.
    function testFuzz_fund_whilePaused_succeeds(uint256 _amount) public {
        // Assume
        _amount = bound(_amount, 1, STARTING_LIQUIDITY_BALANCE);

        // Arrange
        _setPaused(true);
        vm.deal(alice, _amount);

        // Act
        vm.expectEmit(address(ethLiquidity));
        emit LiquidityFunded(alice, _amount);
        vm.prank(alice);
        ethLiquidity.fund{ value: _amount }();

        // Assert
        assertEq(alice.balance, 0);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE + _amount);
    }

    /// @notice Tests that the fund function reverts when called with zero value.
    function test_fund_zeroAmount_reverts() public {
        // Arrange
        // Nothing to arrange.

        // Act
        vm.expectRevert(InvalidAmount.selector);
        ethLiquidity.fund{ value: 0 }();

        // Assert
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE);
    }
}

/// @title ETHLiquidity_SetPaused_Test
/// @notice Tests the `setPaused` function of the `ETHLiquidity` contract.
contract ETHLiquidity_SetPaused_Test is ETHLiquidity_TestInit {
    /// @notice Tests that the pause state relayed from this chain's L1CrossDomainMessenger is
    ///         applied, with its message nonce, and reported.
    function test_setPaused_fromL1Messenger_succeeds() public {
        uint256 pauseNonce = _nextNonce();
        vm.expectEmit(address(ethLiquidity));
        emit PausedSet(true, pauseNonce);
        bytes32 pauseHash = _relaySetPaused(true, pauseNonce, 1_000_000);
        assertTrue(l2CrossDomainMessenger.successfulMessages(pauseHash));
        assertTrue(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedMessageNonce(), pauseNonce);

        uint256 unpauseNonce = _nextNonce();
        vm.expectEmit(address(ethLiquidity));
        emit PausedSet(false, unpauseNonce);
        _relaySetPaused(false, unpauseNonce, 1_000_000);
        assertFalse(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedMessageNonce(), unpauseNonce);
    }

    /// @notice Tests that no caller but the L2CrossDomainMessenger can set the pause state.
    /// @param _caller       Address of the caller.
    /// @param _paused       Value to set.
    /// @param _messageNonce Message nonce to set.
    function testFuzz_setPaused_notL2Messenger_reverts(address _caller, bool _paused, uint256 _messageNonce) public {
        vm.assume(_caller != Predeploys.L2_CROSS_DOMAIN_MESSENGER);

        vm.expectRevert(IETHLiquidity.ETHLiquidity_NotL1CrossDomainMessenger.selector);
        vm.prank(_caller);
        ethLiquidity.setPaused(_paused, _messageNonce);
    }

    /// @notice Tests that a message relayed from anyone on L1 but this chain's
    ///         L1CrossDomainMessenger lands in the L2CrossDomainMessenger's failed messages and
    ///         leaves the pause state alone.
    /// @param _sender L1 sender of the message.
    function testFuzz_setPaused_notL1Messenger_fails(address _sender) public {
        vm.assume(_sender != address(l1CrossDomainMessenger));
        uint256 nonce = _nextNonce();

        bytes32 hash = _relayFromL1(
            nonce,
            _sender,
            Predeploys.ETH_LIQUIDITY,
            0,
            SET_PAUSED_GAS_LIMIT,
            abi.encodeCall(IETHLiquidity.setPaused, (true, nonce)),
            1_000_000
        );

        assertTrue(l2CrossDomainMessenger.failedMessages(hash));
        assertFalse(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedMessageNonce(), 0);
    }

    /// @notice Tests that replaying an older message that failed for lack of gas, after a newer
    ///         one applied, is consumed and changes nothing.
    function test_setPaused_olderFailedReplay_succeeds() public {
        uint256 pauseNonce = _nextNonce();
        bytes32 pauseHash = _relaySetPaused(true, pauseNonce, 150_000);
        assertTrue(l2CrossDomainMessenger.failedMessages(pauseHash));
        assertFalse(ethLiquidity.paused());

        _setPaused(true);
        _setPaused(false);
        uint256 appliedNonce = ethLiquidity.pausedMessageNonce();

        vm.recordLogs();
        vm.prank(bob);
        l2CrossDomainMessenger.relayMessage(
            pauseNonce,
            address(l1CrossDomainMessenger),
            Predeploys.ETH_LIQUIDITY,
            0,
            SET_PAUSED_GAS_LIMIT,
            abi.encodeCall(IETHLiquidity.setPaused, (true, pauseNonce))
        );

        assertTrue(l2CrossDomainMessenger.successfulMessages(pauseHash));
        assertEq(vm.getRecordedLogs().length, 1, "only the messenger's RelayedMessage");
        assertFalse(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedMessageNonce(), appliedNonce);
    }

    /// @notice Tests that a message with the applied message's nonce, which the
    ///         L1CrossDomainMessenger never sends, is consumed and changes nothing.
    function test_setPaused_sameNonce_succeeds() public {
        uint256 nonce = _nextNonce();
        _relaySetPaused(true, nonce, 1_000_000);

        vm.recordLogs();
        bytes32 hash = _relayFromL1(
            nonce,
            address(l1CrossDomainMessenger),
            Predeploys.ETH_LIQUIDITY,
            0,
            SET_PAUSED_GAS_LIMIT,
            abi.encodeCall(IETHLiquidity.setPaused, (false, nonce)),
            1_000_000
        );

        assertTrue(l2CrossDomainMessenger.successfulMessages(hash));
        assertEq(vm.getRecordedLogs().length, 1, "only the messenger's RelayedMessage");
        assertTrue(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedMessageNonce(), nonce);
    }

    /// @notice Tests that `setPaused` fits in the gas the L2CrossDomainMessenger guarantees it:
    ///         relayed through ETHLiquidity's proxy on cold accounts, with just enough gas to pass
    ///         the messenger's minimum gas check, it applies.
    function test_setPaused_minRelayGas_succeeds() public {
        address implementation = makeAddr("ETHLiquidity implementation");
        vm.etch(implementation, vm.getDeployedCode("ETHLiquidity.sol:ETHLiquidity"));
        vm.etch(Predeploys.ETH_LIQUIDITY, DeployUtils.getDeployedCode("Proxy"));
        vm.store(
            Predeploys.ETH_LIQUIDITY, Constants.PROXY_IMPLEMENTATION_ADDRESS, bytes32(uint256(uint160(implementation)))
        );
        uint256 nonce = _nextNonce();
        uint256 gas = _minRelayGas(nonce, implementation);

        _coolRelay(implementation);
        bytes32 hash = _relaySetPaused(true, nonce, gas);

        assertTrue(l2CrossDomainMessenger.successfulMessages(hash));
        assertTrue(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedMessageNonce(), nonce);
    }

    /// @notice Tests that the pause flag lives at storage slot 0 and its message nonce at slot 1,
    ///         where the storage layout snapshot puts them.
    function test_setPaused_storageSlots_succeeds() public {
        assertEq(vm.load(address(ethLiquidity), bytes32(uint256(0))), bytes32(0));
        assertEq(vm.load(address(ethLiquidity), bytes32(uint256(1))), bytes32(0));

        uint256 nonce = _nextNonce();
        _relaySetPaused(true, nonce, 1_000_000);

        assertEq(vm.load(address(ethLiquidity), bytes32(uint256(0))), bytes32(uint256(1)));
        assertEq(vm.load(address(ethLiquidity), bytes32(uint256(1))), bytes32(nonce));
    }

    /// @notice Cools every account the relay of `setPaused` touches, as at the start of its
    ///         deposit.
    function _coolRelay(address _implementation) internal {
        vm.cool(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        vm.cool(
            address(
                uint160(uint256(vm.load(Predeploys.L2_CROSS_DOMAIN_MESSENGER, Constants.PROXY_IMPLEMENTATION_ADDRESS)))
            )
        );
        vm.cool(Predeploys.ETH_LIQUIDITY);
        vm.cool(_implementation);
    }

    /// @notice Returns the least gas with which the relay of `setPaused(true, _nonce)` passes the
    ///         L2CrossDomainMessenger's minimum gas check and calls ETHLiquidity.
    function _minRelayGas(uint256 _nonce, address _implementation) internal returns (uint256 gas_) {
        gas_ = 100_000;
        uint256 high = 1_000_000;
        while (gas_ < high) {
            uint256 mid = (gas_ + high) / 2;
            uint256 snapshot = vm.snapshotState();
            _coolRelay(_implementation);
            vm.startStateDiffRecording();
            _relaySetPaused(true, _nonce, mid);
            VmSafe.AccountAccess[] memory accesses = vm.stopAndReturnStateDiff();
            vm.revertToState(snapshot);

            bool called;
            for (uint256 i; i < accesses.length; i++) {
                if (
                    accesses[i].account == Predeploys.ETH_LIQUIDITY && accesses[i].kind == VmSafe.AccountAccessKind.Call
                ) {
                    called = true;
                }
            }
            if (called) high = mid;
            else gas_ = mid + 1;
        }
    }
}

/// @title ETHLiquidity_Integration_Test
/// @notice Tests syncing this chain's L1 pause onto `ETHLiquidity` through the real
///         L1CrossDomainMessenger.
contract ETHLiquidity_Integration_Test is ETHLiquidity_TestInit {
    /// @notice A message this chain's L1CrossDomainMessenger sent.
    struct SentL1Message {
        uint256 nonce;
        address sender;
        address target;
        bytes message;
        uint256 minGasLimit;
    }

    /// @notice Test setup.
    function setUp() public override {
        super.setUp();
        if (!systemConfig.isFeatureEnabled(Features.INTEROP)) {
            vm.prank(proxyAdminOwner);
            systemConfig.setFeature(Features.INTEROP, true);
        }
    }

    /// @notice Tests the whole pause path: the guardian pauses this chain on L1, anyone syncs it,
    ///         and sends revert; the guardian unpauses, anyone syncs again, and sends go through.
    function test_sendETH_l1PauseSynced_reverts() public {
        vm.deal(alice, 1 ether);
        _setL1Paused(true);
        _relay(_sync(), 0, 1_000_000);
        assertTrue(ethLiquidity.paused());

        vm.expectRevert(IETHLiquidity.ETHLiquidity_Paused.selector);
        vm.prank(alice);
        superchainETHBridge.sendETH{ value: 1 ether }(bob, 902);

        _setL1Paused(false);
        _relay(_sync(), 0, 1_000_000);
        assertFalse(ethLiquidity.paused());

        vm.prank(alice);
        superchainETHBridge.sendETH{ value: 1 ether }(bob, 902);
        assertEq(alice.balance, 0);
    }

    /// @notice Tests that two syncs in one L1 block around an unpause apply in message order: if
    ///         the pause sync fails on L2 and is replayed after the unpause sync, the chain stays
    ///         unpaused.
    function test_setPaused_syncsInOneL1Block_succeeds() public {
        _setL1Paused(true);
        SentL1Message memory pauseSync = _sync();
        _setL1Paused(false);
        SentL1Message memory unpauseSync = _sync();
        assertGt(unpauseSync.nonce, pauseSync.nonce);

        bytes32 pauseHash = _relay(pauseSync, 0, 150_000);
        assertTrue(l2CrossDomainMessenger.failedMessages(pauseHash));
        _relay(unpauseSync, 0, 1_000_000);
        assertFalse(ethLiquidity.paused());

        vm.prank(bob);
        l2CrossDomainMessenger.relayMessage(
            pauseSync.nonce, pauseSync.sender, pauseSync.target, 0, pauseSync.minGasLimit, pauseSync.message
        );
        assertTrue(l2CrossDomainMessenger.successfulMessages(pauseHash));
        assertFalse(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedMessageNonce(), unpauseSync.nonce);
    }

    /// @notice Tests that an ETH deposit through the standard bridge still lands on L2 while this
    ///         chain is paused and ETHLiquidity mirrors it.
    function test_finalizeBridgeETH_whilePaused_succeeds() public {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);
        _setL1Paused(true);
        _relay(_sync(), 0, 1_000_000);
        assertTrue(ethLiquidity.paused());
        uint256 bobBalanceBefore = bob.balance;

        vm.deal(alice, 1 ether);
        vm.recordLogs();
        vm.prank(alice);
        l1StandardBridge.bridgeETHTo{ value: 1 ether }(bob, 200_000, hex"");
        bytes32 hash = _relay(_lastSentMessage(), 1 ether, 1_000_000);

        assertTrue(l2CrossDomainMessenger.successfulMessages(hash));
        assertEq(bob.balance, bobBalanceBefore + 1 ether);
    }

    /// @notice Pauses or unpauses every chain on L1, as the guardian.
    function _setL1Paused(bool _paused) internal {
        vm.prank(superchainConfig.guardian());
        if (_paused) superchainConfig.pause(address(0));
        else superchainConfig.unpause(address(0));
    }

    /// @notice Syncs this chain's L1 pause and returns the message it sent, which carries its own
    ///         nonce.
    function _sync() internal returns (SentL1Message memory m_) {
        vm.recordLogs();
        vm.prank(alice);
        l1CrossDomainMessenger.syncETHLiquidityPause();
        m_ = _lastSentMessage();
        assertEq(m_.target, Predeploys.ETH_LIQUIDITY);
        assertEq(m_.message, abi.encodeCall(IETHLiquidity.setPaused, (l1CrossDomainMessenger.paused(), m_.nonce)));
    }

    /// @notice Returns the last message the L1CrossDomainMessenger sent in the recorded logs.
    function _lastSentMessage() internal view returns (SentL1Message memory m_) {
        Vm.Log[] memory logs = vm.getRecordedLogs();
        for (uint256 i = logs.length; i > 0; i--) {
            Vm.Log memory log = logs[i - 1];
            if (
                log.emitter == address(l1CrossDomainMessenger) && log.topics.length > 0
                    && log.topics[0] == SentMessage.selector
            ) {
                m_.target = address(uint160(uint256(log.topics[1])));
                (m_.sender, m_.message, m_.nonce, m_.minGasLimit) =
                    abi.decode(log.data, (address, bytes, uint256, uint256));
                return m_;
            }
        }
        revert("ETHLiquidity_Integration_Test: no SentMessage from the L1CrossDomainMessenger");
    }

    /// @notice Relays `_m` with `_value` on L2, as its deposit does, with `_gas` for the relay.
    function _relay(SentL1Message memory _m, uint256 _value, uint256 _gas) internal returns (bytes32) {
        return _relayFromL1(_m.nonce, _m.sender, _m.target, _value, _m.minGasLimit, _m.message, _gas);
    }
}
