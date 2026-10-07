// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Halmos symbolic checks on the REAL L1CrossDomainMessenger (src/L1/L1CrossDomainMessenger.sol, with the real
// CrossDomainMessenger.sendMessage / relayMessage / baseGas it inherits). See README.md for exact statements.
//
// Group (4): relayUndeliveredMessage(H, t), called by `caller`, succeeds iff no dependency getter reverts and
//   (a) caller.portal().systemConfig().l1CrossDomainMessenger() == caller
//   (b) A.portal.ethLockbox().authorizedPortals(caller.portal())
//   (c) caller.xDomainMessageSender() == TRUSTED_EXPORTER (0x4200..0023 at 37b44c48c7)
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

    function set(address _messenger, bool _paused) external {
        messenger = _messenger;
        pausedFlag = _paused;
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

/// @notice Relay target that records what the calling messenger reports as xDomainMessageSender().
contract SenderProbe {
    bool public called;
    address public seen;

    fallback() external {
        called = true;
        seen = L1CrossDomainMessenger(msg.sender).xDomainMessageSender();
    }
}

contract L1CDMExpiryHalmos is Test {
    SVM internal constant svm = SVM(0xF3993A62377BCd56AE39D773740A5390411E8BC9);
    address internal constant L2_TO_L2 = 0x4200000000000000000000000000000000000023;
    address internal constant L2CDM = 0x4200000000000000000000000000000000000007;
    uint32 internal constant EXPIRE_MESSAGE_GAS_LIMIT = 100_000; // internal constant in L1CrossDomainMessenger
    /// @dev The L2 sender relayUndeliveredMessage trusts (check (c)). At 37b44c48c7 it is the
    ///      L2ToL2CrossDomainMessenger; the planned exporter predeploy changes this line (plus an interop gate).
    address internal constant TRUSTED_EXPORTER = L2_TO_L2;

    // Storage slots of L1CrossDomainMessenger (forge inspect L1CrossDomainMessenger storageLayout).
    uint256 internal constant SLOT_XDOMAIN_MSG_SENDER = 204;
    uint256 internal constant SLOT_MSG_NONCE = 205;
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

    /// @notice (4) accepts iff no dependency reverts and (a) && (b) && (c); on success exactly one deposit with the
    ///         exact fields in the header; on revert none.
    function check_relayUndelivered_iff_and_deposit(bytes32 _h, uint256 _t, bool _rvLockbox, bool _rvDeposit) public {
        uint256 nonce = _symbolicWorld(_rvLockbox, _rvDeposit);
        bool noRevert = _noRevert();
        (bool a, bool b, bool c) = _checks();

        bool ok = _relayUndeliveredFrom(address(caller), _h, _t);

        assert(ok == (noRevert && a && b && c));
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
            assert(portalA.lastGasLimit() == l1cdm.baseGas(inner, EXPIRE_MESSAGE_GAS_LIMIT));
            assert(portalA.lastDataHash() == keccak256(data));
        } else {
            assert(portalA.deposits() == 0);
        }
    }

    /// @notice A contract that claims A's own portal as its portal is rejected: A's SystemConfig names A's L1CDM,
    ///         not the claimant, so (a) fails. Its xDomainMessageSender and A's lockbox answers are symbolic.
    function check_relayUndelivered_rejectsCallerClaimingPortalA(bytes32 _h, uint256 _t) public {
        MockCallerMessenger claimant = new MockCallerMessenger(address(portalA));
        svm.enableSymbolicStorage(address(claimant));
        svm.enableSymbolicStorage(address(lockbox));
        assert(!_relayUndeliveredFrom(address(claimant), _h, _t));
    }

    /// @notice A's L1CDM calling relayUndeliveredMessage on itself while not relaying is rejected: (a) and (b) can
    ///         hold (A's SystemConfig names it, A's lockbox may authorize A's portal), but its own
    ///         xDomainMessageSender() reverts outside a relay. Inside a relay it would have to be relaying a message to
    ///         itself, which check_L1_relayMessage_rejectsSelfAndPortalTargets rules out.
    function check_relayUndelivered_rejectsSelfCallOutsideRelay(bytes32 _h, uint256 _t) public {
        svm.enableSymbolicStorage(address(lockbox));
        assert(!_relayUndeliveredFrom(address(l1cdm), _h, _t));
    }

    /// @notice NON-VACUITY (expected FAIL): accepts iff no revert && (a) && (c), i.e. the lockbox check (b) is
    ///         redundant. The counterexample is a real messenger of a chain outside A's lockbox.
    function check_FALSE_relayUndelivered_lockboxCheckRedundant(bytes32 _h, uint256 _t) public {
        _symbolicWorld(false, false);
        bool noRevert = _noRevert();
        (bool a,, bool c) = _checks();
        bool ok = _relayUndeliveredFrom(address(caller), _h, _t);
        assert(ok == (noRevert && a && c));
    }

    /// @notice NON-VACUITY (expected FAIL): relayUndeliveredMessage never produces a deposit.
    function check_FALSE_relayUndelivered_neverDeposits(bytes32 _h, uint256 _t) public {
        _symbolicWorld(false, false);
        _relayUndeliveredFrom(address(caller), _h, _t);
        assert(portalA.deposits() == 0);
    }

    // ================================================================ (7) L1 sender exclusivity

    /// @notice relayMessage never succeeds with target == A's L1CDM or A's portal, for any message, any caller
    ///         (A's portal with any l2Sender, or anyone else), any failedMessages entry for the message, paused or not.
    ///         It also never deposits.
    function check_L1_relayMessage_rejectsSelfAndPortalTargets(
        bool _fromPortal,
        address _other,
        bool _toPortal,
        uint256 _nonce,
        address _sender,
        uint256 _minGas,
        bytes calldata _message,
        bool _failed,
        address _l2Sender,
        bool _paused
    )
        public
    {
        sysCfgA.set(address(l1cdm), _paused);
        portalA.setL2Sender(_l2Sender);
        address target = _toPortal ? address(portalA) : address(l1cdm);
        bytes32 vh = Hashing.hashCrossDomainMessageV1(_nonce, _sender, target, 0, _minGas, _message);
        vm.store(address(l1cdm), keccak256(abi.encode(vh, SLOT_FAILED_MESSAGES)), bytes32(uint256(_failed ? 1 : 0)));
        assert(l1cdm.failedMessages(vh) == _failed); // slot check

        vm.prank(_fromPortal ? address(portalA) : _other);
        (bool ok,) =
            address(l1cdm).call(abi.encodeCall(l1cdm.relayMessage, (_nonce, _sender, target, 0, _minGas, _message)));
        assert(!ok);
        assert(portalA.deposits() == 0);
    }

    /// @notice xDomainMessageSender() reverts when no message is being relayed.
    function check_L1_xDomainMessageSender_revertsOutsideRelay() public view {
        (bool ok,) = address(l1cdm).staticcall(abi.encodeCall(l1cdm.xDomainMessageSender, ()));
        assert(!ok);
    }

    /// @notice During relayMessage (delivered by A's portal with l2Sender == the L2CrossDomainMessenger), the target
    ///         observes xDomainMessageSender() == the relayed message's sender, and afterwards it reverts again.
    function check_L1_xDomainMessageSender_isRelayedSender(
        uint256 _nonce,
        address _sender,
        uint256 _minGas,
        bytes calldata _message
    )
        public
    {
        vm.assume(_sender != Constants.DEFAULT_L2_SENDER);
        portalA.setL2Sender(L2CDM);
        SenderProbe probe = new SenderProbe();
        vm.prank(address(portalA));
        (bool ok,) = address(l1cdm)
            .call(abi.encodeCall(l1cdm.relayMessage, (_nonce, _sender, address(probe), 0, _minGas, _message)));
        if (ok && probe.called()) assert(probe.seen() == _sender);
        (bool okAfter,) = address(l1cdm).staticcall(abi.encodeCall(l1cdm.xDomainMessageSender, ()));
        assert(!okAfter);
    }

    /// @notice NON-VACUITY (expected FAIL): the probe is never reached (so the check above is not vacuous).
    function check_FALSE_L1_probeNeverCalled(uint256 _nonce, address _sender, uint256 _minGas) public {
        portalA.setL2Sender(L2CDM);
        SenderProbe probe = new SenderProbe();
        vm.prank(address(portalA));
        (bool ok,) =
            address(l1cdm).call(abi.encodeCall(l1cdm.relayMessage, (_nonce, _sender, address(probe), 0, _minGas, "")));
        ok;
        assert(!probe.called());
    }

    /// @notice sendMessage called by anyone other than A's L1CDM produces a deposit whose message sender is that
    ///         caller: data == relayMessage(messageNonce(), caller, target, 0, minGas, message). So A's L1CDM is the
    ///         sender of an L1->L2 message only when it calls sendMessage itself, which its code does only in
    ///         relayUndeliveredMessage (`this.sendMessage`, code inspection) or via a relayed call to itself (ruled out
    ///         above).
    function check_L1_sendMessage_senderFieldIsCaller(
        address _from,
        address _target,
        uint32 _minGas,
        bytes calldata _message
    )
        public
    {
        vm.assume(_from != address(l1cdm));
        uint256 nonce = l1cdm.messageNonce();
        vm.prank(_from);
        l1cdm.sendMessage(_target, _message, _minGas);
        assert(portalA.deposits() == 1);
        assert(portalA.lastFrom() == address(l1cdm));
        assert(
            portalA.lastDataHash()
                == keccak256(
                    abi.encodeWithSelector(
                        CrossDomainMessenger.relayMessage.selector,
                        nonce,
                        _from,
                        _target,
                        uint256(0),
                        uint256(_minGas),
                        _message
                    )
                )
        );
    }
}
