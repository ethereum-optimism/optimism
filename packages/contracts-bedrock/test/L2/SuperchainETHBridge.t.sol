// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Vm } from "forge-std/Vm.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { MockHelper } from "test/utils/MockHelper.sol";

// Contracts
import { Proxy } from "src/universal/Proxy.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Unauthorized, ZeroAddress } from "src/libraries/errors/CommonErrors.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Features } from "src/libraries/Features.sol";

// Interfaces
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";
import { ISuperchainETHBridge } from "interfaces/L2/ISuperchainETHBridge.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IUndeliveredMessageExporter } from "interfaces/L2/IUndeliveredMessageExporter.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

/// @title SuperchainETHBridge_Sender_Harness
/// @notice A contract that sends ETH through the bridge for its users. If a send expires, it
///         claims the refund and pays its user in the same call.
contract SuperchainETHBridge_Sender_Harness {
    ISuperchainETHBridge internal immutable bridge;

    /// @notice User of each send, by the send's message nonce.
    mapping(uint256 => address) public users;

    constructor(ISuperchainETHBridge _bridge) {
        bridge = _bridge;
    }

    function send(address _to, uint256 _chainId) external payable returns (bytes32 msgHash_) {
        users[IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).messageNonce()] = msg.sender;
        msgHash_ = bridge.sendETH{ value: msg.value }(_to, _chainId);
    }

    /// @notice Claims an expired send's refund and pays it to the send's user.
    function refund(uint256 _destination, uint256 _nonce, address _to, uint256 _amount) external {
        bridge.refundETH(_destination, _nonce, _to, _amount);
        (bool success,) = users[_nonce].call{ value: _amount }("");
        require(success, "SuperchainETHBridge_Sender_Harness: payment failed");
    }
}

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

    /// @notice Nonce of the next pause sync relayed from the L1CrossDomainMessenger.
    uint240 internal pauseSyncNonce;

    /// @notice Pauses or unpauses ETHLiquidity: relays the pause state, read on L1 now, from this
    ///         chain's L1CrossDomainMessenger, as the deposit
    ///         `L1CrossDomainMessenger.syncETHLiquidityPause` makes does.
    /// @param _paused Whether to pause ETHLiquidity.
    function _setPaused(bool _paused) internal {
        address l1Messenger = address(l1CrossDomainMessenger);
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(l1Messenger));
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: pauseSyncNonce++, _version: 1 }),
            l1Messenger,
            Predeploys.ETH_LIQUIDITY,
            0,
            100_000,
            abi.encodeCall(IETHLiquidity.setPaused, (_paused, block.timestamp))
        );
        assertEq(ethLiquidity.paused(), _paused);
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
        vm.assume(_sender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
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

    /// @notice Tests that the L2ToL2CrossDomainMessenger cannot send ETH.
    function testFuzz_sendETH_senderIsMessenger_reverts(address _to, uint256 _amount, uint256 _chainId) external {
        vm.assume(_to != ZERO_ADDRESS);
        vm.deal(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER, _amount);

        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_SenderIsMessenger.selector);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        superchainETHBridge.sendETH{ value: _amount }(_to, _chainId);
    }

    /// @notice Returns a relayable message from chain 901 to this chain, from alice, that calls
    ///         `_target` with `_message`, and has the CrossL2Inbox accept it.
    function _relayable(
        address _target,
        bytes memory _message
    )
        internal
        returns (Identifier memory id_, bytes memory sentMessage_)
    {
        id_ = Identifier(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER, 1, 1, uint64(block.timestamp), 901);
        sentMessage_ = abi.encodePacked(
            abi.encode(
                keccak256("SentMessage(uint256,address,uint256,address,bytes)"), block.chainid, _target, uint256(0)
            ),
            abi.encode(alice, _message)
        );
        vm.mockCall(
            Predeploys.CROSS_L2_INBOX, abi.encodeCall(ICrossL2Inbox.validateMessage, (id_, keccak256(sentMessage_))), ""
        );
    }

    /// @notice Tests that a relayed message that calls sendETH reverts its whole relay, so the
    ///         relayer keeps the ETH it attached.
    function test_sendETH_relayedMessage_reverts() external {
        address relayer = makeAddr("relayer");
        vm.deal(relayer, 1 ether);
        bytes memory message = abi.encodeCall(ISuperchainETHBridge.sendETH, (bob, 901));
        (Identifier memory id, bytes memory sentMessage) = _relayable(address(superchainETHBridge), message);

        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_SenderIsMessenger.selector);
        vm.prank(relayer);
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).relayMessage{ value: 1 ether }(
            id, sentMessage
        );

        assertEq(relayer.balance, 1 ether);
        assertEq(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER.balance, 0);
    }

    /// @notice Tests that a message from L1 can send ETH.
    function test_sendETH_fromL1Message_succeeds() external {
        address l1Messenger = address(l1CrossDomainMessenger);
        uint256 nonce = Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 });
        bytes memory message = abi.encodeCall(ISuperchainETHBridge.sendETH, (bob, 902));
        vm.deal(AddressAliasHelper.applyL1ToL2Alias(l1Messenger), 1 ether);

        vm.expectEmit(address(superchainETHBridge));
        emit SendETH(Predeploys.L2_CROSS_DOMAIN_MESSENGER, bob, 1 ether, 902);
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(l1Messenger));
        l2CrossDomainMessenger.relayMessage{ value: 1 ether }(
            nonce, alice, address(superchainETHBridge), 1 ether, 200_000, message
        );

        assertEq(address(l2CrossDomainMessenger).balance, 0);
    }

    /// @notice Tests that `sendETH` reverts as a whole while ETHLiquidity is paused: the sender
    ///         keeps the ETH and no message is sent. Once unpaused, the same send goes through.
    function test_sendETH_whilePaused_reverts() external {
        IL2ToL2CrossDomainMessenger messenger = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        uint256 destination = 902;
        vm.deal(alice, 1 ether);
        uint256 liquidityBefore = address(ethLiquidity).balance;
        uint256 nonceBefore = messenger.messageNonce();
        _setPaused(true);

        vm.prank(alice);
        vm.expectRevert(IETHLiquidity.ETHLiquidity_Paused.selector);
        superchainETHBridge.sendETH{ value: 1 ether }(bob, destination);

        assertEq(alice.balance, 1 ether);
        assertEq(address(ethLiquidity).balance, liquidityBefore);
        assertEq(messenger.messageNonce(), nonceBefore);

        _setPaused(false);
        vm.prank(alice);
        bytes32 messageHash = superchainETHBridge.sendETH{ value: 1 ether }(bob, destination);

        assertEq(alice.balance, 0);
        assertEq(address(ethLiquidity).balance, liquidityBefore + 1 ether);
        assertEq(messenger.messageNonce(), nonceBefore + 1);
        assertGt(messenger.sentMessageTimestamps(messageHash), 0);
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

    /// @notice Tests that relaying a send while ETHLiquidity is paused reverts the whole
    ///         `relayMessage`, so the message is not marked relayed and can be relayed once
    ///         ETHLiquidity is unpaused.
    function test_relayETH_whilePaused_reverts() external {
        IL2ToL2CrossDomainMessenger messenger = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        uint256 source = 901;
        uint256 nonce = 7;
        bytes memory message = abi.encodeCall(ISuperchainETHBridge.relayETH, (alice, bob, 1 ether));
        bytes memory sentMessage = abi.encodePacked(
            abi.encode(
                keccak256("SentMessage(uint256,address,uint256,address,bytes)"),
                block.chainid,
                address(superchainETHBridge),
                nonce
            ),
            abi.encode(address(superchainETHBridge), message)
        );
        Identifier memory id =
            Identifier(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER, 1, 1, uint64(block.timestamp), source);
        bytes32 messageHash = Hashing.hashL2toL2CrossDomainMessage({
            _destination: block.chainid,
            _source: source,
            _nonce: nonce,
            _sender: address(superchainETHBridge),
            _target: address(superchainETHBridge),
            _message: message
        });
        vm.mockCall(
            Predeploys.CROSS_L2_INBOX, abi.encodeCall(ICrossL2Inbox.validateMessage, (id, keccak256(sentMessage))), ""
        );
        vm.deal(address(ethLiquidity), 1 ether);
        uint256 bobBalanceBefore = bob.balance;
        _setPaused(true);

        vm.expectRevert(IETHLiquidity.ETHLiquidity_Paused.selector);
        messenger.relayMessage(id, sentMessage);

        assertFalse(messenger.successfulMessages(messageHash));
        assertEq(bob.balance, bobBalanceBefore);
        assertEq(address(ethLiquidity).balance, 1 ether);

        // Once unpaused, anyone can relay the same message.
        _setPaused(false);
        vm.expectEmit(address(superchainETHBridge));
        emit RelayETH(alice, bob, 1 ether, source);
        messenger.relayMessage(id, sentMessage);

        assertTrue(messenger.successfulMessages(messageHash));
        assertEq(bob.balance, bobBalanceBefore + 1 ether);
        assertEq(address(ethLiquidity).balance, 0);
    }
}

/// @title SuperchainETHBridge_RefundETH_Test
/// @notice Tests the `refundETH` function of the `SuperchainETHBridge` contract.
contract SuperchainETHBridge_RefundETH_Test is SuperchainETHBridge_TestInit {
    event RefundETH(address indexed from, uint256 amount, bytes32 indexed messageHash);

    uint256 internal constant DESTINATION = 902;

    IL2ToL2CrossDomainMessenger internal messenger =
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

    /// @notice Sends ETH from `_from` and returns the send's message nonce and hash.
    function _send(address _from, address _to, uint256 _amount) internal returns (uint256 nonce_, bytes32 hash_) {
        nonce_ = messenger.messageNonce();
        vm.deal(_from, _amount);
        vm.prank(_from);
        hash_ = superchainETHBridge.sendETH{ value: _amount }(_to, DESTINATION);
    }

    /// @notice Marks a message expired, as word from the L1CrossDomainMessenger relayed by the
    ///         L2CrossDomainMessenger.
    function _expire(bytes32 _messageHash) internal {
        uint256 undeliveredAt = messenger.sentMessageTimestamps(_messageHash) + messenger.expiryPeriod() + 1;
        vm.mockCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l2CrossDomainMessenger.otherMessenger()))
        );
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messenger.expireMessage(_messageHash, undeliveredAt);
    }

    /// @notice Tests that the ETH of an expired send goes back to the sender, once.
    function testFuzz_refundETH_succeeds(address _from, address _to, uint256 _amount) external {
        vm.assume(_from != address(ethLiquidity) && _from != address(0) && _to != address(0));
        vm.assume(_from != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        assumeNotPrecompile(_from);
        _amount = bound(_amount, 1, type(uint248).max - 1);
        (uint256 nonce, bytes32 messageHash) = _send(_from, _to, _amount);
        _expire(messageHash);
        uint256 balanceBefore = _from.balance;

        vm.expectCall(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.mint, (_amount)), 1);
        vm.expectEmit(address(superchainETHBridge));
        emit RefundETH(_from, _amount, messageHash);
        vm.prank(_from);
        superchainETHBridge.refundETH(DESTINATION, nonce, _to, _amount);

        assertEq(_from.balance, balanceBefore + _amount);
        assertTrue(superchainETHBridge.refunded(messageHash));

        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_AlreadyRefunded.selector);
        vm.prank(_from);
        superchainETHBridge.refundETH(DESTINATION, nonce, _to, _amount);
    }

    /// @notice Tests that a refund reverts while ETHLiquidity is paused, without marking the send
    ///         refunded, and goes through once it is unpaused. The message can still expire while
    ///         the pause lasts, since expiry does not touch ETHLiquidity.
    function test_refundETH_whilePaused_reverts() external {
        (uint256 nonce, bytes32 messageHash) = _send(alice, bob, 1 ether);
        _setPaused(true);
        _expire(messageHash);
        assertTrue(messenger.expiredMessages(messageHash));

        vm.expectRevert(IETHLiquidity.ETHLiquidity_Paused.selector);
        vm.prank(alice);
        superchainETHBridge.refundETH(DESTINATION, nonce, bob, 1 ether);

        assertFalse(superchainETHBridge.refunded(messageHash));
        assertEq(alice.balance, 0);

        _setPaused(false);
        vm.expectEmit(address(superchainETHBridge));
        emit RefundETH(alice, 1 ether, messageHash);
        vm.prank(alice);
        superchainETHBridge.refundETH(DESTINATION, nonce, bob, 1 ether);

        assertTrue(superchainETHBridge.refunded(messageHash));
        assertEq(alice.balance, 1 ether);
    }

    /// @notice Tests that an expired send of no ETH is still marked refunded, once, and reported.
    function test_refundETH_zeroAmount_succeeds() external {
        (uint256 nonce, bytes32 messageHash) = _send(alice, bob, 0);
        _expire(messageHash);

        vm.expectEmit(address(superchainETHBridge));
        emit RefundETH(alice, 0, messageHash);
        vm.prank(alice);
        superchainETHBridge.refundETH(DESTINATION, nonce, bob, 0);

        assertTrue(superchainETHBridge.refunded(messageHash));
        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_AlreadyRefunded.selector);
        vm.prank(alice);
        superchainETHBridge.refundETH(DESTINATION, nonce, bob, 0);
    }

    /// @notice Tests that the refund record lives in the `refunded` mapping at storage slot 0, where
    ///         the storage layout snapshot puts it, so that a declaration added before it cannot
    ///         move the records an upgrade must keep.
    function test_refundETH_refundedAtSlotZero_succeeds() external {
        (uint256 nonce, bytes32 messageHash) = _send(alice, bob, 1 ether);
        _expire(messageHash);
        bytes32 slot = keccak256(abi.encode(messageHash, uint256(0)));
        assertEq(vm.load(address(superchainETHBridge), slot), bytes32(0));

        vm.prank(alice);
        superchainETHBridge.refundETH(DESTINATION, nonce, bob, 1 ether);

        assertEq(vm.load(address(superchainETHBridge), slot), bytes32(uint256(1)));
    }

    /// @notice Tests that a send whose message has not expired is not refunded.
    function test_refundETH_notExpired_reverts() external {
        (uint256 nonce,) = _send(alice, bob, 1 ether);

        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_MessageNotExpired.selector);
        vm.prank(alice);
        superchainETHBridge.refundETH(DESTINATION, nonce, bob, 1 ether);
    }

    /// @notice Tests that a refund whose arguments differ from the expired send's message is
    ///         rejected, so no other amount or recipient can be paid.
    function testFuzz_refundETH_wrongPreimage_reverts(
        uint256 _destination,
        uint256 _nonce,
        address _to,
        uint256 _amount
    )
        external
    {
        (uint256 nonce, bytes32 messageHash) = _send(alice, bob, 1 ether);
        _expire(messageHash);
        vm.assume(
            keccak256(abi.encode(_destination, _nonce, _to, _amount))
                != keccak256(abi.encode(DESTINATION, nonce, bob, 1 ether))
        );

        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_MessageNotExpired.selector);
        vm.prank(alice);
        superchainETHBridge.refundETH(_destination, _nonce, _to, _amount);
    }

    /// @notice Tests that only the sender can claim the refund.
    function testFuzz_refundETH_notSender_reverts(address _caller) external {
        vm.assume(_caller != alice);
        (uint256 nonce, bytes32 messageHash) = _send(alice, bob, 1 ether);
        _expire(messageHash);

        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_MessageNotExpired.selector);
        vm.prank(_caller);
        superchainETHBridge.refundETH(DESTINATION, nonce, bob, 1 ether);
        assertFalse(superchainETHBridge.refunded(messageHash));
    }

    /// @notice Tests that naming this chain as the destination refunds nothing: no message can be
    ///         sent to the chain it is sent from.
    function test_refundETH_destinationIsThisChain_reverts() external {
        (uint256 nonce, bytes32 messageHash) = _send(alice, bob, 1 ether);
        _expire(messageHash);

        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_MessageNotExpired.selector);
        vm.prank(alice);
        superchainETHBridge.refundETH(block.chainid, nonce, bob, 1 ether);
    }

    /// @notice Tests that an expired message shaped like a send but not sent by the bridge refunds
    ///         nothing: the bridge rebuilds the hash with itself as the sender.
    function test_refundETH_messageNotFromBridge_reverts() external {
        uint256 nonce = messenger.messageNonce();
        vm.prank(alice);
        bytes32 messageHash = messenger.sendMessage(
            DESTINATION,
            address(superchainETHBridge),
            abi.encodeCall(ISuperchainETHBridge.relayETH, (alice, bob, 1 ether))
        );
        _expire(messageHash);

        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_MessageNotExpired.selector);
        vm.prank(alice);
        superchainETHBridge.refundETH(DESTINATION, nonce, bob, 1 ether);
    }

    /// @notice Tests that a contract that sends for its users can claim a refund and pay its user
    ///         in the same call.
    function test_refundETH_contractSender_succeeds() external {
        SuperchainETHBridge_Sender_Harness sender = new SuperchainETHBridge_Sender_Harness(superchainETHBridge);
        uint256 nonce = messenger.messageNonce();
        vm.deal(alice, 1 ether);
        vm.prank(alice);
        bytes32 messageHash = sender.send{ value: 1 ether }(bob, DESTINATION);
        _expire(messageHash);

        vm.prank(bob);
        sender.refund(DESTINATION, nonce, bob, 1 ether);

        assertEq(alice.balance, 1 ether);
        assertEq(address(sender).balance, 0);
        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_AlreadyRefunded.selector);
        sender.refund(DESTINATION, nonce, bob, 1 ether);
    }

    /// @notice Tests that the refund is a forced send: a sender contract that rejects ETH still
    ///         gets it back, and is not called.
    function test_refundETH_senderRejectsETH_succeeds() external {
        address sender = makeAddr("sender");
        (uint256 nonce, bytes32 messageHash) = _send(sender, bob, 1 ether);
        _expire(messageHash);
        // Code that reverts on any call.
        vm.etch(sender, hex"5f5ffd");

        vm.prank(sender);
        superchainETHBridge.refundETH(DESTINATION, nonce, bob, 1 ether);

        assertEq(sender.balance, 1 ether);
    }
}

/// @title SuperchainETHBridge_Integration_Test
/// @notice Runs an expired send's refund through the real contracts: the destination's
///         UndeliveredMessageExporter exports it, the destination's L1CrossDomainMessenger relays
///         the withdrawal into this chain's L1CrossDomainMessenger, and its deposit expires the
///         message here.
contract SuperchainETHBridge_Integration_Test is SuperchainETHBridge_TestInit {
    event MessageExpired(bytes32 indexed messageHash, uint256 undeliveredAt);

    uint256 internal constant DESTINATION = 902;

    IL2ToL2CrossDomainMessenger internal messenger =
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

    /// @notice Sends 1 ether from alice to bob on the destination and lets the expiry period pass.
    function _sendAndWait() internal returns (uint256 nonce_, bytes32 hash_) {
        nonce_ = messenger.messageNonce();
        vm.deal(alice, 1 ether);
        vm.prank(alice);
        hash_ = superchainETHBridge.sendETH{ value: 1 ether }(bob, DESTINATION);
        vm.warp(block.timestamp + messenger.expiryPeriod() + 1);
    }

    /// @notice Relays an L1 message from this chain's L1CrossDomainMessenger to the
    ///         L2ToL2CrossDomainMessenger, as the deposit does.
    function _relayToL2(uint256 _nonce, bytes memory _message, uint256 _gas) internal {
        address l1Messenger = address(l1CrossDomainMessenger);
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(l1Messenger));
        l2CrossDomainMessenger.relayMessage{ gas: _gas }(
            _nonce, l1Messenger, Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER, 0, 100_000, _message
        );
    }

    /// @notice Syncs this chain's L1 pause through the real L1CrossDomainMessenger and relays the
    ///         message it sent to ETHLiquidity, as its deposit does.
    function _syncPause() internal {
        if (!systemConfig.isFeatureEnabled(Features.INTEROP)) {
            vm.prank(proxyAdminOwner);
            systemConfig.setFeature(Features.INTEROP, true);
        }
        vm.recordLogs();
        vm.prank(alice);
        l1CrossDomainMessenger.syncETHLiquidityPause();

        Vm.Log[] memory logs = vm.getRecordedLogs();
        for (uint256 i; i < logs.length; i++) {
            if (logs[i].emitter != address(l1CrossDomainMessenger) || logs[i].topics[0] != SentMessage.selector) {
                continue;
            }
            assertEq(logs[i].topics[1], bytes32(uint256(uint160(Predeploys.ETH_LIQUIDITY))));
            (address sender, bytes memory message, uint256 nonce, uint256 gasLimit) =
                abi.decode(logs[i].data, (address, bytes, uint256, uint256));
            vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)));
            l2CrossDomainMessenger.relayMessage(nonce, sender, Predeploys.ETH_LIQUIDITY, 0, gasLimit, message);
            return;
        }
        revert("SuperchainETHBridge_Integration_Test: no SentMessage from the L1CrossDomainMessenger");
    }

    /// @notice Tests the whole pause path: the guardian pauses the chain on L1, anyone syncs it to
    ///         L2, and sends revert; it unpauses, anyone syncs again, and sends go through.
    function test_sendETH_l1PauseSynced_reverts() external {
        vm.deal(alice, 1 ether);

        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(0));
        _syncPause();
        assertTrue(ethLiquidity.paused());
        assertEq(ethLiquidity.pausedL1Timestamp(), block.timestamp);

        vm.expectRevert(IETHLiquidity.ETHLiquidity_Paused.selector);
        vm.prank(alice);
        superchainETHBridge.sendETH{ value: 1 ether }(bob, DESTINATION);

        vm.prank(superchainConfig.guardian());
        superchainConfig.unpause(address(0));
        vm.warp(block.timestamp + 12);
        _syncPause();
        assertFalse(ethLiquidity.paused());

        vm.prank(alice);
        superchainETHBridge.sendETH{ value: 1 ether }(bob, DESTINATION);
        assertEq(alice.balance, 0);
    }

    /// @notice Tests the whole path: export on the destination (as a deposit), the destination's
    ///         L1CrossDomainMessenger relaying into this chain's, the deposit expiring the message,
    ///         and the refund.
    function test_refundETH_endToEnd_succeeds() external {
        (uint256 nonce, bytes32 messageHash) = _sendAndWait();
        bytes memory relayETH = abi.encodeCall(ISuperchainETHBridge.relayETH, (alice, bob, 1 ether));
        bytes memory relayUndelivered =
            abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (messageHash, block.timestamp));

        // On the destination, export from a deposit: no sequencer can keep it out.
        uint256 source = block.chainid;
        vm.chainId(DESTINATION);
        vm.expectEmit(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        emit SentMessage(
            address(l1CrossDomainMessenger),
            Predeploys.UNDELIVERED_MESSAGE_EXPORTER,
            relayUndelivered,
            l2CrossDomainMessenger.messageNonce(),
            1_000_000
        );
        address depositor = AddressAliasHelper.applyL1ToL2Alias(alice);
        vm.prank(depositor, depositor);
        IUndeliveredMessageExporter(Predeploys.UNDELIVERED_MESSAGE_EXPORTER)
            .exportUndeliveredMessage(
                address(l1CrossDomainMessenger),
                source,
                nonce,
                address(superchainETHBridge),
                address(superchainETHBridge),
                relayETH,
                1_000_000
            );
        vm.chainId(source);

        // On L1, the destination's messenger relays the withdrawal into this chain's.
        IL1CrossDomainMessenger destinationMessenger = _destinationL1Messenger();
        uint256 depositNonce = l1CrossDomainMessenger.messageNonce();
        bytes memory expire = abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (messageHash, block.timestamp));
        // This chain's portal deposits expireMessage for this chain's L2ToL2CrossDomainMessenger, sent by
        // this chain's L1CrossDomainMessenger, with the gas it reserves for it.
        vm.expectCall(
            address(optimismPortal2),
            abi.encodeCall(
                IOptimismPortal2.depositTransaction,
                (
                    Predeploys.L2_CROSS_DOMAIN_MESSENGER,
                    0,
                    l1CrossDomainMessenger.baseGas(expire, 100_000),
                    false,
                    Encoding.encodeCrossDomainMessage(
                        depositNonce,
                        address(l1CrossDomainMessenger),
                        Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
                        0,
                        100_000,
                        expire
                    )
                )
            )
        );
        vm.prank(address(destinationMessenger.portal()));
        destinationMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            Predeploys.UNDELIVERED_MESSAGE_EXPORTER,
            address(l1CrossDomainMessenger),
            0,
            1_000_000,
            relayUndelivered
        );
        assertEq(l1CrossDomainMessenger.messageNonce(), depositNonce + 1);

        // Here, the deposit expires the message, and the sender is refunded.
        vm.expectEmit(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        emit MessageExpired(messageHash, block.timestamp);
        _relayToL2(depositNonce, expire, 1_000_000);
        assertTrue(messenger.expiredMessages(messageHash));

        vm.prank(alice);
        superchainETHBridge.refundETH(DESTINATION, nonce, bob, 1 ether);
        assertEq(alice.balance, 1 ether);
    }

    /// @notice Tests that EXPIRE_MESSAGE_GAS_LIMIT covers expireMessage on cold storage, as in the
    ///         deposit's own transaction.
    function test_refundETH_expireGasLimit_succeeds() external {
        (, bytes32 messageHash) = _sendAndWait();
        // Read the implementation first: reading it warms the proxy's slot.
        address implementation = EIP1967Helper.getImplementation(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        vm.cool(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        vm.cool(implementation);
        vm.mockCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l1CrossDomainMessenger))
        );

        // The real xDomainMessageSender lookup costs under 10k more than the mock, so 90k here
        // means expireMessage fits in the 100k the L1CrossDomainMessenger asks for.
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messenger.expireMessage{ gas: 90_000 }(messageHash, block.timestamp);
        assertTrue(messenger.expiredMessages(messageHash));
    }

    /// @notice Tests that a deposit that runs out of gas lands in the L2CrossDomainMessenger's
    ///         failed messages, and anyone can replay it to expire the message.
    function test_refundETH_expireReplay_succeeds() external {
        (uint256 nonce, bytes32 messageHash) = _sendAndWait();
        bytes memory expire = abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (messageHash, block.timestamp));
        uint256 depositNonce = Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 });

        _relayToL2(depositNonce, expire, 150_000);
        bytes32 versionedHash = Hashing.hashCrossDomainMessageV1(
            depositNonce,
            address(l1CrossDomainMessenger),
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            0,
            100_000,
            expire
        );
        assertTrue(l2CrossDomainMessenger.failedMessages(versionedHash));
        assertFalse(messenger.expiredMessages(messageHash));

        vm.prank(bob);
        l2CrossDomainMessenger.relayMessage(
            depositNonce,
            address(l1CrossDomainMessenger),
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            0,
            100_000,
            expire
        );
        assertTrue(messenger.expiredMessages(messageHash));

        vm.prank(alice);
        superchainETHBridge.refundETH(DESTINATION, nonce, bob, 1 ether);
        assertEq(alice.balance, 1 ether);
    }

    /// @notice Deploys a real L1CrossDomainMessenger for the destination chain. Its portal and
    ///         SystemConfig are stand-ins that bind to it, and the portal is authorized by this
    ///         chain's lockbox through the real authorizePortal.
    function _destinationL1Messenger() internal returns (IL1CrossDomainMessenger messenger_) {
        address portal = makeAddr("destinationPortal");
        address config = makeAddr("destinationConfig");
        vm.etch(portal, hex"01");
        vm.etch(config, hex"01");

        Proxy proxy = new Proxy(address(proxyAdmin));
        address implementation = vm.deployCode("L1CrossDomainMessenger.sol:L1CrossDomainMessenger");
        vm.prank(address(proxyAdmin));
        proxy.upgradeToAndCall(
            implementation,
            abi.encodeCall(
                IL1CrossDomainMessenger.initialize, (ISystemConfig(config), IOptimismPortal2(payable(portal)))
            )
        );
        messenger_ = IL1CrossDomainMessenger(address(proxy));

        vm.mockCall(portal, abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(config));
        vm.mockCall(
            portal, abi.encodeCall(IOptimismPortal2.l2Sender, ()), abi.encode(Predeploys.L2_CROSS_DOMAIN_MESSENGER)
        );
        vm.mockCall(portal, abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(proxyAdminOwner));
        vm.mockCall(config, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(address(proxy)));
        vm.mockCall(config, abi.encodeCall(ISystemConfig.superchainConfig, ()), abi.encode(superchainConfig));
        vm.mockCall(config, abi.encodeCall(ISystemConfig.paused, ()), abi.encode(false));

        IETHLockbox lockbox = optimismPortal2.ethLockbox();
        vm.prank(proxyAdminOwner);
        lockbox.authorizePortal(IOptimismPortal2(payable(portal)));

        // This chain runs interop, so its L1CrossDomainMessenger accepts the word.
        if (!systemConfig.isFeatureEnabled(Features.INTEROP)) {
            vm.prank(proxyAdminOwner);
            systemConfig.setFeature(Features.INTEROP, true);
        }
    }
}
