// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";

// Interfaces
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";

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
    event PausedSet(bool paused, uint256 l1Timestamp);

    /// @notice Gas limit the L1CrossDomainMessenger gives `setPaused` on L2.
    uint32 internal constant SET_PAUSED_GAS_LIMIT = 100_000;

    /// @notice Nonce of the next message relayed from the L1CrossDomainMessenger.
    uint240 internal l1MessageNonce;

    /// @notice The starting balance of the ETHLiquidity contract.
    uint256 public constant STARTING_LIQUIDITY_BALANCE = type(uint248).max;

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
        assertEq(ethLiquidity.pausedL1Timestamp(), 0);
    }

    /// @notice Relays `setPaused(_paused, _l1Timestamp)` from this chain's L1CrossDomainMessenger,
    ///         as the deposit `L1CrossDomainMessenger.syncETHLiquidityPause` makes does, with
    ///         `_gas` for the relay. A relay that fails lands in the L2CrossDomainMessenger's
    ///         failed messages.
    /// @return nonce_         Nonce of the relayed message.
    /// @return versionedHash_ Hash of the relayed message.
    function _relaySetPaused(
        bool _paused,
        uint256 _l1Timestamp,
        uint256 _gas
    )
        internal
        returns (uint256 nonce_, bytes32 versionedHash_)
    {
        address l1Messenger = address(l1CrossDomainMessenger);
        bytes memory message = abi.encodeCall(IETHLiquidity.setPaused, (_paused, _l1Timestamp));
        nonce_ = Encoding.encodeVersionedNonce({ _nonce: l1MessageNonce++, _version: 1 });
        versionedHash_ = Hashing.hashCrossDomainMessageV1(
            nonce_, l1Messenger, Predeploys.ETH_LIQUIDITY, 0, SET_PAUSED_GAS_LIMIT, message
        );
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(l1Messenger));
        l2CrossDomainMessenger.relayMessage{ gas: _gas }(
            nonce_, l1Messenger, Predeploys.ETH_LIQUIDITY, 0, SET_PAUSED_GAS_LIMIT, message
        );
    }

    /// @notice Syncs `_paused` from L1, read now, and checks that it applied.
    /// @param _paused Whether to pause the contract.
    function _setPaused(bool _paused) internal {
        _relaySetPaused(_paused, block.timestamp, 1_000_000);
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

        // Assert
        assertEq(address(superchainETHBridge).balance, _amount);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE);

        // Unpausing lets the same burn through.
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

        // Assert
        assertEq(address(superchainETHBridge).balance, 0);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE);

        // Unpausing lets the same mint through.
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
    ///         applied, with its L1 timestamp, and reported.
    function test_setPaused_fromL1Messenger_succeeds() public {
        vm.expectEmit(address(ethLiquidity));
        emit PausedSet(true, 100);
        (, bytes32 pauseHash) = _relaySetPaused(true, 100, 1_000_000);
        assertTrue(l2CrossDomainMessenger.successfulMessages(pauseHash));
        assertTrue(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedL1Timestamp(), 100);

        vm.expectEmit(address(ethLiquidity));
        emit PausedSet(false, 200);
        _relaySetPaused(false, 200, 1_000_000);
        assertFalse(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedL1Timestamp(), 200);
    }

    /// @notice Tests that no caller but the L2CrossDomainMessenger can set the pause state.
    /// @param _caller      Address of the caller.
    /// @param _paused      Value to set.
    /// @param _l1Timestamp L1 timestamp to set.
    function testFuzz_setPaused_notL2Messenger_reverts(address _caller, bool _paused, uint256 _l1Timestamp) public {
        vm.assume(_caller != Predeploys.L2_CROSS_DOMAIN_MESSENGER);

        vm.expectRevert(IETHLiquidity.ETHLiquidity_NotL1CrossDomainMessenger.selector);
        vm.prank(_caller);
        ethLiquidity.setPaused(_paused, _l1Timestamp);

        assertFalse(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedL1Timestamp(), 0);
    }

    /// @notice Tests that a message relayed by the L2CrossDomainMessenger from anyone on L1 but
    ///         this chain's L1CrossDomainMessenger can't set the pause state.
    /// @param _sender L1 sender of the message.
    function testFuzz_setPaused_notL1Messenger_reverts(address _sender) public {
        vm.assume(_sender != address(l2CrossDomainMessenger.otherMessenger()));
        vm.mockCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(_sender)
        );

        vm.expectRevert(IETHLiquidity.ETHLiquidity_NotL1CrossDomainMessenger.selector);
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        ethLiquidity.setPaused(true, block.timestamp);

        assertFalse(ethLiquidity.paused());
    }

    /// @notice Tests that a pause state read on L1 before the applied one changes nothing and
    ///         emits nothing, and that one read at the same L1 timestamp is applied.
    function test_setPaused_staleTimestamp_succeeds() public {
        _relaySetPaused(true, 100, 1_000_000);

        vm.recordLogs();
        (, bytes32 staleHash) = _relaySetPaused(false, 99, 1_000_000);
        // The stale update is consumed as a successful message, and ETHLiquidity logs nothing.
        assertTrue(l2CrossDomainMessenger.successfulMessages(staleHash));
        assertEq(vm.getRecordedLogs().length, 1, "only the messenger's RelayedMessage");
        assertTrue(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedL1Timestamp(), 100);

        _relaySetPaused(false, 100, 1_000_000);
        assertFalse(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedL1Timestamp(), 100);
    }

    /// @notice Tests that replaying an older sync that failed for lack of gas, after a newer sync
    ///         applied, doesn't undo the newer state.
    function test_setPaused_olderFailedReplay_succeeds() public {
        // The pause, read at 100, runs out of gas and lands in the failed messages.
        (uint256 pauseNonce, bytes32 pauseHash) = _relaySetPaused(true, 100, 150_000);
        assertTrue(l2CrossDomainMessenger.failedMessages(pauseHash));
        assertFalse(ethLiquidity.paused());

        // The pause applies on a later sync, and the unpause after it.
        _relaySetPaused(true, 150, 1_000_000);
        assertTrue(ethLiquidity.paused());
        _relaySetPaused(false, 200, 1_000_000);
        assertFalse(ethLiquidity.paused());

        // Anyone replays the failed pause: it is consumed and changes nothing.
        vm.prank(bob);
        l2CrossDomainMessenger.relayMessage(
            pauseNonce,
            address(l1CrossDomainMessenger),
            Predeploys.ETH_LIQUIDITY,
            0,
            SET_PAUSED_GAS_LIMIT,
            abi.encodeCall(IETHLiquidity.setPaused, (true, 100))
        );
        assertTrue(l2CrossDomainMessenger.successfulMessages(pauseHash));
        assertFalse(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedL1Timestamp(), 200);
    }

    /// @notice Tests that the gas the L1CrossDomainMessenger gives `setPaused` covers it on cold
    ///         storage, as in the deposit's own transaction.
    function test_setPaused_gasLimit_succeeds() public {
        vm.cool(address(ethLiquidity));
        vm.mockCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l1CrossDomainMessenger))
        );

        // The real xDomainMessageSender lookup costs under 10k more than the mock, so 90k here
        // means setPaused fits in the 100k the L1CrossDomainMessenger asks for.
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        ethLiquidity.setPaused{ gas: 90_000 }(true, block.timestamp);
        assertTrue(ethLiquidity.paused());
    }

    /// @notice Tests that the pause flag lives at storage slot 0 and its L1 timestamp at slot 1,
    ///         where the storage layout snapshot puts them.
    function test_setPaused_storageSlots_succeeds() public {
        assertEq(vm.load(address(ethLiquidity), bytes32(uint256(0))), bytes32(0));
        assertEq(vm.load(address(ethLiquidity), bytes32(uint256(1))), bytes32(0));

        _relaySetPaused(true, 100, 1_000_000);

        assertEq(vm.load(address(ethLiquidity), bytes32(uint256(0))), bytes32(uint256(1)));
        assertEq(vm.load(address(ethLiquidity), bytes32(uint256(1))), bytes32(uint256(100)));
    }
}
