// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Kontrol (KEVM) proofs for the L2ToL2CrossDomainMessenger side of per-message interop expiry.
//
// Model: the REAL L2ToL2CrossDomainMessenger runtime code (built from src/) is etched at
// 0x4200..0023. Its storage is made fully symbolic (kevm.symbolicStorage) wherever the proof says
// so, so every mapping entry (successfulMessages, sentMessageTimestamps, expiredMessages, msgNonce,
// ...) starts as an arbitrary value. Mocks (ExpiryMocks0825.sol) are etched at 0x..0007, 0x..0016
// and 0x..0022. Every other address has no code (the implementations deployed in setUp are wiped
// after being copied). block.chainid, block.timestamp and msg.sender are symbolic where stated.
// Message bytes have a fixed length of 600 (`kontrol-bytes-length-equals`).
//
// Not runnable with plain forge (uses Kontrol-only cheat codes
// freshUInt/freshAddress/symbolicStorage).

// Contracts
import {
    L2ToL2CrossDomainMessenger,
    L2ToL2CrossDomainMessenger_MessageTargetUnsafe
} from "src/L2/L2ToL2CrossDomainMessenger.sol";
import {
    ExpiryKontrolBaseL2,
    RecordingMock,
    AcceptAllCrossL2Inbox,
    AuthL2CrossDomainMessenger
} from "test/formal/expiry/kontrol/solc0825/ExpiryMocks0825.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

contract L2ToL2CrossDomainMessengerExpiryKontrol is ExpiryKontrolBaseL2 {
    address internal constant L2TOL2 = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER; // 0x..23
    address internal constant L2CDM = Predeploys.L2_CROSS_DOMAIN_MESSENGER; // 0x..07
    address internal constant PASSER = Predeploys.L2_TO_L1_MESSAGE_PASSER; // 0x..16
    address internal constant INBOX = Predeploys.CROSS_L2_INBOX; // 0x..22

    /// @notice keccak256("SentMessage(uint256,address,uint256,address,bytes)"), as in the contract.
    bytes32 internal constant SENT_MESSAGE_EVENT_SELECTOR =
        0x382409ac69001e11931a28435afef442cbfd20d9891907e8fa373ba7d351f320;

    /// @notice RecordingMock storage: slot 0 = mapping(address => uint256) callsFrom.
    uint256 internal constant REC_CALLS_SLOT = 0;

    /// @notice L2ToL2CrossDomainMessenger storage slot of msgNonce (uint240).
    uint256 internal constant MSG_NONCE_SLOT = 1;

    /// @notice L2ToL2CrossDomainMessenger storage slot of expiryPeriod.
    uint256 internal constant EXPIRY_PERIOD_SLOT = 5;

    /// @notice The bound initialize puts on the expiry period.
    uint256 internal constant MAX_EXPIRY_PERIOD = 365 days;

    L2ToL2CrossDomainMessenger internal constant l2tol2 = L2ToL2CrossDomainMessenger(L2TOL2);

    function setUp() public {
        _etch(L2TOL2, address(new L2ToL2CrossDomainMessenger()));
        // expiryPeriod (slot 5), as initialize sets it on every production network. Proofs that make
        // the messenger's storage symbolic make it symbolic too.
        vm.store(L2TOL2, bytes32(EXPIRY_PERIOD_SLOT), bytes32(Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD));
        _etch(INBOX, address(new AcceptAllCrossL2Inbox()));
        _etch(PASSER, address(new RecordingMock()));
    }

    /// @notice Copies `impl`'s runtime code to `at` and then wipes `impl`'s code, so the only live
    ///         copy of each contract is the one at the predeploy address.
    function _etch(address _at, address _impl) internal {
        vm.etch(_at, _impl.code);
        vm.etch(_impl, hex"");
    }

    function _useRecordingL2CDM() internal {
        _etch(L2CDM, address(new RecordingMock()));
    }

    function _callsFrom(address _mock, address _caller) internal view returns (uint256) {
        return uint256(vm.load(_mock, keccak256(abi.encode(_caller, REC_CALLS_SLOT))));
    }

    /// @notice Canonical SentMessage payload: abi.encodePacked(abi.encode(selector, destination,
    ///         target, nonce), abi.encode(sender, message)), exactly what the source messenger's
    ///         event yields. relayMessage's behaviour after decoding depends only on the decoded
    ///         tuple, and every payload that decodes yields a tuple that this canonical encoding
    ///         also produces, so fixing the encoding loses no decoded behaviour.
    function _payload(
        uint256 _destination,
        address _target,
        uint256 _nonce,
        address _sender,
        bytes memory _message
    )
        internal
        pure
        returns (bytes memory)
    {
        return abi.encodePacked(
            abi.encode(SENT_MESSAGE_EVENT_SELECTOR, _destination, _target, _nonce), abi.encode(_sender, _message)
        );
    }

    function _relay(
        uint256 _blockNumber,
        uint256 _logIndex,
        uint256 _timestamp,
        uint256 _source,
        bytes memory _sentMessage
    )
        internal
        returns (bool ok_, bytes memory ret_)
    {
        Identifier memory id = Identifier(L2TOL2, _blockNumber, _logIndex, _timestamp, _source);
        (ok_, ret_) = L2TOL2.call(abi.encodeCall(L2ToL2CrossDomainMessenger.relayMessage, (id, _sentMessage)));
    }

    /// @notice Relays a message from `_sender` on `_source` whose target is the messenger itself,
    ///         addressed to the current chain.
    function _relaySelf(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        bytes memory _message
    )
        internal
        returns (bool ok_)
    {
        (ok_,) = _relay(1, 0, 1, _source, _payload(block.chainid, L2TOL2, _nonce, _sender, _message));
    }

    function _symbolicChain() internal returns (uint256 chainId_) {
        chainId_ = kevm.freshUInt(32);
        vm.chainId(chainId_);
    }

    function _send(uint256 _destination, address _target, bytes calldata _message) internal returns (bool ok_) {
        vm.prank(kevm.freshAddress());
        (ok_,) = L2TOL2.call(abi.encodeCall(L2ToL2CrossDomainMessenger.sendMessage, (_destination, _target, _message)));
    }

    /// @notice msgNonce++ on a uint240 is checked; the nonce space (2^240 sends) cannot be
    ///         exhausted in practice. The 16 bits above msgNonce in its slot are never written by
    ///         the contract, so they are zero.
    function _assumeNonceNotExhausted() internal view {
        uint256 slot = uint256(vm.load(L2TOL2, bytes32(MSG_NONCE_SLOT)));
        vm.assume(slot < type(uint240).max);
    }

    // ---------------------------------------------------------------------------------------------
    // 1. UnsafeTargetRule
    // ---------------------------------------------------------------------------------------------

    /// @notice sendMessage succeeds IFF destination != chainid and target is none of the messenger
    ///         itself, the L2CrossDomainMessenger and the L2ToL1MessagePasser (for ALL destination,
    ///         target, 600-byte message, caller, timestamp, chainid and storage with msgNonce <
    ///         2^240 - 1).
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_sendMessage_unsafeTargetRule(
        uint256 _destination,
        address _target,
        bytes calldata _message
    )
        external
    {
        uint256 chainId = _symbolicChain();
        vm.warp(kevm.freshUInt(8));
        kevm.symbolicStorage(L2TOL2);
        _assumeNonceNotExhausted();

        bool ok = _send(_destination, _target, _message);

        assert(ok == (_destination != chainId && _target != L2TOL2 && _target != L2CDM && _target != PASSER));
    }

    /// @notice WITNESS (expected to FAIL): under the assumptions of the sendMessage proofs, a send
    ///         can succeed.
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_sendMessage_canSucceed_WITNESS(
        uint256 _destination,
        address _target,
        bytes calldata _message
    )
        external
    {
        _symbolicChain();
        vm.warp(kevm.freshUInt(8));
        _useRecordingL2CDM();
        kevm.symbolicStorage(L2TOL2);
        _assumeNonceNotExhausted();
        assert(!_send(_destination, _target, _message));
    }

    /// @notice sendMessage to the L2ToL1MessagePasser always reverts (was the pending rule at
    ///         37b44c48c7).
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_sendMessage_rejectsPasser(uint256 _destination, bytes calldata _message) external {
        _symbolicChain();
        kevm.symbolicStorage(L2TOL2);
        assert(!_send(_destination, PASSER, _message));
    }

    /// @notice relayMessage reverts with L2ToL2CrossDomainMessenger_MessageTargetUnsafe for EVERY
    ///         payload whose target is the L2CrossDomainMessenger (inbox accepts everything;
    ///         successfulMessages arbitrary; identifier, source, nonce, sender, 600-byte message
    ///         symbolic; destination = chainid).
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_relayMessage_rejectsL2CrossDomainMessenger(
        uint256 _blockNumber,
        uint256 _logIndex,
        uint256 _timestamp,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        bytes calldata _message
    )
        external
    {
        uint256 chainId = _symbolicChain();
        _useRecordingL2CDM();
        kevm.symbolicStorage(L2TOL2);

        bytes memory payload = _payload(chainId, L2CDM, _nonce, _sender, _message);
        (bool ok, bytes memory ret) = _relay(_blockNumber, _logIndex, _timestamp, _source, payload);

        assert(!ok);
        assert(ret.length == 4);
        assert(bytes4(ret) == L2ToL2CrossDomainMessenger_MessageTargetUnsafe.selector);
    }

    /// @notice Same for target == L2ToL1MessagePasser (was the pending rule at 37b44c48c7).
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_relayMessage_rejectsPasser(
        uint256 _blockNumber,
        uint256 _logIndex,
        uint256 _timestamp,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        bytes calldata _message
    )
        external
    {
        uint256 chainId = _symbolicChain();
        kevm.symbolicStorage(L2TOL2);

        bytes memory payload = _payload(chainId, PASSER, _nonce, _sender, _message);
        (bool ok, bytes memory ret) = _relay(_blockNumber, _logIndex, _timestamp, _source, payload);

        assert(!ok);
        assert(ret.length == 4);
        assert(bytes4(ret) == L2ToL2CrossDomainMessenger_MessageTargetUnsafe.selector);
    }

    /// @notice WITNESS (expected to FAIL): relayMessage can succeed in this harness (target without
    ///         code), so the relayMessage proofs here are not vacuous.
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_relayMessage_canSucceed_WITNESS(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        bytes calldata _message
    )
        external
    {
        uint256 chainId = _symbolicChain();
        kevm.symbolicStorage(L2TOL2);
        bytes memory payload = _payload(chainId, address(0xC0DE), _nonce, _sender, _message);
        (bool ok,) = _relay(1, 0, 1, _source, payload);
        assert(!ok);
    }

    // ---------------------------------------------------------------------------------------------
    // 2. OnlyExportReachesL1 (messenger side: the messenger never calls 0x..07 or 0x..16)
    // ---------------------------------------------------------------------------------------------

    /// @notice sendMessage never calls anything, so it never makes 0x..23 a caller of 0x..07 or
    ///         0x..16.
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_sendMessage_neverCallsL2CDMOrPasser(
        uint256 _destination,
        address _target,
        bytes calldata _message
    )
        external
    {
        _symbolicChain();
        _useRecordingL2CDM();
        kevm.symbolicStorage(L2TOL2);
        _send(_destination, _target, _message);
        assert(_callsFrom(L2CDM, L2TOL2) == 0);
        assert(_callsFrom(PASSER, L2TOL2) == 0);
    }

    /// @notice No relayMessage, for ANY target and 600-byte message, makes 0x..23 a (non-static)
    ///         caller of 0x..07 or 0x..16. Target == 0x..23 is covered by the next proof. Harness
    ///         exclusions: the test contract, the cheat-code address, the console.log address, and
    ///         the precompile range 0x0..0x1ff (precompiles never call other contracts; KEVM's
    ///         BLS12 precompile hooks crash on symbolic input).
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_relayMessage_neverMakesMessengerCallL2CDMOrPasser(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message
    )
        external
    {
        uint256 chainId = _symbolicChain();
        _useRecordingL2CDM();
        kevm.symbolicStorage(L2TOL2);
        vm.assume(_target != L2TOL2);
        vm.assume(_target != address(this) && _target != address(vm) && _target != CONSOLE);
        vm.assume(uint160(_target) > 0x1ff);

        bytes memory payload = _payload(chainId, _target, _nonce, _sender, _message);
        _relay(1, 0, 1, _source, payload);

        assert(_callsFrom(L2CDM, L2TOL2) == 0);
        assert(_callsFrom(PASSER, L2TOL2) == 0);
    }

    /// @notice WITNESS (expected to FAIL): under the assumptions of the proof above (symbolic
    ///         target with its exclusions), a relay can succeed.
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_relayMessage_symbolicTargetCanSucceed_WITNESS(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message
    )
        external
    {
        uint256 chainId = _symbolicChain();
        _useRecordingL2CDM();
        kevm.symbolicStorage(L2TOL2);
        vm.assume(_target != L2TOL2);
        vm.assume(_target != address(this) && _target != address(vm) && _target != CONSOLE);
        vm.assume(uint160(_target) > 0x1ff);
        (bool ok,) = _relay(1, 0, 1, _source, _payload(chainId, _target, _nonce, _sender, _message));
        assert(!ok);
    }

    // Target == 0x..23 itself (a payload no real source messenger emits, since sendMessage rejects
    // it): the relayed self-call never makes 0x..23 call 0x..07 or 0x..16. One proof per selector
    // of the nested message, because a fully symbolic nested message sends Kontrol through the
    // messenger's ABI decoder with symbolic offsets. Each nested message is the canonical ABI
    // encoding of its function's arguments: a message that does not decode reverts in the decoder,
    // before any call, and every message that decodes yields an argument tuple that the canonical
    // encoding also produces, so fixing the encoding loses no decoded behaviour. The byte lengths
    // are fixed, as Kontrol requires: the nested sendMessage's message is copied, never parsed, so
    // its length does not change which calls are made; nested relayMessage and expireMessage revert
    // before reading their arguments' contents; the other functions read at most one word. The
    // lengths are still a bound of these proofs: nested sendMessage is checked with an empty and a
    // 200-byte message (and sendMessage on its own with 600 bytes).

    /// @notice Nested sendMessage(destination, target, 200-byte message), every argument symbolic.
    /// @custom:kontrol-bytes-length-equals _inner: 200,
    function prove_relayMessage_selfTarget_sendMessage_neverCallsL2CDMOrPasser(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        uint256 _destination,
        address _target,
        bytes calldata _inner
    )
        external
    {
        _symbolicChain();
        _useRecordingL2CDM();
        kevm.symbolicStorage(L2TOL2);
        bytes memory message = abi.encodeCall(L2ToL2CrossDomainMessenger.sendMessage, (_destination, _target, _inner));
        _relaySelf(_source, _nonce, _sender, message);
        assert(_callsFrom(L2CDM, L2TOL2) == 0);
        assert(_callsFrom(PASSER, L2TOL2) == 0);
    }

    /// @notice Nested sendMessage(destination, target, empty message): the other end of the
    ///         message lengths, next to the 200-byte case above.
    function prove_relayMessage_selfTarget_sendMessageEmpty_neverCallsL2CDMOrPasser(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        uint256 _destination,
        address _target
    )
        external
    {
        _symbolicChain();
        _useRecordingL2CDM();
        kevm.symbolicStorage(L2TOL2);
        bytes memory message =
            abi.encodeCall(L2ToL2CrossDomainMessenger.sendMessage, (_destination, _target, bytes("")));
        _relaySelf(_source, _nonce, _sender, message);
        assert(_callsFrom(L2CDM, L2TOL2) == 0);
        assert(_callsFrom(PASSER, L2TOL2) == 0);
    }

    /// @notice WITNESS (expected to FAIL): a relayed self-call of sendMessage can succeed.
    /// @custom:kontrol-bytes-length-equals _inner: 200,
    function prove_relayMessage_selfTargetSendMessageCanSucceed_WITNESS(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        uint256 _destination,
        address _target,
        bytes calldata _inner
    )
        external
    {
        _symbolicChain();
        _useRecordingL2CDM();
        kevm.symbolicStorage(L2TOL2);
        bytes memory message = abi.encodeCall(L2ToL2CrossDomainMessenger.sendMessage, (_destination, _target, _inner));
        bool ok = _relaySelf(_source, _nonce, _sender, message);
        assert(!ok);
    }

    /// @notice Nested relayMessage(id, 288-byte payload), every argument symbolic. The relay
    ///         always fails: relayMessage is nonReentrant, and the outer relay is in progress.
    /// @custom:kontrol-bytes-length-equals _innerPayload: 288,
    function prove_relayMessage_selfTarget_relayMessage_neverCallsL2CDMOrPasser(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _idOrigin,
        uint256 _idBlockNumber,
        uint256 _idLogIndex,
        uint256 _idTimestamp,
        uint256 _idChainId,
        bytes calldata _innerPayload
    )
        external
    {
        _symbolicChain();
        _useRecordingL2CDM();
        kevm.symbolicStorage(L2TOL2);
        bytes memory message = abi.encodeCall(
            L2ToL2CrossDomainMessenger.relayMessage,
            (Identifier(_idOrigin, _idBlockNumber, _idLogIndex, _idTimestamp, _idChainId), _innerPayload)
        );
        bool ok = _relaySelf(_source, _nonce, _sender, message);
        assert(!ok);
        assert(_callsFrom(L2CDM, L2TOL2) == 0);
        assert(_callsFrom(PASSER, L2TOL2) == 0);
    }

    /// @notice Nested expireMessage(messageHash, undeliveredAt), both symbolic. The relay always
    ///         fails: the nested call's msg.sender is 0x..23, not 0x..07.
    function prove_relayMessage_selfTarget_expireMessage_neverCallsL2CDMOrPasser(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        bytes32 _messageHash,
        uint256 _undeliveredAt
    )
        external
    {
        _symbolicChain();
        _useRecordingL2CDM();
        kevm.symbolicStorage(L2TOL2);
        bytes memory message =
            abi.encodeCall(L2ToL2CrossDomainMessenger.expireMessage, (_messageHash, _undeliveredAt));
        bool ok = _relaySelf(_source, _nonce, _sender, message);
        assert(!ok);
        assert(_callsFrom(L2CDM, L2TOL2) == 0);
        assert(_callsFrom(PASSER, L2TOL2) == 0);
    }

    /// @notice Any other selector, followed by 64 symbolic bytes. The other functions are views
    ///         that take at most one 32-byte argument, so their decoders ignore bytes past it; a
    ///         message shorter than 4 bytes, or an unknown selector, reverts (no fallback).
    /// @custom:kontrol-bytes-length-equals _args: 64,
    function prove_relayMessage_selfTarget_otherSelectors_neverCallsL2CDMOrPasser(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        bytes4 _selector,
        bytes calldata _args
    )
        external
    {
        vm.assume(_selector != L2ToL2CrossDomainMessenger.sendMessage.selector);
        vm.assume(_selector != L2ToL2CrossDomainMessenger.relayMessage.selector);
        vm.assume(_selector != L2ToL2CrossDomainMessenger.expireMessage.selector);
        _symbolicChain();
        _useRecordingL2CDM();
        kevm.symbolicStorage(L2TOL2);
        bytes memory message = abi.encodePacked(_selector, _args);
        _relaySelf(_source, _nonce, _sender, message);
        assert(_callsFrom(L2CDM, L2TOL2) == 0);
        assert(_callsFrom(PASSER, L2TOL2) == 0);
    }

    /// @notice WITNESS (expected to FAIL): under the assumptions of the proof above, a relayed
    ///         self-call of some other selector (a view) succeeds.
    /// @custom:kontrol-bytes-length-equals _args: 64,
    function prove_relayMessage_selfTargetOtherSelectorCanSucceed_WITNESS(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        bytes4 _selector,
        bytes calldata _args
    )
        external
    {
        vm.assume(_selector != L2ToL2CrossDomainMessenger.sendMessage.selector);
        vm.assume(_selector != L2ToL2CrossDomainMessenger.relayMessage.selector);
        vm.assume(_selector != L2ToL2CrossDomainMessenger.expireMessage.selector);
        _symbolicChain();
        _useRecordingL2CDM();
        kevm.symbolicStorage(L2TOL2);
        bool ok = _relaySelf(_source, _nonce, _sender, abi.encodePacked(_selector, _args));
        assert(!ok);
    }

    // ---------------------------------------------------------------------------------------------
    // 4. expireMessage: auth and window boundary
    // ---------------------------------------------------------------------------------------------

    /// @notice For ALL messageHash, undeliveredAt, caller, L2CDM.xDomainMessageSender(),
    ///         L2CDM.otherMessenger() and storage, with sentAt = sentMessageTimestamps[H] < 2^64
    ///         (realistic block timestamps) and ANY stored expiryPeriod in initialize's range
    ///         (0, 365 days]: expireMessage succeeds IFF msg.sender == 0x..07 &&
    ///         xDomainMessageSender == otherMessenger && (expiredMessages[H] was already set ||
    ///         (sentAt != 0 && undeliveredAt > sentAt + EXPIRY_PERIOD, read from the contract, not
    ///         hardcoded)); afterwards expiredMessages[H] == old || success, and
    ///         sentMessageTimestamps[H] is unchanged. Events and other slots are not asserted.
    function prove_expireMessage_spec(bytes32 _messageHash, uint256 _undeliveredAt) external {
        _etch(L2CDM, address(new AuthL2CrossDomainMessenger()));
        kevm.symbolicStorage(L2TOL2);
        address xSender = kevm.freshAddress();
        address other = kevm.freshAddress();
        vm.store(L2CDM, bytes32(uint256(0)), bytes32(uint256(uint160(xSender))));
        vm.store(L2CDM, bytes32(uint256(1)), bytes32(uint256(uint160(other))));

        uint256 sentAt = l2tol2.sentMessageTimestamps(_messageHash);
        vm.assume(sentAt < 2 ** 64);
        bool expiredBefore = l2tol2.expiredMessages(_messageHash);
        // Any stored period initialize accepts.
        uint256 period = l2tol2.expiryPeriod();
        vm.assume(period > 0 && period <= MAX_EXPIRY_PERIOD);

        address caller = kevm.freshAddress();
        vm.prank(caller);
        (bool ok,) =
            L2TOL2.call(abi.encodeCall(L2ToL2CrossDomainMessenger.expireMessage, (_messageHash, _undeliveredAt)));

        // An already-expired message is accepted before the timestamp check.
        assert(
            ok
                == (
                    caller == L2CDM && xSender == other
                        && (expiredBefore || (sentAt != 0 && _undeliveredAt > sentAt + period))
                )
        );
        assert(l2tol2.expiredMessages(_messageHash) == (expiredBefore || ok));
        assert(l2tol2.sentMessageTimestamps(_messageHash) == sentAt);
    }

    /// @notice WITNESS (expected to FAIL): expireMessage can succeed in this harness, so
    ///         prove_expireMessage_spec is not vacuous.
    function prove_expireMessage_canSucceed_WITNESS(bytes32 _messageHash, uint256 _undeliveredAt) external {
        _etch(L2CDM, address(new AuthL2CrossDomainMessenger()));
        kevm.symbolicStorage(L2TOL2);
        vm.store(L2CDM, bytes32(uint256(0)), bytes32(uint256(uint160(kevm.freshAddress()))));
        vm.store(L2CDM, bytes32(uint256(1)), bytes32(uint256(uint160(kevm.freshAddress()))));
        vm.assume(l2tol2.sentMessageTimestamps(_messageHash) < 2 ** 64);
        vm.prank(kevm.freshAddress());
        (bool ok,) =
            L2TOL2.call(abi.encodeCall(L2ToL2CrossDomainMessenger.expireMessage, (_messageHash, _undeliveredAt)));
        assert(!ok);
    }

    /// @notice WITNESS (expected to FAIL): the early return is reachable. A message already marked
    ///         expired, with no send timestamp and an undeliveredAt of 0, is accepted again, so the
    ///         `expiredBefore` disjunct of prove_expireMessage_spec is not vacuous.
    function prove_expireMessage_alreadyExpiredCanSucceed_WITNESS(bytes32 _messageHash) external {
        _etch(L2CDM, address(new AuthL2CrossDomainMessenger()));
        kevm.symbolicStorage(L2TOL2);
        address sender = kevm.freshAddress();
        vm.store(L2CDM, bytes32(uint256(0)), bytes32(uint256(uint160(sender))));
        vm.store(L2CDM, bytes32(uint256(1)), bytes32(uint256(uint160(sender))));
        vm.assume(l2tol2.expiredMessages(_messageHash));
        vm.assume(l2tol2.sentMessageTimestamps(_messageHash) == 0);
        vm.prank(L2CDM);
        (bool ok,) = L2TOL2.call(abi.encodeCall(L2ToL2CrossDomainMessenger.expireMessage, (_messageHash, 0)));
        assert(!ok);
    }

    /// @notice The messenger set up as initialize sets it on production networks
    ///         (Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD, 8 days) has an expiry period of at least
    ///         the protocol's relay window, which op-core and kona cap at 7 days
    ///         (assumption P_contract >= W_protocol).
    function prove_expiryPeriod_atLeastProtocolWindow() external view {
        assert(l2tol2.expiryPeriod() >= 7 days);
    }
}
