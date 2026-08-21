// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { MockHelper } from "test/utils/MockHelper.sol";

// Libraries
import { Hashing } from "src/libraries/Hashing.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Unauthorized, ZeroAddress } from "src/libraries/errors/CommonErrors.sol";

// Interfaces
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";
import { IMessageExpiryRelay } from "interfaces/L2/IMessageExpiryRelay.sol";
import { ISuperchainETHBridge } from "interfaces/L2/ISuperchainETHBridge.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

/// @title SuperchainETHBridge_TestInit
/// @notice Reusable test initialization for `SuperchainETHBridge` tests.
abstract contract SuperchainETHBridge_TestInit is CommonTest, MockHelper {
    event SendETH(address indexed from, address indexed to, uint256 amount, uint256 destination);

    event RelayETH(address indexed from, address indexed to, uint256 amount, uint256 source);

    event RefundETH(address indexed from, uint256 amount, bytes32 indexed msgHash);

    address internal constant ZERO_ADDRESS = address(0);

    /// @notice Test setup.
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();

        {
            // TODO: Remove this block when L2Genesis includes this contract.
            vm.etch(address(superchainETHBridge), vm.getDeployedCode("SuperchainETHBridge.sol:SuperchainETHBridge"));
            vm.etch(address(ethLiquidity), vm.getDeployedCode("ETHLiquidity.sol:ETHLiquidity"));
            vm.etch(Predeploys.MESSAGE_EXPIRY_RELAY, vm.getDeployedCode("MessageExpiryRelay.sol:MessageExpiryRelay"));
        }
    }

    /// @notice Computes the hash of the message that `sendETH` sends, and makes the messenger report
    ///         it as sent at `_nonce` so that the `MessageExpiryRelay` accepts the recording.
    /// @param _from    Address calling `sendETH`.
    /// @param _to      Recipient of the ETH on the destination chain.
    /// @param _amount  Amount of ETH sent.
    /// @param _chainId Chain ID of the destination chain.
    /// @param _nonce   Nonce the message is sent with.
    /// @return msgHash_ Hash of the message.
    function _mockSentMessage(
        address _from,
        address _to,
        uint256 _amount,
        uint256 _chainId,
        uint256 _nonce
    )
        internal
        returns (bytes32 msgHash_)
    {
        msgHash_ = Hashing.hashL2toL2CrossDomainMessage({
            _destination: _chainId,
            _source: block.chainid,
            _nonce: _nonce,
            _sender: address(superchainETHBridge),
            _target: address(superchainETHBridge),
            _message: abi.encodeCall(superchainETHBridge.relayETH, (_from, _to, _amount))
        });
        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.sentMessages, (_nonce)),
            abi.encode(msgHash_)
        );
    }

    /// @notice Performs a full `sendETH` against a mocked messenger that reports the real message
    ///         hash, so that the resulting pending send is keyed by the same hash the
    ///         `MessageExpiryRelay` recorded.
    /// @param _from    Address calling `sendETH`.
    /// @param _to      Recipient of the ETH on the destination chain.
    /// @param _amount  Amount of ETH to send.
    /// @param _chainId Chain ID of the destination chain.
    /// @return msgHash_ Hash of the message.
    function _sendETH(
        address _from,
        address _to,
        uint256 _amount,
        uint256 _chainId
    )
        internal
        returns (bytes32 msgHash_)
    {
        uint256 nonce = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).messageNonce();
        msgHash_ = _mockSentMessage(_from, _to, _amount, _chainId, nonce);
        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(
                IL2ToL2CrossDomainMessenger.sendMessage,
                (
                    _chainId,
                    address(superchainETHBridge),
                    abi.encodeCall(superchainETHBridge.relayETH, (_from, _to, _amount))
                )
            ),
            abi.encode(msgHash_)
        );

        vm.deal(_from, _from.balance + _amount);
        vm.prank(_from);
        superchainETHBridge.sendETH{ value: _amount }(_to, _chainId);
    }
}

/// @title SuperchainETHBridge_SendETH_Test
/// @notice Tests the `sendETH` function of the `SuperchainETHBridge` contract.
contract SuperchainETHBridge_SendETH_Test is SuperchainETHBridge_TestInit {
    /// @notice Tests the `sendETH` function reverts when the address `_to` is zero.
    function testFuzz_sendETH_zeroAddressTo_reverts(address _sender, uint256 _amount, uint256 _chainId) public {
        // Expect the revert with `ZeroAddress` selector
        vm.expectRevert(ZeroAddress.selector);

        vm.deal(_sender, _amount);
        vm.prank(_sender);
        // Call the `sendETH` function with the zero address as `_to`
        superchainETHBridge.sendETH{ value: _amount }(ZERO_ADDRESS, _chainId);
    }

    /// @notice Tests the `sendETH` function burns the sender ETH, sends the message, and emits the
    ///         `SendETH` event.
    function testFuzz_sendETH_succeeds(
        address _sender,
        address _to,
        uint256 _amount,
        uint256 _chainId,
        bytes32 _msgHash
    )
        external
    {
        // Assume
        vm.assume(_sender != address(ethLiquidity));
        vm.assume(_sender != ZERO_ADDRESS);
        vm.assume(_to != ZERO_ADDRESS);
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(_sender, _amount);

        // Get the total balance of `_sender` before the send to compare later on the assertions
        uint256 _senderBalanceBefore = _sender.balance;

        // Look for the emit of the `SendETH` event
        vm.expectEmit(address(superchainETHBridge));
        emit SendETH(_sender, _to, _amount, _chainId);

        // Expect the call to the `burn` function in the `ETHLiquidity` contract
        vm.expectCall(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.burn, ()), 1);

        // Mock the call over the `sendMessage` function and expect it to be called properly
        bytes memory _message = abi.encodeCall(superchainETHBridge.relayETH, (_sender, _to, _amount));
        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.sendMessage, (_chainId, address(superchainETHBridge), _message)),
            abi.encode(_msgHash)
        );

        // Make the messenger report the message as sent so that the `MessageExpiryRelay` accepts
        // the recording the bridge performs
        uint256 _nonce = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).messageNonce();
        _mockSentMessage(_sender, _to, _amount, _chainId, _nonce);

        // Call the `sendETH` function
        vm.prank(_sender);
        bytes32 _returnedMsgHash = superchainETHBridge.sendETH{ value: _amount }(_to, _chainId);

        // Check the message hash was generated correctly
        assertEq(_msgHash, _returnedMsgHash);

        // Check the total supply and balance of `_sender` after the send were updated correctly
        assertEq(_sender.balance, _senderBalanceBefore - _amount);
    }

    /// @notice Tests the `sendETH` function records the pending send and registers the message with
    ///         the `MessageExpiryRelay` so that it can be refunded if it expires undelivered.
    function testFuzz_sendETH_recordsExpiryHandler_succeeds(
        address _sender,
        address _to,
        uint256 _amount,
        uint256 _chainId
    )
        external
    {
        // Assume
        vm.assume(_sender != address(ethLiquidity));
        vm.assume(_sender != ZERO_ADDRESS);
        vm.assume(_to != ZERO_ADDRESS);
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        uint256 _nonce = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).messageNonce();
        bytes memory _message = abi.encodeCall(superchainETHBridge.relayETH, (_sender, _to, _amount));

        // Expect the bridge to record the message it just sent with the `MessageExpiryRelay`
        vm.expectCall(
            Predeploys.MESSAGE_EXPIRY_RELAY,
            abi.encodeCall(
                IMessageExpiryRelay.recordSentMessage, (_chainId, _nonce, address(superchainETHBridge), _message)
            ),
            1
        );

        // Act
        bytes32 _msgHash = _sendETH(_sender, _to, _amount, _chainId);

        // The pending send is stored under the message hash
        (address _from, uint256 _pendingAmount) = superchainETHBridge.pendingETHSends(_msgHash);
        assertEq(_from, _sender);
        assertEq(_pendingAmount, _amount);

        // And the relay recorded the bridge as the handler for that same message hash
        (address _app,, uint256 _destination) =
            IMessageExpiryRelay(Predeploys.MESSAGE_EXPIRY_RELAY).sentMessageRecords(_msgHash);
        assertEq(_app, address(superchainETHBridge));
        assertEq(_destination, _chainId);
    }
}

/// @title SuperchainETHBridge_RelayETH_Test
/// @notice Tests the `relayETH` function of the `SuperchainETHBridge` contract.
contract SuperchainETHBridge_RelayETH_Test is SuperchainETHBridge_TestInit {
    /// @notice Tests the `relayETH` function reverts when the caller is not the
    ///         `L2ToL2CrossDomainMessenger`.
    function testFuzz_relayETH_notMessenger_reverts(address _caller, address _to, uint256 _amount) public {
        // Ensure the caller is not the messenger
        vm.assume(_caller != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        // Expect the revert with `Unauthorized` selector
        vm.expectRevert(Unauthorized.selector);

        // Call the `relayETH` function with the non-messenger caller
        vm.prank(_caller);
        superchainETHBridge.relayETH(_caller, _to, _amount);
    }

    /// @notice Tests the `relayETH` function reverts when the `crossDomainMessageSender` that sent
    ///         the message is not the same `SuperchainETHBridge`.
    function testFuzz_relayETH_notCrossDomainSender_reverts(
        address _crossDomainMessageSender,
        uint256 _source,
        address _to,
        uint256 _amount
    )
        public
    {
        vm.assume(_crossDomainMessageSender != address(superchainETHBridge));

        // Mock the call over the `crossDomainMessageContext` function setting a wrong sender
        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(_crossDomainMessageSender, _source)
        );

        // Expect the revert with `InvalidCrossDomainSender` selector
        vm.expectRevert(ISuperchainETHBridge.InvalidCrossDomainSender.selector);

        // Call the `relayETH` function with the sender caller
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        superchainETHBridge.relayETH(_crossDomainMessageSender, _to, _amount);
    }

    /// @notice Tests the `relayETH` function relays the proper amount of ETH and emits the
    ///         `RelayETH` event.
    function testFuzz_relayETH_succeeds(address _from, address _to, uint256 _amount, uint256 _source) public {
        // Assume
        vm.assume(_to != ZERO_ADDRESS);
        assumePayable(_to);
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(address(superchainETHBridge), _amount);
        vm.deal(Predeploys.ETH_LIQUIDITY, _amount);
        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(address(superchainETHBridge), _source)
        );

        uint256 _toBalanceBefore = _to.balance;

        // Look for the emit of the `RelayETH` event
        vm.expectEmit(address(superchainETHBridge));
        emit RelayETH(_from, _to, _amount, _source);

        // Expect the call to the `mint` function in the `ETHLiquidity` contract
        vm.expectCall(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.mint, (_amount)), 1);

        // Call the `RelayETH` function with the messenger caller
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        superchainETHBridge.relayETH(_from, _to, _amount);

        assertEq(_to.balance, _toBalanceBefore + _amount);
    }
}

/// @title SuperchainETHBridge_OnMessageExpired_Test
/// @notice Tests the `onMessageExpired` function of the `SuperchainETHBridge` contract.
contract SuperchainETHBridge_OnMessageExpired_Test is SuperchainETHBridge_TestInit {
    /// @notice Chain ID of the destination chain used for the expired sends.
    uint256 internal constant DESTINATION = 902;

    /// @notice Amount of ETH used for the expired sends.
    uint256 internal constant AMOUNT = 1 ether;

    /// @notice Tests the `onMessageExpired` function reverts when the caller is not the
    ///         `MessageExpiryRelay`.
    function testFuzz_onMessageExpired_notRelay_reverts(address _caller, bytes32 _msgHash) public {
        // Ensure the caller is not the relay
        vm.assume(_caller != Predeploys.MESSAGE_EXPIRY_RELAY);

        // Expect the revert with `Unauthorized` selector
        vm.expectRevert(Unauthorized.selector);

        vm.prank(_caller);
        superchainETHBridge.onMessageExpired(_msgHash, DESTINATION, block.timestamp);
    }

    /// @notice Tests the `onMessageExpired` function reverts when there is no pending send for the
    ///         given message hash.
    function testFuzz_onMessageExpired_noPendingSend_reverts(bytes32 _msgHash) public {
        // Expect the revert with `NoPendingSend` selector
        vm.expectRevert(ISuperchainETHBridge.NoPendingSend.selector);

        vm.prank(Predeploys.MESSAGE_EXPIRY_RELAY);
        superchainETHBridge.onMessageExpired(_msgHash, DESTINATION, block.timestamp);
    }

    /// @notice Tests the `onMessageExpired` function refunds the original sender, clears the
    ///         pending send and emits the `RefundETH` event.
    function test_onMessageExpired_succeeds() public {
        bytes32 _msgHash = _sendETH(alice, bob, AMOUNT, DESTINATION);
        uint256 _aliceBalanceBefore = alice.balance;

        // Expect the call to the `mint` function in the `ETHLiquidity` contract
        vm.expectCall(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.mint, (AMOUNT)), 1);

        // Look for the emit of the `RefundETH` event
        vm.expectEmit(address(superchainETHBridge));
        emit RefundETH(alice, AMOUNT, _msgHash);

        vm.prank(Predeploys.MESSAGE_EXPIRY_RELAY);
        superchainETHBridge.onMessageExpired(_msgHash, DESTINATION, block.timestamp);

        // The refund is force sent back to the original sender
        assertEq(alice.balance, _aliceBalanceBefore + AMOUNT);

        // The pending send is consumed
        (address _from, uint256 _amount) = superchainETHBridge.pendingETHSends(_msgHash);
        assertEq(_from, ZERO_ADDRESS);
        assertEq(_amount, 0);
    }

    /// @notice Tests the `onMessageExpired` function refunds the exact amount that was sent.
    /// @param _amount Amount of ETH to send and then refund.
    function testFuzz_onMessageExpired_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 0, type(uint248).max - 1);

        bytes32 _msgHash = _sendETH(alice, bob, _amount, DESTINATION);
        uint256 _aliceBalanceBefore = alice.balance;

        vm.expectEmit(address(superchainETHBridge));
        emit RefundETH(alice, _amount, _msgHash);

        vm.prank(Predeploys.MESSAGE_EXPIRY_RELAY);
        superchainETHBridge.onMessageExpired(_msgHash, DESTINATION, block.timestamp);

        assertEq(alice.balance, _aliceBalanceBefore + _amount);
    }

    /// @notice Tests the `onMessageExpired` function reverts when the same message is expired
    ///         twice, so a send can never be refunded more than once.
    function test_onMessageExpired_alreadyRefunded_reverts() public {
        bytes32 _msgHash = _sendETH(alice, bob, AMOUNT, DESTINATION);

        vm.prank(Predeploys.MESSAGE_EXPIRY_RELAY);
        superchainETHBridge.onMessageExpired(_msgHash, DESTINATION, block.timestamp);

        vm.expectRevert(ISuperchainETHBridge.NoPendingSend.selector);
        vm.prank(Predeploys.MESSAGE_EXPIRY_RELAY);
        superchainETHBridge.onMessageExpired(_msgHash, DESTINATION, block.timestamp);
    }
}
