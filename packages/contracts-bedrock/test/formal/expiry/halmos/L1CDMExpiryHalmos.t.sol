// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Halmos symbolic checks on the REAL L1CrossDomainMessenger (src/L1/L1CrossDomainMessenger.sol, with the real
// CrossDomainMessenger.sendMessage / relayMessage / baseGas it inherits). See README.md for exact statements.
//
// Group (4): relayUndeliveredMessage(H, t), called by `caller`, succeeds iff no dependency getter reverts and
//   (a) caller.portal().systemConfig().l1CrossDomainMessenger() == caller
//   (b) A.portal.ethLockbox().authorizedPortals(caller.portal())
//   (0) A's SystemConfig has the INTEROP feature enabled
//   (c) caller.xDomainMessageSender() == TRUSTED_EXPORTER (Predeploys.UNDELIVERED_MESSAGE_EXPORTER)
// and on success A's portal receives EXACTLY ONE depositTransaction, from A's L1CrossDomainMessenger, with
//   _to = 0x4200..0007, _value = 0, _isCreation = false, _gasLimit = baseGas(expireMessage(H, t), 100_000),
//   _data = relayMessage(messageNonce(), A's L1CDM, 0x4200..0023, 0, 100_000, expireMessage(H, t)).
// Group (7) (L1 sender exclusivity, the parts that live in L1CrossDomainMessenger):
//   - relayMessage never calls A's L1CDM itself or A's portal (any caller, any message, any failed-message state);
//   - so A's L1CDM can be the caller of relayUndeliveredMessage only while relaying, which it cannot do to itself;
//     and called directly while not relaying, check (c) reverts;
//   - xDomainMessageSender() reverts outside relayMessage and, inside it, equals the relayed message's sender;
//   - sendMessage from any caller other than A's L1CDM carries that caller as the message sender.
//
// Mocks (oracles): every getter is guarded by a revert flag; flags and returned values come from symbolic storage
// (svm.enableSymbolicStorage) or symbolic check parameters. Addresses of the mock contracts themselves are fixed
// (see README "Topologies explored / not explored").

import { Test } from "forge-std/Test.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L2CrossDomainMessenger } from "src/L2/L2CrossDomainMessenger.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

interface SVM {
    function enableSymbolicStorage(address) external;
    function createUint256(string memory) external returns (uint256);
}

/// @notice A SystemConfig stand-in: l1CrossDomainMessenger() and paused(), each reverting when `rv` is set.
contract MockSystemConfig {
    address internal messenger;
    bool public rv;
    bool internal pausedFlag;
    bool public interop;

    function set(address _messenger, bool _paused) external {
        messenger = _messenger;
        pausedFlag = _paused;
    }

    function setInterop(bool _interop) external {
        interop = _interop;
    }

    /// @dev Only the INTEROP feature can be enabled here (the real SystemConfig keeps a feature map).
    function isFeatureEnabled(bytes32 _feature) external view returns (bool) {
        require(!rv);
        return _feature == "INTEROP" && interop;
    }

    function l1CrossDomainMessenger() external view returns (address) {
        require(!rv);
        return messenger;
    }

    function paused() external view returns (bool) {
        require(!rv);
        return pausedFlag;
    }
}

contract MockCallerPortal {
    address internal immutable sc;
    bool public rv;

    constructor(address _systemConfig) {
        sc = _systemConfig;
    }

    function systemConfig() external view returns (address) {
        require(!rv);
        return sc;
    }
}

contract MockCallerMessenger {
    address internal immutable portalAddr;
    bool public rvPortal;
    bool public rvX;
    address internal xSender;

    constructor(address _portal) {
        portalAddr = _portal;
    }

    function portal() external view returns (address) {
        require(!rvPortal);
        return portalAddr;
    }

    function xDomainMessageSender() external view returns (address) {
        require(!rvX);
        return xSender;
    }
}

contract MockLockbox {
    bool public rv;
    mapping(address => bool) internal authorized;

    function authorizedPortals(address _p) external view returns (bool) {
        require(!rv);
        return authorized[_p];
    }
}

/// @notice A's OptimismPortal stand-in. Records depositTransaction calls; getters and the deposit can be made to
///         revert (flags set by the test from symbolic values).
contract MockPortalA {
    address internal immutable lockbox;
    address internal immutable sysCfg;

    bool public rvLockbox;
    bool public rvDeposit;
    address internal l2SenderValue;

    uint256 public deposits;
    address public lastFrom;
    address public lastTo;
    uint256 public lastValue;
    uint256 public lastMsgValue;
    uint64 public lastGasLimit;
    bool public lastIsCreation;
    bytes32 public lastDataHash;

    constructor(address _lockbox, address _sysCfg) {
        lockbox = _lockbox;
        sysCfg = _sysCfg;
    }

    function setFlags(bool _rvLockbox, bool _rvDeposit) external {
        rvLockbox = _rvLockbox;
        rvDeposit = _rvDeposit;
    }

    function setL2Sender(address _l2Sender) external {
        l2SenderValue = _l2Sender;
    }

    function ethLockbox() external view returns (address) {
        require(!rvLockbox);
        return lockbox;
    }

    function systemConfig() external view returns (address) {
        return sysCfg;
    }

    function l2Sender() external view returns (address) {
        return l2SenderValue;
    }

    function depositTransaction(
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    )
        external
        payable
    {
        require(!rvDeposit);
        deposits++;
        lastFrom = msg.sender;
        lastTo = _to;
        lastValue = _value;
        lastMsgValue = msg.value;
        lastGasLimit = _gasLimit;
        lastIsCreation = _isCreation;
        lastDataHash = keccak256(_data);
    }
}

/// @notice Relay target for the CrossDomainMessenger gate checks. It has NO external functions, so any relayed
///         calldata reaches the recorder. Records (read with vm.load): slot 0 = 1 if it ran, slot 1 = ETH received,
///         slot 2 = 1 if the caller's xDomainMessageSender() succeeded (low-level call: a getter revert is recorded,
///         not unwound), slot 3 = what it returned.
contract GateProbe {
    uint256 internal called;
    uint256 internal value;
    uint256 internal getterOk;
    uint256 internal seen;

    function _record() internal {
        called = 1;
        value = msg.value;
        (bool ok, bytes memory ret) = msg.sender.staticcall(abi.encodeWithSignature("xDomainMessageSender()"));
        if (ok && ret.length == 32) {
            getterOk = 1;
            address a = abi.decode(ret, (address));
            seen = uint256(uint160(a));
        }
    }

    receive() external payable {
        _record();
    }

    fallback() external payable {
        _record();
    }
}

/// @notice Symbolic inputs for one CrossDomainMessenger.relayMessage call.
struct GateIn {
    bool fromPortal; // caller = the messenger's portal (L1) / an aliased L1 address (L2); else `other`
    address other;
    address l2Sender; // L1 only: portal.l2Sender()
    bool failed; // failedMessages[vh] before
    bool succ; // successfulMessages[vh] before
    bool paused; // L1 only
    uint256 nonce;
    address sender;
    uint256 value; // the message's _value
    uint256 mv; // msg.value when the caller is `other`
    uint256 minGas;
    uint256 bal; // messenger's ETH balance before
}

contract L1CDMExpiryHalmos is Test {
    SVM internal constant svm = SVM(0xF3993A62377BCd56AE39D773740A5390411E8BC9);
    address internal constant L2_TO_L2 = 0x4200000000000000000000000000000000000023;
    address internal constant L2CDM = 0x4200000000000000000000000000000000000007;
    uint32 internal constant EXPIRE_MESSAGE_GAS_LIMIT = 100_000; // internal constant in L1CrossDomainMessenger
    /// @dev The L2 sender relayUndeliveredMessage trusts (check (c)): the UndeliveredMessageExporter (dd0931a540).
    address internal immutable TRUSTED_EXPORTER = Predeploys.UNDELIVERED_MESSAGE_EXPORTER;

    // Storage slots of L1CrossDomainMessenger (forge inspect L1CrossDomainMessenger storageLayout).
    uint256 internal constant SLOT_XDOMAIN_MSG_SENDER = 204;
    uint256 internal constant SLOT_MSG_NONCE = 205;
    uint256 internal constant SLOT_SUCCESSFUL_MESSAGES = 203;
    uint256 internal constant SLOT_FAILED_MESSAGES = 206;
    uint256 internal constant SLOT_OTHER_MESSENGER = 207;
    uint256 internal constant SLOT_PORTAL = 252;
    uint256 internal constant SLOT_SYSTEM_CONFIG = 254;

    L1CrossDomainMessenger internal l1cdm;
    MockSystemConfig internal sysCfg; // the caller chain's SystemConfig
    MockCallerPortal internal callerPortal;
    MockCallerMessenger internal caller;
    MockLockbox internal lockbox; // A's lockbox
    MockSystemConfig internal sysCfgA; // A's SystemConfig: names A's L1CDM
    MockPortalA internal portalA;

    function setUp() public {
        sysCfg = new MockSystemConfig();
        callerPortal = new MockCallerPortal(address(sysCfg));
        caller = new MockCallerMessenger(address(callerPortal));
        lockbox = new MockLockbox();
        sysCfgA = new MockSystemConfig();
        portalA = new MockPortalA(address(lockbox), address(sysCfgA));

        l1cdm = new L1CrossDomainMessenger();
        vm.store(address(l1cdm), bytes32(SLOT_PORTAL), bytes32(uint256(uint160(address(portalA)))));
        vm.store(address(l1cdm), bytes32(SLOT_SYSTEM_CONFIG), bytes32(uint256(uint160(address(sysCfgA)))));
        vm.store(address(l1cdm), bytes32(SLOT_OTHER_MESSENGER), bytes32(uint256(uint160(L2CDM))));
        vm.store(
            address(l1cdm), bytes32(SLOT_XDOMAIN_MSG_SENDER), bytes32(uint256(uint160(Constants.DEFAULT_L2_SENDER)))
        );
        sysCfgA.set(address(l1cdm), false);
        // The slot numbers are right iff the getters read back what we stored.
        assert(address(l1cdm.portal()) == address(portalA));
        assert(address(l1cdm.systemConfig()) == address(sysCfgA));
        assert(address(l1cdm.otherMessenger()) == L2CDM);
        assert(L2_TO_L2 == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER && L2CDM == Predeploys.L2_CROSS_DOMAIN_MESSENGER);
    }

    // ---------------------------------------------------------------- helpers

    /// @dev Symbolic answers + revert flags for the caller side and A's lockbox; symbolic L1CDM nonce. Returns the
    ///      versioned nonce the next L1CDM message will carry (messageNonce(), read before the call).
    function _symbolicWorld(bool _rvLockbox, bool _rvDeposit) internal returns (uint256 versionedNonce_) {
        sysCfgA.setInterop(svm.createUint256("interop") & 1 == 1); // A's INTEROP feature: symbolic
        svm.enableSymbolicStorage(address(sysCfg));
        svm.enableSymbolicStorage(address(callerPortal));
        svm.enableSymbolicStorage(address(caller));
        svm.enableSymbolicStorage(address(lockbox));
        portalA.setFlags(_rvLockbox, _rvDeposit);
        uint256 rawNonce = svm.createUint256("msgNonce") & type(uint240).max;
        vm.store(address(l1cdm), bytes32(SLOT_MSG_NONCE), bytes32(rawNonce));
        versionedNonce_ = l1cdm.messageNonce();
        assert(versionedNonce_ == (uint256(1) << 240 | rawNonce)); // MESSAGE_VERSION = 1
    }

    function _relayUndeliveredFrom(address _from, bytes32 _h, uint256 _t) internal returns (bool ok_) {
        vm.prank(_from);
        (ok_,) = address(l1cdm).call(abi.encodeCall(l1cdm.relayUndeliveredMessage, (_h, _t)));
    }

    /// @dev No dependency getter reverts (each flag read without calling the guarded getter).
    function _noRevert() internal view returns (bool) {
        return !caller.rvPortal() && !callerPortal.rv() && !sysCfg.rv() && !portalA.rvLockbox() && !lockbox.rv()
            && !caller.rvX() && !portalA.rvDeposit();
    }

    /// @dev The three checks, each evaluated only when its getters do not revert.
    function _checks() internal view returns (bool a_, bool b_, bool c_) {
        a_ = !sysCfg.rv() && sysCfg.l1CrossDomainMessenger() == address(caller);
        b_ = !lockbox.rv() && lockbox.authorizedPortals(address(callerPortal));
        c_ = !caller.rvX() && caller.xDomainMessageSender() == TRUSTED_EXPORTER;
    }

    // ================================================================ (4) relayUndeliveredMessage

    /// @notice (4) accepts iff A's INTEROP feature is on, no dependency reverts and (a) && (b) && (c), where (c) trusts
    ///         the UndeliveredMessageExporter; on success exactly one deposit with the exact fields in the header; on
    ///         revert none.
    function check_relayUndelivered_iff_and_deposit(bytes32 _h, uint256 _t, bool _rvLockbox, bool _rvDeposit) public {
        uint256 nonce = _symbolicWorld(_rvLockbox, _rvDeposit);
        bool interop = sysCfgA.interop();
        bool noRevert = _noRevert();
        (bool a, bool b, bool c) = _checks();

        bool ok = _relayUndeliveredFrom(address(caller), _h, _t);

        assert(ok == (interop && noRevert && a && b && c));
        if (ok) {
            bytes memory inner = abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (_h, _t));
            bytes memory data = abi.encodeWithSelector(
                CrossDomainMessenger.relayMessage.selector,
                nonce,
                address(l1cdm),
                L2_TO_L2,
                uint256(0),
                uint256(EXPIRE_MESSAGE_GAS_LIMIT),
                inner
            );
            assert(portalA.deposits() == 1);
            assert(portalA.lastFrom() == address(l1cdm));
            assert(portalA.lastTo() == L2CDM);
            assert(portalA.lastValue() == 0);
            assert(portalA.lastMsgValue() == 0);
            assert(!portalA.lastIsCreation());
            assert(portalA.lastGasLimit() == _baseGasSpec(inner.length, EXPIRE_MESSAGE_GAS_LIMIT));
            assert(portalA.lastDataHash() == keccak256(data));
        } else {
            assert(portalA.deposits() == 0);
        }
    }

    /// @notice The L2ToL2CrossDomainMessenger is no longer a trusted sender: a caller reporting xDomainMessageSender ==
    ///         0x..23 is rejected even when everything else (interop, (a), (b), no reverts) holds.
    function check_relayUndelivered_rejectsL2ToL2AsSender(bytes32 _h, uint256 _t) public {
        _symbolicWorld(false, false);
        // MockCallerMessenger packs rvPortal (byte 0), rvX (byte 1) and xSender (bytes 2..21) into slot 0.
        vm.store(address(caller), bytes32(0), bytes32(uint256(uint160(L2_TO_L2)) << 16));
        assert(!caller.rvPortal() && !caller.rvX() && caller.xDomainMessageSender() == L2_TO_L2); // slot check
        assert(!_relayUndeliveredFrom(address(caller), _h, _t));
    }

    /// @notice NON-VACUITY (expected FAIL): the INTEROP gate is redundant (accepts iff no revert && (a) && (b) && (c)).
    function check_FALSE_relayUndelivered_interopGateRedundant(bytes32 _h, uint256 _t) public {
        _symbolicWorld(false, false);
        bool noRevert = _noRevert();
        (bool a, bool b, bool c) = _checks();
        bool ok = _relayUndeliveredFrom(address(caller), _h, _t);
        assert(ok == (noRevert && a && b && c));
    }

    /// @notice A contract that claims A's own portal as its portal is rejected: A's SystemConfig names A's L1CDM,
    ///         not the claimant, so (a) fails. Its xDomainMessageSender and A's lockbox answers are symbolic.
    function check_relayUndelivered_rejectsCallerClaimingPortalA(bytes32 _h, uint256 _t, bool _interop) public {
        sysCfgA.setInterop(_interop);
        MockCallerMessenger claimant = new MockCallerMessenger(address(portalA));
        svm.enableSymbolicStorage(address(claimant));
        svm.enableSymbolicStorage(address(lockbox));
        assert(!_relayUndeliveredFrom(address(claimant), _h, _t));
    }

    /// @notice A's L1CDM calling relayUndeliveredMessage on itself while not relaying is rejected: (a) and (b) can
    ///         hold (A's SystemConfig names it, A's lockbox may authorize A's portal), but its own
    ///         xDomainMessageSender() reverts outside a relay. Inside a relay it would have to be relaying a message to
    ///         itself, which check_L1_relayMessage_rejectsSelfAndPortalTargets rules out.
    function check_relayUndelivered_rejectsSelfCallOutsideRelay(bytes32 _h, uint256 _t, bool _interop) public {
        sysCfgA.setInterop(_interop);
        svm.enableSymbolicStorage(address(lockbox));
        assert(!_relayUndeliveredFrom(address(l1cdm), _h, _t));
    }

    /// @notice NON-VACUITY (expected FAIL): accepts iff no revert && (a) && (c), i.e. the lockbox check (b) is
    ///         redundant. The counterexample is a real messenger of a chain outside A's lockbox.
    function check_FALSE_relayUndelivered_lockboxCheckRedundant(bytes32 _h, uint256 _t) public {
        _symbolicWorld(false, false);
        bool interop = sysCfgA.interop();
        bool noRevert = _noRevert();
        (bool a,, bool c) = _checks();
        bool ok = _relayUndeliveredFrom(address(caller), _h, _t);
        assert(ok == (interop && noRevert && a && c));
    }

    /// @notice NON-VACUITY (expected FAIL): relayUndeliveredMessage never produces a deposit.
    function check_FALSE_relayUndelivered_neverDeposits(bytes32 _h, uint256 _t) public {
        _symbolicWorld(false, false);
        _relayUndeliveredFrom(address(caller), _h, _t);
        assert(portalA.deposits() == 0);
    }

    // ================================================================ (7) L1 sender exclusivity

    function _flags(address _m, bytes32 _vh, bool _failed, bool _succ) internal {
        vm.store(_m, keccak256(abi.encode(_vh, SLOT_FAILED_MESSAGES)), bytes32(uint256(_failed ? 1 : 0)));
        vm.store(_m, keccak256(abi.encode(_vh, SLOT_SUCCESSFUL_MESSAGES)), bytes32(uint256(_succ ? 1 : 0)));
        assert(CrossDomainMessenger(_m).failedMessages(_vh) == _failed); // slot check
        assert(CrossDomainMessenger(_m).successfulMessages(_vh) == _succ); // slot check
    }

    function _bounds(GateIn memory _g) internal pure {
        vm.assume(_g.value <= 1 << 128 && _g.mv <= 1 << 128 && _g.bal <= 1 << 128);
    }

    /// @dev Calls relayMessage on A's L1CDM from the portal (msg.value = message value) or from `other` (msg.value =
    ///      mv), with the given failed/successful flags and messenger balance.
    function _l1Relay(
        GateIn memory _g,
        address _target,
        bytes memory _message
    )
        internal
        returns (bool ok_, bytes32 vh_)
    {
        sysCfgA.set(address(l1cdm), _g.paused);
        portalA.setL2Sender(_g.l2Sender);
        vh_ = Hashing.hashCrossDomainMessageV1(_g.nonce, _g.sender, _target, _g.value, _g.minGas, _message);
        _flags(address(l1cdm), vh_, _g.failed, _g.succ);
        vm.deal(address(l1cdm), _g.bal);
        address from = _g.fromPortal ? address(portalA) : _g.other;
        uint256 v = _g.fromPortal ? _g.value : _g.mv;
        vm.deal(from, v);
        vm.prank(from);
        (ok_,) = address(l1cdm).call{ value: v }(
            abi.encodeCall(l1cdm.relayMessage, (_g.nonce, _g.sender, _target, _g.value, _g.minGas, _message))
        );
    }

    /// @notice relayMessage never succeeds with target == A's L1CDM or A's portal, for any message (any _value), any
    ///         caller (A's portal with any l2Sender and msg.value = _value, or anyone else with any msg.value), any
    ///         failed/successful flags for the message, any messenger balance, paused or not. It also never deposits.
    function check_L1_relayMessage_rejectsSelfAndPortalTargets(
        GateIn memory _g,
        bool _toPortal,
        bytes calldata _message
    )
        public
    {
        _bounds(_g);
        vm.assume(_g.other != address(l1cdm));
        (bool ok,) = _l1Relay(_g, _toPortal ? address(portalA) : address(l1cdm), _message);
        assert(!ok);
        assert(portalA.deposits() == 0);
    }

    /// @notice The CrossDomainMessenger relay GATE on the real L1CrossDomainMessenger, with symbolic caller, l2Sender,
    ///         failed/successful flags, paused, nonce, sender (0x..23 included), _value, msg.value and balance:
    ///           - the target runs only if not paused, the message was not already successful, and
    ///             (caller == portal && portal.l2Sender() == otherMessenger) || failedMessages[vh];
    ///           - whenever it runs it receives exactly _value and xDomainMessageSender() returns exactly _sender
    ///             (observed through a low-level call: a revert would be recorded, not hidden);
    ///           - delivery is not skipped: if the message becomes successful, the target ran;
    ///           - afterwards xDomainMessageSender() reverts again.
    function check_L1_relayGate_and_delivery(GateIn memory _g, bytes calldata _message) public {
        _bounds(_g);
        vm.assume(_g.other != address(portalA) && _g.other != address(l1cdm));
        vm.assume(_g.sender != Constants.DEFAULT_L2_SENDER);
        address p = address(new GateProbe());

        (, bytes32 vh) = _l1Relay(_g, p, _message);

        bool gate = (_g.fromPortal && _g.l2Sender == L2CDM) || _g.failed;
        if (uint256(vm.load(p, 0)) == 1) {
            assert(!_g.paused && !_g.succ && gate);
            assert(uint256(vm.load(p, bytes32(uint256(1)))) == _g.value);
            assert(uint256(vm.load(p, bytes32(uint256(2)))) == 1);
            assert(uint256(vm.load(p, bytes32(uint256(3)))) == uint256(uint160(_g.sender)));
        }
        if (l1cdm.successfulMessages(vh) && !_g.succ) assert(uint256(vm.load(p, 0)) == 1);
        (bool okAfter,) = address(l1cdm).staticcall(abi.encodeCall(l1cdm.xDomainMessageSender, ()));
        assert(!okAfter);
    }

    /// @notice NON-VACUITY (expected FAIL): a portal-delivered message from the L2CrossDomainMessenger never reaches
    ///         its target.
    function check_FALSE_L1_probeNeverCalled(GateIn memory _g) public {
        _bounds(_g);
        _g.fromPortal = true;
        _g.l2Sender = L2CDM;
        address p = address(new GateProbe());
        _l1Relay(_g, p, "");
        assert(uint256(vm.load(p, 0)) == 0);
    }

    /// @notice xDomainMessageSender() reverts when no message is being relayed.
    function check_L1_xDomainMessageSender_revertsOutsideRelay() public view {
        (bool ok,) = address(l1cdm).staticcall(abi.encodeCall(l1cdm.xDomainMessageSender, ()));
        assert(!ok);
    }

    /// @notice baseGas, computed independently of the contract (CrossDomainMessenger.baseGas formula restated).
    function _baseGasSpec(uint256 _len, uint32 _minGas) internal pure returns (uint64) {
        uint256 exec = 200_000 + 40_000 + 40_000 + 5_000 + (uint256(_minGas) * 64) / 63;
        uint256 size = _len + 260;
        uint256 a = exec + size * 16;
        uint256 b = size * 40;
        return uint64(21_000 + (a > b ? a : b));
    }

    /// @notice sendMessage called by anyone other than A's L1CDM, with any msg.value, produces exactly one deposit,
    ///         sent by A's L1CDM, of msg.value to the L2CrossDomainMessenger, gas = baseGas (independent formula), data
    ///         relayMessage(messageNonce(), CALLER, target, msg.value, minGas, message). So A's L1CDM is the sender of
    ///         an L1->L2 message only when it calls sendMessage itself, which its code does only in
    ///         relayUndeliveredMessage (`this.sendMessage`, code inspection) or via a relayed call to itself (ruled
    ///         out above).
    function check_L1_sendMessage_senderFieldIsCaller(
        address _from,
        address _target,
        uint32 _minGas,
        uint256 _mv,
        bytes calldata _message
    )
        public
    {
        vm.assume(_from != address(l1cdm) && _mv <= 1 << 128);
        uint256 nonce = l1cdm.messageNonce();
        vm.deal(_from, _mv);
        vm.prank(_from);
        l1cdm.sendMessage{ value: _mv }(_target, _message, _minGas);
        assert(portalA.deposits() == 1);
        assert(portalA.lastFrom() == address(l1cdm));
        assert(portalA.lastTo() == L2CDM);
        assert(portalA.lastValue() == _mv && portalA.lastMsgValue() == _mv);
        assert(portalA.lastGasLimit() == _baseGasSpec(_message.length, _minGas));
        assert(
            portalA.lastDataHash()
                == keccak256(
                    abi.encodeWithSelector(
                        CrossDomainMessenger.relayMessage.selector,
                        nonce,
                        _from,
                        _target,
                        _mv,
                        uint256(_minGas),
                        _message
                    )
                )
        );
    }
}

/// @notice The same relay gate on the REAL L2CrossDomainMessenger (etched at 0x4200..0007, otherMessenger = A's L1CDM),
///         which expireMessage's authorization relies on: the L2ToL2 messenger accepts expireMessage only from 0x..07
///         while xDomainMessageSender() == otherMessenger(). These checks show the L2CDM reports sender s to a target
///         only when relaying a message (deposited by the aliased otherMessenger, or a replay of a failed one) whose
///         _sender field is s. That the deposit's _sender field is A's L1CDM only for relayUndeliveredMessage is
///         check_L1_sendMessage_senderFieldIsCaller + check_L1_relayMessage_rejectsSelfAndPortalTargets; that the
///         portal derives the deposit's L2 `from` by aliasing the L1 caller is delegated (OptimismPortal2).
contract L2CDMGateHalmos is Test {
    address internal constant L2CDM = 0x4200000000000000000000000000000000000007;
    address internal constant PASSER = 0x4200000000000000000000000000000000000016;
    address internal constant A_L1CDM = address(0xA11CE); // A's L1CrossDomainMessenger (otherMessenger)
    uint256 internal constant SLOT_SUCCESSFUL_MESSAGES = 203;
    uint256 internal constant SLOT_XDOMAIN_MSG_SENDER = 204;
    uint256 internal constant SLOT_FAILED_MESSAGES = 206;
    uint256 internal constant SLOT_OTHER_MESSENGER = 207;

    L2CrossDomainMessenger internal l2cdm = L2CrossDomainMessenger(L2CDM);

    function setUp() public {
        assert(L2CDM == Predeploys.L2_CROSS_DOMAIN_MESSENGER && PASSER == Predeploys.L2_TO_L1_MESSAGE_PASSER);
        vm.etch(L2CDM, address(new L2CrossDomainMessenger()).code);
        vm.store(L2CDM, bytes32(SLOT_OTHER_MESSENGER), bytes32(uint256(uint160(A_L1CDM))));
        vm.store(L2CDM, bytes32(SLOT_XDOMAIN_MSG_SENDER), bytes32(uint256(uint160(Constants.DEFAULT_L2_SENDER))));
        assert(address(l2cdm.otherMessenger()) == A_L1CDM);
    }

    function _l2Relay(
        GateIn memory _g,
        address _target,
        bytes memory _message
    )
        internal
        returns (bool ok_, bytes32 vh_)
    {
        vm.assume(_g.value <= 1 << 128 && _g.mv <= 1 << 128 && _g.bal <= 1 << 128);
        vh_ = Hashing.hashCrossDomainMessageV1(_g.nonce, _g.sender, _target, _g.value, _g.minGas, _message);
        vm.store(L2CDM, keccak256(abi.encode(vh_, SLOT_FAILED_MESSAGES)), bytes32(uint256(_g.failed ? 1 : 0)));
        vm.store(L2CDM, keccak256(abi.encode(vh_, SLOT_SUCCESSFUL_MESSAGES)), bytes32(uint256(_g.succ ? 1 : 0)));
        assert(l2cdm.failedMessages(vh_) == _g.failed && l2cdm.successfulMessages(vh_) == _g.succ); // slot check
        vm.deal(L2CDM, _g.bal);
        address from = _g.fromPortal ? AddressAliasHelper.applyL1ToL2Alias(A_L1CDM) : _g.other;
        uint256 v = _g.fromPortal ? _g.value : _g.mv;
        vm.deal(from, v);
        vm.prank(from);
        (ok_,) = L2CDM.call{ value: v }(
            abi.encodeCall(l2cdm.relayMessage, (_g.nonce, _g.sender, _target, _g.value, _g.minGas, _message))
        );
    }

    /// @notice Gate on the real L2CrossDomainMessenger: the target runs only if the message was not already successful
    ///         and (caller == alias(otherMessenger)) || failedMessages[vh]; when it runs it receives exactly _value and
    ///         xDomainMessageSender() returns exactly _sender; a message that becomes successful was delivered;
    ///         afterwards xDomainMessageSender() reverts again.
    function check_L2_relayGate_and_delivery(GateIn memory _g, bytes calldata _message) public {
        address aliased = AddressAliasHelper.applyL1ToL2Alias(A_L1CDM);
        vm.assume(_g.other != aliased && _g.other != L2CDM);
        vm.assume(_g.sender != Constants.DEFAULT_L2_SENDER);
        address p = address(new GateProbe());

        (, bytes32 vh) = _l2Relay(_g, p, _message);

        if (uint256(vm.load(p, 0)) == 1) {
            assert(!_g.succ && (_g.fromPortal || _g.failed));
            assert(uint256(vm.load(p, bytes32(uint256(1)))) == _g.value);
            assert(uint256(vm.load(p, bytes32(uint256(2)))) == 1);
            assert(uint256(vm.load(p, bytes32(uint256(3)))) == uint256(uint160(_g.sender)));
        }
        if (l2cdm.successfulMessages(vh) && !_g.succ) assert(uint256(vm.load(p, 0)) == 1);
        (bool okAfter,) = L2CDM.staticcall(abi.encodeCall(l2cdm.xDomainMessageSender, ()));
        assert(!okAfter);
    }

    /// @notice relayMessage on the real L2CrossDomainMessenger never succeeds with target == itself or the
    ///         L2ToL1MessagePasser (any caller, value, flags, message).
    function check_L2_relayMessage_rejectsSelfAndPasser(
        GateIn memory _g,
        bool _toPasser,
        bytes calldata _message
    )
        public
    {
        vm.assume(_g.other != L2CDM);
        address target = _toPasser ? PASSER : L2CDM;
        (bool ok,) = _l2Relay(_g, target, _message);
        assert(!ok);
    }

    /// @notice NON-VACUITY (expected FAIL): a message deposited by A's L1CDM never reaches its target.
    function check_FALSE_L2_probeNeverCalled(GateIn memory _g) public {
        _g.fromPortal = true;
        address p = address(new GateProbe());
        _l2Relay(_g, p, "");
        assert(uint256(vm.load(p, 0)) == 0);
    }
}
