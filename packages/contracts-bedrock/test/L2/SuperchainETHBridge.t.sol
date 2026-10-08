// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
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
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IUndeliveredMessageExporter } from "interfaces/L2/IUndeliveredMessageExporter.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

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

        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_AlreadyRefunded.selector);
        superchainETHBridge.refundETH(DESTINATION, nonce, _from, _to, _amount);
    }

    /// @notice Tests that an expired send of no ETH is still marked refunded, once, and reported.
    function test_refundETH_zeroAmount_succeeds() external {
        (uint256 nonce, bytes32 messageHash) = _send(alice, bob, 0);
        _expire(messageHash);

        vm.expectEmit(address(superchainETHBridge));
        emit RefundETH(alice, 0, messageHash);
        superchainETHBridge.refundETH(DESTINATION, nonce, alice, bob, 0);

        assertTrue(superchainETHBridge.refunded(messageHash));
        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_AlreadyRefunded.selector);
        superchainETHBridge.refundETH(DESTINATION, nonce, alice, bob, 0);
    }

    /// @notice Tests that a send whose message has not expired is not refunded.
    function test_refundETH_notExpired_reverts() external {
        (uint256 nonce,) = _send(alice, bob, 1 ether);

        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_MessageNotExpired.selector);
        superchainETHBridge.refundETH(DESTINATION, nonce, alice, bob, 1 ether);
    }

    /// @notice Tests that a refund whose arguments differ from the expired send's message is
    ///         rejected, so no other amount or recipient can be paid.
    function testFuzz_refundETH_wrongPreimage_reverts(
        uint256 _destination,
        uint256 _nonce,
        address _from,
        address _to,
        uint256 _amount
    )
        external
    {
        (uint256 nonce, bytes32 messageHash) = _send(alice, bob, 1 ether);
        _expire(messageHash);
        vm.assume(
            keccak256(abi.encode(_destination, _nonce, _from, _to, _amount))
                != keccak256(abi.encode(DESTINATION, nonce, alice, bob, 1 ether))
        );

        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_MessageNotExpired.selector);
        superchainETHBridge.refundETH(_destination, _nonce, _from, _to, _amount);
    }

    /// @notice Tests that naming this chain as the destination refunds nothing: no message can be
    ///         sent to the chain it is sent from.
    function test_refundETH_destinationIsThisChain_reverts() external {
        (uint256 nonce, bytes32 messageHash) = _send(alice, bob, 1 ether);
        _expire(messageHash);

        vm.expectRevert(ISuperchainETHBridge.SuperchainETHBridge_MessageNotExpired.selector);
        superchainETHBridge.refundETH(block.chainid, nonce, alice, bob, 1 ether);
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
        superchainETHBridge.refundETH(DESTINATION, nonce, alice, bob, 1 ether);
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

        superchainETHBridge.refundETH(DESTINATION, nonce, alice, bob, 1 ether);
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

        superchainETHBridge.refundETH(DESTINATION, nonce, alice, bob, 1 ether);
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
