// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { MockHelper } from "test/utils/MockHelper.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { IMessageExpiryHandler } from "interfaces/L2/IMessageExpiryHandler.sol";
import { IMessageExpiryRelay } from "interfaces/L2/IMessageExpiryRelay.sol";
import { IMessageExpiryHub } from "interfaces/L1/IMessageExpiryHub.sol";

/// @title MessageExpiryRelay_App_Harness
/// @notice Minimal application implementing `IMessageExpiryHandler` so that the callback performed
///         by `receiveExpiry` can be observed, and made to revert, from tests.
contract MessageExpiryRelay_App_Harness is IMessageExpiryHandler {
    /// @notice Thrown by `onMessageExpired` when the harness is configured to revert.
    error MessageExpiryRelay_App_Harness_Reverted();

    /// @notice Whether `onMessageExpired` should revert.
    bool public shouldRevert;

    /// @notice Number of times `onMessageExpired` has been called.
    uint256 public callCount;

    /// @notice Message hash of the last `onMessageExpired` call.
    bytes32 public lastMsgHash;

    /// @notice Attestor chain ID of the last `onMessageExpired` call.
    uint256 public lastAttestorChainId;

    /// @notice Attestation timestamp of the last `onMessageExpired` call.
    uint256 public lastAttestedAt;

    /// @notice Configures whether `onMessageExpired` reverts.
    /// @param _shouldRevert Whether the callback should revert.
    function setShouldRevert(bool _shouldRevert) external {
        shouldRevert = _shouldRevert;
    }

    /// @notice Records the expiry callback, or reverts when configured to do so.
    /// @param _msgHash         Hash of the expired message.
    /// @param _attestorChainId Chain ID of the chain that attested non-delivery.
    /// @param _attestedAt      Timestamp of the attestation.
    function onMessageExpired(bytes32 _msgHash, uint256 _attestorChainId, uint256 _attestedAt) external {
        if (shouldRevert) revert MessageExpiryRelay_App_Harness_Reverted();
        callCount++;
        lastMsgHash = _msgHash;
        lastAttestorChainId = _attestorChainId;
        lastAttestedAt = _attestedAt;
    }
}

/// @title MessageExpiryRelay_TestInit
/// @notice Reusable test initialization for `MessageExpiryRelay` tests.
abstract contract MessageExpiryRelay_TestInit is CommonTest, MockHelper {
    event SentMessageRecorded(bytes32 indexed msgHash, address indexed app, uint256 destination, uint256 recordedAt);

    event UndeliveredMessageAttested(bytes32 indexed msgHash, uint256 indexed sourceChainId, uint256 attestedAt);

    event ExpiredMessageRelayed(
        bytes32 indexed msgHash, address indexed app, uint256 attestorChainId, uint256 attestedAt
    );

    /// @notice Storage slot of the `hub` variable. Slot 0 is taken by `Initializable`.
    bytes32 internal constant HUB_SLOT = bytes32(uint256(2));

    /// @notice Storage slot of the `expiryWindow` variable.
    bytes32 internal constant EXPIRY_WINDOW_SLOT = bytes32(uint256(3));

    /// @notice Expiry window used by the suites that initialize the relay.
    uint256 internal constant EXPIRY_WINDOW = 7 days;

    /// @notice Destination chain ID used for recorded messages.
    uint256 internal constant DESTINATION = 902;

    /// @notice The `MessageExpiryRelay` predeploy under test.
    IMessageExpiryRelay internal messageExpiryRelay = IMessageExpiryRelay(Predeploys.MESSAGE_EXPIRY_RELAY);

    /// @notice Application that records messages and receives expiry callbacks.
    MessageExpiryRelay_App_Harness internal app;

    /// @notice Address of the `MessageExpiryHub` on L1.
    address internal hub;

    /// @notice Test setup.
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();

        {
            // TODO: Remove this block when L2Genesis includes this contract.
            vm.etch(Predeploys.MESSAGE_EXPIRY_RELAY, vm.getDeployedCode("MessageExpiryRelay.sol:MessageExpiryRelay"));
        }

        // The etched account never ran the constructor, so it is initializable. Wipe the
        // initialization state and the configuration slots so that every suite starts from a
        // deterministic, uninitialized contract regardless of what genesis wrote to this address.
        vm.store(Predeploys.MESSAGE_EXPIRY_RELAY, bytes32(0), bytes32(0));
        vm.store(Predeploys.MESSAGE_EXPIRY_RELAY, HUB_SLOT, bytes32(0));
        vm.store(Predeploys.MESSAGE_EXPIRY_RELAY, EXPIRY_WINDOW_SLOT, bytes32(0));

        // Point the EIP-1967 admin slot at the ProxyAdmin so that `initialize` can be authorized by
        // pranking either the ProxyAdmin or its owner.
        vm.store(
            Predeploys.MESSAGE_EXPIRY_RELAY,
            Constants.PROXY_OWNER_ADDRESS,
            bytes32(uint256(uint160(address(proxyAdmin))))
        );

        app = new MessageExpiryRelay_App_Harness();
        hub = makeAddr("hub");
    }

    /// @notice Initializes the relay as the ProxyAdmin owner.
    /// @param _hub          Address of the hub on L1.
    /// @param _expiryWindow Expiry window in seconds.
    function _initializeRelay(address _hub, uint256 _expiryWindow) internal {
        vm.prank(proxyAdminOwner);
        messageExpiryRelay.initialize(_hub, _expiryWindow);
    }

    /// @notice Computes the `L2ToL2CrossDomainMessenger` hash of a message sent from this chain.
    /// @param _destination Chain ID of the destination chain.
    /// @param _nonce       Nonce of the sent message.
    /// @param _sender      Sender of the message.
    /// @param _target      Target of the message on the destination chain.
    /// @param _message     Message payload.
    /// @return Hash of the message.
    function _messageHash(
        uint256 _destination,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes memory _message
    )
        internal
        view
        returns (bytes32)
    {
        return Hashing.hashL2toL2CrossDomainMessage({
            _destination: _destination,
            _source: block.chainid,
            _nonce: _nonce,
            _sender: _sender,
            _target: _target,
            _message: _message
        });
    }

    /// @notice Makes the messenger report `_sender`'s message at `_nonce` as sent, then records it.
    /// @param _sender      Application recording the message.
    /// @param _destination Chain ID of the destination chain.
    /// @param _nonce       Nonce of the sent message.
    /// @param _target      Target of the message on the destination chain.
    /// @param _message     Message payload.
    /// @return msgHash_ Hash of the recorded message.
    function _recordSentMessage(
        address _sender,
        uint256 _destination,
        uint256 _nonce,
        address _target,
        bytes memory _message
    )
        internal
        returns (bytes32 msgHash_)
    {
        msgHash_ = _messageHash(_destination, _nonce, _sender, _target, _message);
        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.sentMessages, (_nonce)),
            abi.encode(msgHash_)
        );
        vm.prank(_sender);
        messageExpiryRelay.recordSentMessage(_destination, _nonce, _target, _message);
    }

    /// @notice Mocks the `L2CrossDomainMessenger`'s cross domain sender.
    /// @param _sender Cross domain sender to report.
    function _mockXDomainMessageSender(address _sender) internal {
        vm.mockCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(_sender)
        );
    }
}

/// @title MessageExpiryRelay_Initialize_Test
/// @notice Tests the `initialize` function of the `MessageExpiryRelay` contract.
contract MessageExpiryRelay_Initialize_Test is MessageExpiryRelay_TestInit {
    /// @notice Tests that the ProxyAdmin owner can initialize the contract.
    function test_initialize_succeeds() public {
        _initializeRelay(hub, EXPIRY_WINDOW);

        assertEq(messageExpiryRelay.hub(), hub);
        assertEq(messageExpiryRelay.expiryWindow(), EXPIRY_WINDOW);
    }

    /// @notice Tests that the ProxyAdmin itself can initialize the contract.
    function test_initialize_proxyAdmin_succeeds() public {
        vm.prank(address(proxyAdmin));
        messageExpiryRelay.initialize(hub, EXPIRY_WINDOW);

        assertEq(messageExpiryRelay.hub(), hub);
        assertEq(messageExpiryRelay.expiryWindow(), EXPIRY_WINDOW);
    }

    /// @notice Tests that zero values are accepted, which locks the contract into its fail-closed
    ///         state rather than reverting.
    function test_initialize_zeroValues_succeeds() public {
        _initializeRelay(address(0), 0);

        assertEq(messageExpiryRelay.hub(), address(0));
        assertEq(messageExpiryRelay.expiryWindow(), 0);
    }

    /// @notice Tests that the largest expiry window that can never overflow the expiry check is
    ///         accepted.
    function test_initialize_maxExpiryWindow_succeeds() public {
        _initializeRelay(hub, type(uint64).max);

        assertEq(messageExpiryRelay.expiryWindow(), type(uint64).max);
    }

    /// @notice Tests that an expiry window beyond `type(uint64).max` is rejected, since
    ///         `recordedAt + expiryWindow` could then overflow and brick expiry consumption.
    /// @param _expiryWindow Expiry window to initialize with.
    function testFuzz_initialize_expiryWindowTooLarge_reverts(uint256 _expiryWindow) public {
        _expiryWindow = bound(_expiryWindow, uint256(type(uint64).max) + 1, type(uint256).max);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_InvalidExpiryWindow.selector);
        vm.prank(proxyAdminOwner);
        messageExpiryRelay.initialize(hub, _expiryWindow);
    }

    /// @notice Tests that the first value beyond the allowed expiry window is rejected.
    function test_initialize_expiryWindowTooLarge_reverts() public {
        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_InvalidExpiryWindow.selector);
        vm.prank(proxyAdminOwner);
        messageExpiryRelay.initialize(hub, uint256(type(uint64).max) + 1);
    }

    /// @notice Tests that the contract cannot be initialized by anyone other than the ProxyAdmin or
    ///         the ProxyAdmin owner.
    /// @param _caller Address attempting the initialization.
    function testFuzz_initialize_notProxyAdminOrOwner_reverts(address _caller) public {
        vm.assume(_caller != address(proxyAdmin));
        vm.assume(_caller != proxyAdminOwner);

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);
        vm.prank(_caller);
        messageExpiryRelay.initialize(hub, EXPIRY_WINDOW);
    }

    /// @notice Tests that the contract cannot be initialized twice.
    function test_initialize_alreadyInitialized_reverts() public {
        _initializeRelay(hub, EXPIRY_WINDOW);

        vm.expectRevert("Initializable: contract is already initialized");
        vm.prank(proxyAdminOwner);
        messageExpiryRelay.initialize(hub, EXPIRY_WINDOW);
    }
}

/// @title MessageExpiryRelay_RecordSentMessage_Test
/// @notice Tests the `recordSentMessage` function of the `MessageExpiryRelay` contract.
contract MessageExpiryRelay_RecordSentMessage_Test is MessageExpiryRelay_TestInit {
    /// @notice Tests that a message sent by the caller is recorded, emits the event and returns the
    ///         message hash.
    function test_recordSentMessage_succeeds() public {
        uint256 nonce = 7;
        address target = makeAddr("target");
        bytes memory message = hex"deadbeef";
        bytes32 expectedHash = _messageHash(DESTINATION, nonce, address(app), target, message);

        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.sentMessages, (nonce)),
            abi.encode(expectedHash)
        );

        vm.expectEmit(address(messageExpiryRelay));
        emit SentMessageRecorded(expectedHash, address(app), DESTINATION, block.timestamp);

        vm.prank(address(app));
        bytes32 msgHash = messageExpiryRelay.recordSentMessage(DESTINATION, nonce, target, message);

        assertEq(msgHash, expectedHash);

        (address recordedApp, uint96 recordedAt, uint256 destination) = messageExpiryRelay.sentMessageRecords(msgHash);
        assertEq(recordedApp, address(app));
        assertEq(uint256(recordedAt), block.timestamp);
        assertEq(destination, DESTINATION);
    }

    /// @notice Tests that any sent message can be recorded by its sender.
    /// @param _sender      Application recording the message.
    /// @param _destination Chain ID of the destination chain.
    /// @param _nonce       Nonce of the sent message.
    /// @param _target      Target of the message on the destination chain.
    /// @param _message     Message payload.
    function testFuzz_recordSentMessage_succeeds(
        address _sender,
        uint256 _destination,
        uint256 _nonce,
        address _target,
        bytes memory _message
    )
        public
    {
        vm.assume(_sender != address(0));
        assumeNotPrecompile(_sender);
        assumeNotForgeAddress(_sender);
        vm.assume(_sender != Predeploys.MESSAGE_EXPIRY_RELAY);
        vm.assume(_sender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        // A handler must be a contract (see `recordSentMessage`), so give the fuzzed sender code.
        vm.etch(_sender, hex"01");

        bytes32 expectedHash = _messageHash(_destination, _nonce, _sender, _target, _message);

        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.sentMessages, (_nonce)),
            abi.encode(expectedHash)
        );

        vm.expectEmit(address(messageExpiryRelay));
        emit SentMessageRecorded(expectedHash, _sender, _destination, block.timestamp);

        vm.prank(_sender);
        bytes32 msgHash = messageExpiryRelay.recordSentMessage(_destination, _nonce, _target, _message);

        assertEq(msgHash, expectedHash);

        (address recordedApp, uint96 recordedAt, uint256 destination) = messageExpiryRelay.sentMessageRecords(msgHash);
        assertEq(recordedApp, _sender);
        assertEq(uint256(recordedAt), block.timestamp);
        assertEq(destination, _destination);
    }

    /// @notice Tests that recording reverts when the messenger has no matching sent message at the
    ///         given nonce.
    function test_recordSentMessage_messageNotSent_reverts() public {
        uint256 nonce = 7;
        address target = makeAddr("target");
        bytes memory message = hex"deadbeef";

        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.sentMessages, (nonce)),
            abi.encode(keccak256("some other message"))
        );

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_MessageNotSent.selector);
        vm.prank(address(app));
        messageExpiryRelay.recordSentMessage(DESTINATION, nonce, target, message);
    }

    /// @notice Tests that a different caller cannot record a message it did not send, even when it
    ///         supplies the exact preimage of a real sent message: the hash is bound to the caller,
    ///         so the recomputed hash no longer matches the messenger's record.
    function test_recordSentMessage_differentSender_reverts() public {
        uint256 nonce = 7;
        address target = makeAddr("target");
        bytes memory message = hex"deadbeef";
        address thief = makeAddr("thief");
        // The thief is a contract (a different one than the real sender), so it passes the handler
        // code check and reaches the sender-bound hash check.
        vm.etch(thief, hex"01");

        // The real sender records the message, which also mocks the messenger's `sentMessages`.
        _recordSentMessage(address(app), DESTINATION, nonce, target, message);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_MessageNotSent.selector);
        vm.prank(thief);
        messageExpiryRelay.recordSentMessage(DESTINATION, nonce, target, message);
    }

    /// @notice Tests that an EOA cannot register itself as a handler: an EOA handler would make the
    ///         expiry callback revert forever, so recording from a codeless caller is rejected.
    function test_recordSentMessage_eoaHandler_reverts() public {
        address eoa = makeAddr("eoa");

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_HandlerNotContract.selector);
        vm.prank(eoa);
        messageExpiryRelay.recordSentMessage(DESTINATION, 7, makeAddr("target"), hex"deadbeef");
    }

    /// @notice Tests that a message cannot be recorded twice.
    function test_recordSentMessage_alreadyRecorded_reverts() public {
        uint256 nonce = 7;
        address target = makeAddr("target");
        bytes memory message = hex"deadbeef";

        _recordSentMessage(address(app), DESTINATION, nonce, target, message);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_AlreadyRecorded.selector);
        vm.prank(address(app));
        messageExpiryRelay.recordSentMessage(DESTINATION, nonce, target, message);
    }
}

/// @title MessageExpiryRelay_AttestUndelivered_Test
/// @notice Tests the `attestUndelivered` function of the `MessageExpiryRelay` contract.
contract MessageExpiryRelay_AttestUndelivered_Test is MessageExpiryRelay_TestInit {
    /// @notice Chain ID the attested message was sent from.
    uint256 internal constant SOURCE = 903;

    /// @notice Minimum gas limit requested for the hub call on L1.
    uint32 internal constant MIN_GAS_LIMIT = 200_000;

    /// @notice Hash of the message being attested.
    bytes32 internal msgHash = keccak256("undelivered message");

    /// @notice Tests that attesting an undelivered message exports the attestation to the hub and
    ///         emits the event.
    function test_attestUndelivered_succeeds() public {
        _initializeRelay(hub, EXPIRY_WINDOW);

        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.successfulMessages, (msgHash)),
            abi.encode(false)
        );
        _mockAndExpect(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(
                ICrossDomainMessenger.sendMessage,
                (
                    hub,
                    abi.encodeCall(IMessageExpiryHub.receiveExpiryNotice, (msgHash, SOURCE, block.timestamp)),
                    MIN_GAS_LIMIT
                )
            ),
            abi.encode()
        );

        vm.expectEmit(address(messageExpiryRelay));
        emit UndeliveredMessageAttested(msgHash, SOURCE, block.timestamp);

        messageExpiryRelay.attestUndelivered(msgHash, SOURCE, MIN_GAS_LIMIT);
    }

    /// @notice Tests that attesting reverts when the message was already delivered on this chain.
    function test_attestUndelivered_messageDelivered_reverts() public {
        _initializeRelay(hub, EXPIRY_WINDOW);

        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.successfulMessages, (msgHash)),
            abi.encode(true)
        );

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_MessageDelivered.selector);
        messageExpiryRelay.attestUndelivered(msgHash, SOURCE, MIN_GAS_LIMIT);
    }

    /// @notice Tests that attesting reverts when the named source chain is this chain.
    function test_attestUndelivered_invalidSourceChain_reverts() public {
        _initializeRelay(hub, EXPIRY_WINDOW);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_InvalidSourceChain.selector);
        messageExpiryRelay.attestUndelivered(msgHash, block.chainid, MIN_GAS_LIMIT);
    }

    /// @notice Tests that attesting reverts when the relay has no hub configured.
    function test_attestUndelivered_notInitialized_reverts() public {
        assertEq(messageExpiryRelay.hub(), address(0));

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_NotInitialized.selector);
        messageExpiryRelay.attestUndelivered(msgHash, SOURCE, MIN_GAS_LIMIT);
    }
}

/// @title MessageExpiryRelay_ReceiveExpiry_Test
/// @notice Tests the `receiveExpiry` function of the `MessageExpiryRelay` contract.
contract MessageExpiryRelay_ReceiveExpiry_Test is MessageExpiryRelay_TestInit {
    /// @notice Nonce of the recorded message.
    uint256 internal constant NONCE = 42;

    /// @notice Hash of the recorded message.
    bytes32 internal msgHash;

    /// @notice Timestamp at which the message was recorded.
    uint256 internal recordedAt;

    /// @notice Test setup. Initializes the relay and records a message sent by `app`.
    function setUp() public virtual override {
        super.setUp();

        _initializeRelay(hub, EXPIRY_WINDOW);
        msgHash = _recordSentMessage(address(app), DESTINATION, NONCE, makeAddr("target"), hex"deadbeef");
        recordedAt = block.timestamp;
    }

    /// @notice Tests that a valid expiry consumes the record, notifies the application and emits
    ///         the event.
    function test_receiveExpiry_succeeds() public {
        uint256 attestedAt = recordedAt + EXPIRY_WINDOW + 1;
        _mockXDomainMessageSender(hub);

        vm.expectCall(
            address(app), abi.encodeCall(IMessageExpiryHandler.onMessageExpired, (msgHash, DESTINATION, attestedAt)), 1
        );
        vm.expectEmit(address(messageExpiryRelay));
        emit ExpiredMessageRelayed(msgHash, address(app), DESTINATION, attestedAt);

        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messageExpiryRelay.receiveExpiry(msgHash, DESTINATION, attestedAt);

        // The record is consumed.
        (address recordedApp,,) = messageExpiryRelay.sentMessageRecords(msgHash);
        assertEq(recordedApp, address(0));

        // The application was notified exactly once, with the attestation as given.
        assertEq(app.callCount(), 1);
        assertEq(app.lastMsgHash(), msgHash);
        assertEq(app.lastAttestorChainId(), DESTINATION);
        assertEq(app.lastAttestedAt(), attestedAt);
    }

    /// @notice Tests that an expiry is rejected when it does not come from the
    ///         `L2CrossDomainMessenger`.
    /// @param _caller Address attempting the call.
    function testFuzz_receiveExpiry_notMessenger_reverts(address _caller) public {
        vm.assume(_caller != Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        _mockXDomainMessageSender(hub);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_InvalidExpirySender.selector);
        vm.prank(_caller);
        messageExpiryRelay.receiveExpiry(msgHash, DESTINATION, recordedAt + EXPIRY_WINDOW + 1);
    }

    /// @notice Tests that an expiry relayed by the messenger on behalf of anyone other than the hub
    ///         is rejected.
    /// @param _sender Cross domain sender of the relayed message.
    function testFuzz_receiveExpiry_notHub_reverts(address _sender) public {
        vm.assume(_sender != hub);
        _mockXDomainMessageSender(_sender);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_InvalidExpirySender.selector);
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messageExpiryRelay.receiveExpiry(msgHash, DESTINATION, recordedAt + EXPIRY_WINDOW + 1);
    }

    /// @notice Tests that an expiry for a message that was never recorded is rejected.
    function test_receiveExpiry_messageNotRecorded_reverts() public {
        _mockXDomainMessageSender(hub);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_MessageNotRecorded.selector);
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messageExpiryRelay.receiveExpiry(keccak256("unknown"), DESTINATION, recordedAt + EXPIRY_WINDOW + 1);
    }

    /// @notice Tests that an expiry attested by a chain other than the recorded destination is
    ///         rejected.
    /// @param _attestorChainId Chain ID claimed by the attestation.
    function testFuzz_receiveExpiry_wrongAttestor_reverts(uint256 _attestorChainId) public {
        vm.assume(_attestorChainId != DESTINATION);
        _mockXDomainMessageSender(hub);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_WrongAttestor.selector);
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messageExpiryRelay.receiveExpiry(msgHash, _attestorChainId, recordedAt + EXPIRY_WINDOW + 1);
    }

    /// @notice Tests that an attestation taken exactly at the end of the expiry window is rejected,
    ///         since delivery may still have been possible at that timestamp.
    function test_receiveExpiry_atWindowBoundary_reverts() public {
        _mockXDomainMessageSender(hub);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_MessageNotExpired.selector);
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messageExpiryRelay.receiveExpiry(msgHash, DESTINATION, recordedAt + EXPIRY_WINDOW);
    }

    /// @notice Tests that any attestation at or before the end of the expiry window is rejected.
    /// @param _attestedAt Timestamp of the attestation.
    function testFuzz_receiveExpiry_notExpired_reverts(uint256 _attestedAt) public {
        _attestedAt = bound(_attestedAt, 0, recordedAt + EXPIRY_WINDOW);
        _mockXDomainMessageSender(hub);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_MessageNotExpired.selector);
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messageExpiryRelay.receiveExpiry(msgHash, DESTINATION, _attestedAt);
    }

    /// @notice Tests that an attestation one second past the end of the expiry window is accepted.
    function test_receiveExpiry_justPastWindow_succeeds() public {
        uint256 attestedAt = recordedAt + EXPIRY_WINDOW + 1;
        _mockXDomainMessageSender(hub);

        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messageExpiryRelay.receiveExpiry(msgHash, DESTINATION, attestedAt);

        assertEq(app.callCount(), 1);
        (address recordedApp,,) = messageExpiryRelay.sentMessageRecords(msgHash);
        assertEq(recordedApp, address(0));
    }

    /// @notice Tests that a message can only be expired once.
    function test_receiveExpiry_alreadyConsumed_reverts() public {
        uint256 attestedAt = recordedAt + EXPIRY_WINDOW + 1;
        _mockXDomainMessageSender(hub);

        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messageExpiryRelay.receiveExpiry(msgHash, DESTINATION, attestedAt);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_MessageNotRecorded.selector);
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messageExpiryRelay.receiveExpiry(msgHash, DESTINATION, attestedAt);

        assertEq(app.callCount(), 1);
    }

    /// @notice Tests that a reverting application callback reverts the whole call and leaves the
    ///         record in place, so the relayed message stays replayable.
    function test_receiveExpiry_handlerReverts_reverts() public {
        uint256 attestedAt = recordedAt + EXPIRY_WINDOW + 1;
        app.setShouldRevert(true);
        _mockXDomainMessageSender(hub);

        vm.expectRevert(MessageExpiryRelay_App_Harness.MessageExpiryRelay_App_Harness_Reverted.selector);
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messageExpiryRelay.receiveExpiry(msgHash, DESTINATION, attestedAt);

        // The record survives the reverted call.
        (address recordedApp, uint96 storedRecordedAt, uint256 destination) =
            messageExpiryRelay.sentMessageRecords(msgHash);
        assertEq(recordedApp, address(app));
        assertEq(uint256(storedRecordedAt), recordedAt);
        assertEq(destination, DESTINATION);
    }
}

/// @title MessageExpiryRelay_ReceiveExpiry_ZeroWindow_Test
/// @notice Tests that the `receiveExpiry` function of the `MessageExpiryRelay` contract fails
///         closed when the expiry window was never configured.
contract MessageExpiryRelay_ReceiveExpiry_ZeroWindow_Test is MessageExpiryRelay_TestInit {
    /// @notice Hash of the recorded message.
    bytes32 internal msgHash;

    /// @notice Test setup. Initializes the relay with a hub but a zero expiry window.
    function setUp() public virtual override {
        super.setUp();

        _initializeRelay(hub, 0);
        msgHash = _recordSentMessage(address(app), DESTINATION, 42, makeAddr("target"), hex"deadbeef");
    }

    /// @notice Tests that expiries are rejected while the expiry window is zero, even for a
    ///         recorded message relayed by the hub.
    function test_receiveExpiry_zeroExpiryWindow_reverts() public {
        assertEq(messageExpiryRelay.expiryWindow(), 0);
        _mockXDomainMessageSender(hub);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_NotInitialized.selector);
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messageExpiryRelay.receiveExpiry(msgHash, DESTINATION, block.timestamp + 1);
    }
}

/// @title MessageExpiryRelay_SetExpiryWindow_Test
/// @notice Tests the `setExpiryWindow` function of the `MessageExpiryRelay` contract.
contract MessageExpiryRelay_SetExpiryWindow_Test is MessageExpiryRelay_TestInit {
    event ExpiryWindowSet(uint256 oldWindow, uint256 newWindow);

    /// @notice Test setup. Initializes the relay with a hub and a nonzero expiry window.
    function setUp() public virtual override {
        super.setUp();
        _initializeRelay(hub, EXPIRY_WINDOW);
    }

    /// @notice Tests that the ProxyAdmin owner can raise the expiry window and that the event is
    ///         emitted.
    function test_setExpiryWindow_increase_succeeds() public {
        uint256 newWindow = EXPIRY_WINDOW + 1 days;

        vm.expectEmit(address(messageExpiryRelay));
        emit ExpiryWindowSet(EXPIRY_WINDOW, newWindow);

        vm.prank(proxyAdminOwner);
        messageExpiryRelay.setExpiryWindow(newWindow);

        assertEq(messageExpiryRelay.expiryWindow(), newWindow);
    }

    /// @notice Tests that the ProxyAdmin itself can raise the expiry window.
    function test_setExpiryWindow_proxyAdmin_succeeds() public {
        uint256 newWindow = EXPIRY_WINDOW + 1;

        vm.prank(address(proxyAdmin));
        messageExpiryRelay.setExpiryWindow(newWindow);

        assertEq(messageExpiryRelay.expiryWindow(), newWindow);
    }

    /// @notice Tests that the window can be raised up to the largest value that fits in a uint64.
    function test_setExpiryWindow_maxExpiryWindow_succeeds() public {
        vm.prank(proxyAdminOwner);
        messageExpiryRelay.setExpiryWindow(type(uint64).max);

        assertEq(messageExpiryRelay.expiryWindow(), type(uint64).max);
    }

    /// @notice Tests that setting the window to its current value is rejected: the window may only
    ///         increase.
    function test_setExpiryWindow_equal_reverts() public {
        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_ExpiryWindowNotIncreased.selector);
        vm.prank(proxyAdminOwner);
        messageExpiryRelay.setExpiryWindow(EXPIRY_WINDOW);
    }

    /// @notice Tests that lowering the window (or leaving it equal) is rejected.
    /// @param _expiryWindow New expiry window to attempt.
    function testFuzz_setExpiryWindow_decreaseOrEqual_reverts(uint256 _expiryWindow) public {
        _expiryWindow = bound(_expiryWindow, 0, EXPIRY_WINDOW);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_ExpiryWindowNotIncreased.selector);
        vm.prank(proxyAdminOwner);
        messageExpiryRelay.setExpiryWindow(_expiryWindow);
    }

    /// @notice Tests that a window beyond `type(uint64).max` is rejected, since `recordedAt +
    ///         expiryWindow` could then overflow.
    /// @param _expiryWindow New expiry window to attempt.
    function testFuzz_setExpiryWindow_tooLarge_reverts(uint256 _expiryWindow) public {
        _expiryWindow = bound(_expiryWindow, uint256(type(uint64).max) + 1, type(uint256).max);

        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_InvalidExpiryWindow.selector);
        vm.prank(proxyAdminOwner);
        messageExpiryRelay.setExpiryWindow(_expiryWindow);
    }

    /// @notice Tests that the window cannot be changed by anyone other than the ProxyAdmin or the
    ///         ProxyAdmin owner.
    /// @param _caller Address attempting the change.
    function testFuzz_setExpiryWindow_notProxyAdminOrOwner_reverts(address _caller) public {
        vm.assume(_caller != address(proxyAdmin));
        vm.assume(_caller != proxyAdminOwner);

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);
        vm.prank(_caller);
        messageExpiryRelay.setExpiryWindow(EXPIRY_WINDOW + 1);
    }
}

/// @title MessageExpiryRelay_SetHub_Test
/// @notice Tests the `setHub` function of the `MessageExpiryRelay` contract.
contract MessageExpiryRelay_SetHub_Test is MessageExpiryRelay_TestInit {
    event HubSet(address oldHub, address newHub);

    /// @notice Test setup. Initializes the relay with a hub and a nonzero expiry window.
    function setUp() public virtual override {
        super.setUp();
        _initializeRelay(hub, EXPIRY_WINDOW);
    }

    /// @notice Tests that the ProxyAdmin owner can re-point the hub and that the event is emitted.
    function test_setHub_succeeds() public {
        address newHub = makeAddr("newHub");

        vm.expectEmit(address(messageExpiryRelay));
        emit HubSet(hub, newHub);

        vm.prank(proxyAdminOwner);
        messageExpiryRelay.setHub(newHub);

        assertEq(messageExpiryRelay.hub(), newHub);
    }

    /// @notice Tests that the ProxyAdmin itself can re-point the hub.
    function test_setHub_proxyAdmin_succeeds() public {
        address newHub = makeAddr("newHub");

        vm.prank(address(proxyAdmin));
        messageExpiryRelay.setHub(newHub);

        assertEq(messageExpiryRelay.hub(), newHub);
    }

    /// @notice Tests that the hub cannot be set to the zero address, which would disable
    ///         attestation.
    function test_setHub_zeroAddress_reverts() public {
        vm.expectRevert(IMessageExpiryRelay.MessageExpiryRelay_InvalidHub.selector);
        vm.prank(proxyAdminOwner);
        messageExpiryRelay.setHub(address(0));
    }

    /// @notice Tests that the hub cannot be changed by anyone other than the ProxyAdmin or the
    ///         ProxyAdmin owner.
    /// @param _caller Address attempting the change.
    function testFuzz_setHub_notProxyAdminOrOwner_reverts(address _caller) public {
        vm.assume(_caller != address(proxyAdmin));
        vm.assume(_caller != proxyAdminOwner);

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);
        vm.prank(_caller);
        messageExpiryRelay.setHub(makeAddr("newHub"));
    }
}
