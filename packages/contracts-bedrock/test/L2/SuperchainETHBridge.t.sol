// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { MockHelper } from "test/utils/MockHelper.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Unauthorized, ZeroAddress } from "src/libraries/errors/CommonErrors.sol";

// Interfaces
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";
import { ISuperchainETHBridge } from "interfaces/L2/ISuperchainETHBridge.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";

// Contracts
import { MessageExpiryHub } from "src/L1/MessageExpiryHub.sol";

/// @title SuperchainETHBridge_TestInit
/// @notice Reusable test initialization for `SuperchainETHBridge` tests.
abstract contract SuperchainETHBridge_TestInit is CommonTest, MockHelper {
    event SendETH(address indexed from, address indexed to, uint256 amount, uint256 destination);

    event RelayETH(address indexed from, address indexed to, uint256 amount, uint256 source);

    address internal constant ZERO_ADDRESS = address(0);

    /// @notice Test setup.
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();

        {
            // TODO: Remove this block when L2Genesis includes this contract.
            vm.etch(address(superchainETHBridge), vm.getDeployedCode("SuperchainETHBridge.sol:SuperchainETHBridge"));
            vm.etch(address(ethLiquidity), vm.getDeployedCode("ETHLiquidity.sol:ETHLiquidity"));
        }
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

        // Call the `sendETH` function
        vm.prank(_sender);
        bytes32 _returnedMsgHash = superchainETHBridge.sendETH{ value: _amount }(_to, _chainId);

        // Check the message hash was generated correctly
        assertEq(_msgHash, _returnedMsgHash);

        // Check the total supply and balance of `_sender` after the send were updated correctly
        assertEq(_sender.balance, _senderBalanceBefore - _amount);
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

/// @title SuperchainETHBridge_RefundETH_Test
/// @notice Tests the `refundETH` function of the `SuperchainETHBridge` contract.
contract SuperchainETHBridge_RefundETH_Test is SuperchainETHBridge_TestInit {
    event RefundETH(address indexed from, uint256 amount, bytes32 indexed messageHash);

    uint256 internal constant DESTINATION = 902;

    IL2ToL2CrossDomainMessenger internal messenger =
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
    address internal hub = makeAddr("hub");

    /// @notice Sets the messenger's expiry hub.
    function setUp() public override {
        super.setUp();
        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        messenger.setExpiryHub(hub);
    }

    /// @notice Sends ETH from `_from` and returns the send's message nonce and hash.
    function _send(address _from, address _to, uint256 _amount) internal returns (uint256 nonce_, bytes32 hash_) {
        nonce_ = messenger.messageNonce();
        vm.deal(_from, _amount);
        vm.prank(_from);
        hash_ = superchainETHBridge.sendETH{ value: _amount }(_to, DESTINATION);
    }

    /// @notice Marks a message expired, as the hub's fact relayed by the L2CrossDomainMessenger.
    function _expire(bytes32 _messageHash) internal {
        uint256 undeliveredAt = messenger.sentMessageTimestamps(_messageHash) + messenger.MESSAGE_EXPIRY_WINDOW() + 1;
        vm.mockCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(hub)
        );
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messenger.expireMessage(_messageHash, undeliveredAt);
    }

    /// @notice Tests that the ETH of an expired send goes back to the sender, once.
    function testFuzz_refundETH_succeeds(address _from, address _to, uint256 _amount) external {
        vm.assume(_from != address(ethLiquidity) && _from != address(0) && _to != address(0));
        assumeNotPrecompile(_from);
        _amount = bound(_amount, 1, type(uint248).max - 1);
        (uint256 nonce, bytes32 messageHash) = _send(_from, _to, _amount);
        _expire(messageHash);
        uint256 balanceBefore = _from.balance;

        vm.expectCall(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.mint, (_amount)), 1);
        vm.expectEmit(address(superchainETHBridge));
        emit RefundETH(_from, _amount, messageHash);
        superchainETHBridge.refundETH(DESTINATION, nonce, _from, _to, _amount);

        assertEq(_from.balance, balanceBefore + _amount);
        assertTrue(superchainETHBridge.refunded(messageHash));

        vm.expectRevert(ISuperchainETHBridge.AlreadyRefunded.selector);
        superchainETHBridge.refundETH(DESTINATION, nonce, _from, _to, _amount);
    }

    /// @notice Tests that a send whose message has not expired is not refunded.
    function test_refundETH_notExpired_reverts() external {
        (uint256 nonce,) = _send(alice, bob, 1 ether);

        vm.expectRevert(ISuperchainETHBridge.MessageNotExpired.selector);
        superchainETHBridge.refundETH(DESTINATION, nonce, alice, bob, 1 ether);
    }

    /// @notice Tests that a refund whose arguments differ from the expired send's message is
    ///         rejected, so no other amount or recipient can be paid.
    function testFuzz_refundETH_wrongPreimage_reverts(address _from, uint256 _amount) external {
        (uint256 nonce, bytes32 messageHash) = _send(alice, bob, 1 ether);
        _expire(messageHash);
        vm.assume(_from != alice || _amount != 1 ether);

        vm.expectRevert(ISuperchainETHBridge.MessageNotExpired.selector);
        superchainETHBridge.refundETH(DESTINATION, nonce, _from, bob, _amount);
    }

    /// @notice Tests that the refund is a forced send: a sender contract that rejects ETH still
    ///         gets it back, and is not called.
    function test_refundETH_senderRejectsETH_succeeds() external {
        address sender = makeAddr("sender");
        (uint256 nonce, bytes32 messageHash) = _send(sender, bob, 1 ether);
        _expire(messageHash);
        // Code that reverts on any call.
        vm.etch(sender, hex"5f5ffd");

        superchainETHBridge.refundETH(DESTINATION, nonce, sender, bob, 1 ether);

        assertEq(sender.balance, 1 ether);
    }

    /// @notice Tests the whole refund path across chain IDs: the destination exports that the send
    ///         was never relayed, the hub records it and forwards it to the source, the source marks
    ///         the message expired once the window has passed, and the sender is refunded.
    function test_refundETH_endToEnd_succeeds() external {
        uint256 source = block.chainid;
        (uint256 nonce, bytes32 messageHash) = _send(alice, bob, 1 ether);

        // L1: a hub, and the two chains of one cluster.
        MessageExpiryHub l1Hub = new MessageExpiryHub();
        address lockbox = _mockL1Contract("lockbox");
        (, address destinationMessenger) = _mockL1Chain("destination", DESTINATION, lockbox);
        (address sourceConfig, address sourceMessenger) = _mockL1Chain("source", source, lockbox);
        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        messenger.setExpiryHub(address(l1Hub));

        // Destination, after the window: the message was never relayed, so it exports that.
        vm.chainId(DESTINATION);
        vm.warp(block.timestamp + messenger.MESSAGE_EXPIRY_WINDOW() + 1);
        bytes memory toHub = _exportUndelivered(address(l1Hub), source, nonce, messageHash);

        // L1: the withdrawal is relayed to the hub, then forwarded to the source.
        vm.prank(destinationMessenger);
        (bool ok,) = address(l1Hub).call(toHub);
        assertTrue(ok);
        bytes memory toSource =
            abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (messageHash, block.timestamp));
        bytes memory deposit = abi.encodeCall(
            ICrossDomainMessenger.sendMessage, (Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER, toSource, 200_000)
        );
        vm.mockCall(sourceMessenger, deposit, "");
        vm.expectCall(sourceMessenger, deposit);
        l1Hub.forwardUndeliveredMessage(ISystemConfig(sourceConfig), messageHash, block.timestamp, 200_000);

        // Source: the deposit is relayed by the L2CrossDomainMessenger, from the hub.
        vm.chainId(source);
        vm.mockCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l1Hub))
        );
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        (ok,) = address(messenger).call(toSource);
        assertTrue(ok);
        assertTrue(messenger.expiredMessages(messageHash));

        uint256 balanceBefore = alice.balance;
        superchainETHBridge.refundETH(DESTINATION, nonce, alice, bob, 1 ether);
        assertEq(alice.balance, balanceBefore + 1 ether);
    }

    /// @notice Exports the send of 1 ether from alice to bob as undelivered, on the destination, and
    ///         returns the hub call it sends through the L2CrossDomainMessenger.
    function _exportUndelivered(
        address _hub,
        uint256 _source,
        uint256 _nonce,
        bytes32 _messageHash
    )
        internal
        returns (bytes memory toHub_)
    {
        toHub_ = abi.encodeCall(MessageExpiryHub.receiveUndeliveredMessage, (_messageHash, _source, block.timestamp));
        vm.expectCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(ICrossDomainMessenger.sendMessage, (_hub, toHub_, 100_000))
        );
        bytes32 exported = messenger.exportUndeliveredMessage(
            _source,
            _nonce,
            address(superchainETHBridge),
            address(superchainETHBridge),
            abi.encodeCall(ISuperchainETHBridge.relayETH, (alice, bob, 1 ether)),
            100_000
        );
        assertEq(exported, _messageHash);
    }

    /// @notice Creates a labelled address with code, so calls to it can be mocked.
    function _mockL1Contract(string memory _name) internal returns (address addr_) {
        addr_ = makeAddr(_name);
        vm.etch(addr_, hex"01");
    }

    /// @notice Mocks an L1 chain whose SystemConfig, messenger and portal point at each other, in the
    ///         cluster of `_lockbox`, whose messenger relays from the L2ToL2CrossDomainMessenger.
    function _mockL1Chain(
        string memory _name,
        uint256 _chainId,
        address _lockbox
    )
        internal
        returns (address systemConfig_, address messenger_)
    {
        systemConfig_ = _mockL1Contract(string.concat(_name, "Config"));
        messenger_ = _mockL1Contract(string.concat(_name, "Messenger"));
        address portal = _mockL1Contract(string.concat(_name, "Portal"));
        vm.mockCall(systemConfig_, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(messenger_));
        vm.mockCall(systemConfig_, abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(portal));
        vm.mockCall(systemConfig_, abi.encodeCall(ISystemConfig.l2ChainId, ()), abi.encode(_chainId));
        vm.mockCall(messenger_, abi.encodeCall(IL1CrossDomainMessenger.systemConfig, ()), abi.encode(systemConfig_));
        vm.mockCall(
            messenger_,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        );
        vm.mockCall(portal, abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(systemConfig_));
        vm.mockCall(portal, abi.encodeCall(IOptimismPortal2.ethLockbox, ()), abi.encode(_lockbox));
        vm.mockCall(portal, abi.encodeCall(IOptimismPortal2.anchorStateRegistry, ()), abi.encode(makeAddr("asr")));
        vm.mockCall(
            _lockbox,
            abi.encodeCall(IETHLockbox.authorizedPortals, (IOptimismPortal2(payable(portal)))),
            abi.encode(true)
        );
    }
}
