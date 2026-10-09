// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Kontrol (KEVM) proofs for L1CrossDomainMessenger.relayUndeliveredMessage (property 3).
//
// Model: chain A's L1CrossDomainMessenger is the REAL implementation
// (src/L1/L1CrossDomainMessenger.sol), deployed fresh, with its storage written directly: portal
// (slot 252) = MockPortalA, systemConfig (slot 254) = MockSystemConfigA (symbolic feature flags),
// otherMessenger (slot 207) = 0x4200..0007, msgNonce (slot 205) = symbolic (< 2^240). The caller,
// the caller's portal, that portal's SystemConfig and A's ETHLockbox are mocks whose answers are
// symbolic (see ExpiryMocks0815.sol). The trusted L2 sender is read from
// Predeploys.UNDELIVERED_MESSAGE_EXPORTER, never hardcoded. Not runnable with plain forge
// (Kontrol-only cheat codes).

// Contracts
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import {
    ExpiryKontrolBaseL1,
    MockCallerMessenger,
    MockCallerPortal,
    MockSystemConfig,
    MockLockbox,
    MockPortalA,
    MockSystemConfigA
} from "test/formal/expiry/kontrol/solc0815/ExpiryMocks0815.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Features } from "src/libraries/Features.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

contract L1CrossDomainMessengerExpiryKontrol is ExpiryKontrolBaseL1 {
    uint256 internal constant SLOT_MSG_NONCE = 205;
    uint256 internal constant SLOT_OTHER_MESSENGER = 207;
    uint256 internal constant SLOT_PORTAL = 252;
    uint256 internal constant SLOT_SYSTEM_CONFIG = 254;
    uint32 internal constant EXPIRE_MESSAGE_GAS_LIMIT = 100_000;

    L1CrossDomainMessenger internal aCdm;
    MockPortalA internal aPortal;
    MockLockbox internal aLockbox;
    MockCallerMessenger internal caller;
    MockCallerPortal internal callerPortal;
    MockSystemConfig internal callerSystemConfig;
    /// @notice What caller.systemConfig() returns: a separate stand-in from the caller portal's
    ///         SystemConfig, with its own symbolic answers, so a check that reads the caller's own
    ///         systemConfig() (mutation K41) is distinguishable from the real check (a).
    MockSystemConfig internal callerOwnSystemConfig;
    MockSystemConfigA internal aSystemConfig;
    MockLockbox internal callerLockbox;

    function setUp() public {
        aCdm = new L1CrossDomainMessenger();
        aSystemConfig = new MockSystemConfigA();
        aPortal = new MockPortalA();
        aLockbox = new MockLockbox();
        caller = new MockCallerMessenger();
        callerPortal = new MockCallerPortal();
        callerSystemConfig = new MockSystemConfig();
        callerOwnSystemConfig = new MockSystemConfig();
        callerLockbox = new MockLockbox();

        vm.store(address(aCdm), bytes32(SLOT_PORTAL), bytes32(uint256(uint160(address(aPortal)))));
        vm.store(
            address(aCdm),
            bytes32(SLOT_OTHER_MESSENGER),
            bytes32(uint256(uint160(Predeploys.L2_CROSS_DOMAIN_MESSENGER)))
        );
        vm.store(address(aCdm), bytes32(SLOT_SYSTEM_CONFIG), bytes32(uint256(uint160(address(aSystemConfig)))));
        vm.store(address(aPortal), bytes32(uint256(0)), bytes32(uint256(uint160(address(aLockbox)))));
        // Pointer getters every stand-in answers (MockPortalA.systemConfigRet is its slot 8).
        vm.store(address(aPortal), bytes32(uint256(8)), bytes32(uint256(uint160(address(aSystemConfig)))));
        vm.store(address(caller), bytes32(uint256(2)), bytes32(uint256(uint160(address(callerOwnSystemConfig)))));
        vm.store(address(callerPortal), bytes32(uint256(1)), bytes32(uint256(uint160(address(callerLockbox)))));
    }

    /// @notice Leaf answers of the caller's SystemConfig and lockbox, ANY values.
    function _symbolicCallerAnswers() internal returns (address l1cdmAnswer_) {
        kevm.symbolicStorage(address(callerSystemConfig));
        kevm.symbolicStorage(address(callerOwnSystemConfig));
        kevm.symbolicStorage(address(callerLockbox));
        l1cdmAnswer_ = callerSystemConfig.l1CrossDomainMessenger();
    }

    /// @notice The L2 sender relayUndeliveredMessage trusts (check (c)), from the library constant.
    function _trustedL2Sender() internal pure returns (address) {
        return Predeploys.UNDELIVERED_MESSAGE_EXPORTER;
    }

    /// @notice A's interop gate, ANY value: SystemConfig feature flags are fully symbolic.
    function _symbolicInteropGate() internal returns (bool enabled_) {
        kevm.symbolicStorage(address(aSystemConfig));
        enabled_ = aSystemConfig.isFeatureEnabled(Features.INTEROP);
    }

    function _symbolicNonce() internal returns (uint256 nonce_) {
        nonce_ = kevm.freshUInt(30); // < 2^240: msgNonce is a uint240
        vm.store(address(aCdm), bytes32(SLOT_MSG_NONCE), bytes32(nonce_));
    }

    /// @notice relayUndeliveredMessage(H, t) called by ANY contract `caller` succeeds IFF
    ///         (g) A's SystemConfig.isFeatureEnabled(INTEROP),
    ///         (a) caller.portal().systemConfig().l1CrossDomainMessenger() == caller,
    ///         (b) A's lockbox authorizedPortals(caller.portal()), and
    ///         (c) caller.xDomainMessageSender() == Predeploys.UNDELIVERED_MESSAGE_EXPORTER.
    ///         On success, exactly one deposit is made through A's portal, from A's L1CDM, with
    ///         value 0, not a creation, to otherMessenger (0x..07), with gas baseGas(msg, 100_000)
    ///         and data relayMessage(versionedNonce, sender = A's L1CDM, target = 0x..23, 0,
    ///         100_000, expireMessage(H, t)).
    ///         Forged answers covered: every leaf answer of every stand-in is symbolic, including
    ///         those the real code does not read (A's own SystemConfig.l1CrossDomainMessenger(),
    ///         the caller portal's lockbox authorizedPortals(), the caller SystemConfig's feature
    ///         flags), so a contract reading the wrong getter is not saved by a missing function.
    ///         The pointer getters (caller.portal(), caller.systemConfig(), portal.systemConfig(),
    ///         portal.ethLockbox()) return fixed stand-ins; fully symbolic pointers are the
    ///         symbolicPortal and symbolicSystemConfig proofs.
    function prove_relayUndeliveredMessage_spec(bytes32 _messageHash, uint256 _undeliveredAt) external {
        uint256 nonce = _symbolicNonce();
        bool interop = _symbolicInteropGate();
        address xSenderAnswer = kevm.freshAddress();
        vm.store(address(caller), bytes32(uint256(0)), bytes32(uint256(uint160(address(callerPortal)))));
        vm.store(address(caller), bytes32(uint256(1)), bytes32(uint256(uint160(xSenderAnswer))));
        vm.store(address(callerPortal), bytes32(uint256(0)), bytes32(uint256(uint160(address(callerSystemConfig)))));
        address l1cdmAnswer = _symbolicCallerAnswers();
        kevm.symbolicStorage(address(aLockbox));
        bool authorized = aLockbox.authorizedPortals(address(callerPortal));

        (bool ok,) = caller.callRelay(address(aCdm), _messageHash, _undeliveredAt);

        assert(ok == (interop && l1cdmAnswer == address(caller) && authorized && xSenderAnswer == _trustedL2Sender()));

        _checkDeposit(ok, nonce, _messageHash, _undeliveredAt);
    }

    /// @notice On failure: no deposit. On success: exactly the expected deposit (see
    ///         prove_relayUndeliveredMessage_spec).
    function _checkDeposit(bool _ok, uint256 _nonce, bytes32 _messageHash, uint256 _undeliveredAt) internal view {
        (
            uint256 deposits,
            address to,
            uint256 value,
            uint64 gasLimit,
            bool isCreation,
            bytes32 dataHash,
            address depositSender,
            uint256 msgValue
        ) = aPortal.lastDeposit();

        if (!_ok) {
            assert(deposits == 0);
            return;
        }

        assert(deposits == 1);
        assert(depositSender == address(aCdm));
        assert(to == Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        assert(value == 0 && msgValue == 0);
        assert(!isCreation);
        bytes memory expireCall =
            abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (_messageHash, _undeliveredAt));
        assert(gasLimit == aCdm.baseGas(expireCall, EXPIRE_MESSAGE_GAS_LIMIT));
        assert(dataHash == keccak256(_relayCall(_nonce, expireCall)));
    }

    /// @notice relayMessage(versionedNonce, sender = A's L1CDM, target = 0x..23, value 0, minGas
    ///         100_000, _message).
    function _relayCall(uint256 _nonce, bytes memory _message) internal view returns (bytes memory) {
        return abi.encodeCall(
            ICrossDomainMessenger.relayMessage,
            (
                _nonce | (uint256(1) << 240), // Encoding.encodeVersionedNonce(msgNonce, 1)
                address(aCdm),
                Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
                uint256(0),
                uint256(EXPIRE_MESSAGE_GAS_LIMIT),
                _message
            )
        );
    }

    /// @notice WITNESS (expected to FAIL): relayUndeliveredMessage can succeed in this harness, so
    ///         prove_relayUndeliveredMessage_spec is not vacuous.
    function prove_relayUndeliveredMessage_canSucceed_WITNESS(bytes32 _messageHash, uint256 _undeliveredAt) external {
        _symbolicNonce();
        _symbolicInteropGate();
        vm.store(address(caller), bytes32(uint256(0)), bytes32(uint256(uint160(address(callerPortal)))));
        vm.store(address(caller), bytes32(uint256(1)), bytes32(uint256(uint160(kevm.freshAddress()))));
        vm.store(address(callerPortal), bytes32(uint256(0)), bytes32(uint256(uint160(address(callerSystemConfig)))));
        _symbolicCallerAnswers();
        kevm.symbolicStorage(address(aLockbox));
        (bool ok,) = caller.callRelay(address(aCdm), _messageHash, _undeliveredAt);
        assert(!ok);
    }

    // The same statement with the caller's portal P = caller.portal() and its SystemConfig
    // S = P.systemConfig() as SYMBOLIC addresses: success implies the checks evaluated on the P and
    // S the call read, whichever accounts they are. Only the mocks and the other accounts of this
    // test have code. S is read only through P, and every account other than the caller-portal
    // mock answers P.systemConfig() from its own code, so the two proofs below together cover a
    // symbolic P with a symbolic S: the first lets P be any address (the caller-portal mock then
    // answers callerSystemConfig), the second fixes P to the caller-portal mock and lets S be any
    // address. Splitting them keeps one symbolic call target per proof.

    /// @notice Symbolic P (any address), with the caller-portal mock answering callerSystemConfig.
    function prove_relayUndeliveredMessage_symbolicPortal(bytes32 _messageHash, uint256 _undeliveredAt) external {
        _symbolicNonce();
        bool interop = _symbolicInteropGate();
        address p = kevm.freshAddress();
        address xSenderAnswer = kevm.freshAddress();
        _assumeCallable(p);
        vm.store(address(caller), bytes32(uint256(0)), bytes32(uint256(uint160(p))));
        vm.store(address(caller), bytes32(uint256(1)), bytes32(uint256(uint160(xSenderAnswer))));
        vm.store(address(callerPortal), bytes32(uint256(0)), bytes32(uint256(uint160(address(callerSystemConfig)))));
        kevm.symbolicStorage(address(callerSystemConfig));
        kevm.symbolicStorage(address(aLockbox));

        (bool ok,) = caller.callRelay(address(aCdm), _messageHash, _undeliveredAt);

        if (ok) _assertChecksOn(interop, p, xSenderAnswer);
    }

    /// @notice WITNESS (expected to FAIL): with a symbolic P, relayUndeliveredMessage can succeed.
    function prove_relayUndeliveredMessage_symbolicPortalCanSucceed_WITNESS(
        bytes32 _messageHash,
        uint256 _undeliveredAt
    )
        external
    {
        _symbolicNonce();
        _symbolicInteropGate();
        address p = kevm.freshAddress();
        _assumeCallable(p);
        vm.store(address(caller), bytes32(uint256(0)), bytes32(uint256(uint160(p))));
        vm.store(address(caller), bytes32(uint256(1)), bytes32(uint256(uint160(kevm.freshAddress()))));
        vm.store(address(callerPortal), bytes32(uint256(0)), bytes32(uint256(uint160(address(callerSystemConfig)))));
        kevm.symbolicStorage(address(callerSystemConfig));
        kevm.symbolicStorage(address(aLockbox));
        (bool ok,) = caller.callRelay(address(aCdm), _messageHash, _undeliveredAt);
        assert(!ok);
    }

    /// @notice P is the caller-portal mock and S = P.systemConfig() is symbolic (any address).
    function prove_relayUndeliveredMessage_symbolicSystemConfig(
        bytes32 _messageHash,
        uint256 _undeliveredAt
    )
        external
    {
        _symbolicNonce();
        bool interop = _symbolicInteropGate();
        address s = kevm.freshAddress();
        address xSenderAnswer = kevm.freshAddress();
        _assumeCallable(s);
        vm.store(address(caller), bytes32(uint256(0)), bytes32(uint256(uint160(address(callerPortal)))));
        vm.store(address(caller), bytes32(uint256(1)), bytes32(uint256(uint160(xSenderAnswer))));
        vm.store(address(callerPortal), bytes32(uint256(0)), bytes32(uint256(uint160(s))));
        kevm.symbolicStorage(address(callerSystemConfig));
        kevm.symbolicStorage(address(aLockbox));

        (bool ok,) = caller.callRelay(address(aCdm), _messageHash, _undeliveredAt);

        if (ok) _assertChecksOn(interop, address(callerPortal), xSenderAnswer);
    }

    /// @notice WITNESS (expected to FAIL): with a symbolic S, relayUndeliveredMessage can succeed.
    function prove_relayUndeliveredMessage_symbolicSystemConfigCanSucceed_WITNESS(
        bytes32 _messageHash,
        uint256 _undeliveredAt
    )
        external
    {
        _symbolicNonce();
        _symbolicInteropGate();
        address s = kevm.freshAddress();
        _assumeCallable(s);
        vm.store(address(caller), bytes32(uint256(0)), bytes32(uint256(uint160(address(callerPortal)))));
        vm.store(address(caller), bytes32(uint256(1)), bytes32(uint256(uint160(kevm.freshAddress()))));
        vm.store(address(callerPortal), bytes32(uint256(0)), bytes32(uint256(uint160(s))));
        kevm.symbolicStorage(address(callerSystemConfig));
        kevm.symbolicStorage(address(aLockbox));
        (bool ok,) = caller.callRelay(address(aCdm), _messageHash, _undeliveredAt);
        assert(!ok);
    }

    /// @notice Harness exclusions for a symbolic call target: the cheat-code address (Kontrol
    ///         interprets calls to it as cheat codes) and the precompile range (KEVM's precompile
    ///         hooks crash on symbolic input; precompiles do not implement these getters).
    function _assumeCallable(address _a) internal pure {
        vm.assume(_a != address(vm));
        vm.assume(uint160(_a) > 0x1ff);
    }

    /// @notice The checks of a successful relayUndeliveredMessage, re-read after the call from the
    ///         portal `_p` it used (the getters are views, so they answer as they did in the call).
    function _assertChecksOn(bool _interop, address _p, address _xSenderAnswer) internal view {
        assert(_interop);
        assert(aLockbox.authorizedPortals(_p));
        assert(MockSystemConfig(MockCallerPortal(_p).systemConfig()).l1CrossDomainMessenger() == address(caller));
        assert(_xSenderAnswer == _trustedL2Sender());
    }

    /// @notice The L2ToL2CrossDomainMessenger (0x..23) is NOT a trusted sender any more: with the
    ///         interop gate on and checks (a), (b) satisfied, xDomainMessageSender == 0x..23
    ///         reverts (and no deposit is made).
    function prove_relayUndeliveredMessage_rejectsMessengerAsSender(
        bytes32 _messageHash,
        uint256 _undeliveredAt
    )
        external
    {
        _symbolicNonce();
        vm.store(address(aSystemConfig), keccak256(abi.encode(Features.INTEROP, uint256(0))), bytes32(uint256(1)));
        vm.store(address(caller), bytes32(uint256(0)), bytes32(uint256(uint160(address(callerPortal)))));
        vm.store(
            address(caller), bytes32(uint256(1)), bytes32(uint256(uint160(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)))
        );
        vm.store(address(callerPortal), bytes32(uint256(0)), bytes32(uint256(uint160(address(callerSystemConfig)))));
        vm.store(address(callerSystemConfig), bytes32(uint256(0)), bytes32(uint256(uint160(address(caller)))));
        vm.store(address(aLockbox), keccak256(abi.encode(address(callerPortal), uint256(0))), bytes32(uint256(1)));
        assert(aSystemConfig.isFeatureEnabled(Features.INTEROP));
        assert(aLockbox.authorizedPortals(address(callerPortal)));

        (bool ok,) = caller.callRelay(address(aCdm), _messageHash, _undeliveredAt);

        assert(!ok);
        _checkDeposit(ok, 0, _messageHash, _undeliveredAt);
    }

    /// @notice WITNESS (expected to FAIL): the same setup with the exporter as the L2 sender is
    ///         accepted, so the gate and checks (a), (b) are really satisfied there.
    function prove_relayUndeliveredMessage_exporterSenderAccepted_WITNESS(
        bytes32 _messageHash,
        uint256 _undeliveredAt
    )
        external
    {
        _symbolicNonce();
        vm.store(address(aSystemConfig), keccak256(abi.encode(Features.INTEROP, uint256(0))), bytes32(uint256(1)));
        vm.store(address(caller), bytes32(uint256(0)), bytes32(uint256(uint160(address(callerPortal)))));
        vm.store(
            address(caller), bytes32(uint256(1)), bytes32(uint256(uint160(Predeploys.UNDELIVERED_MESSAGE_EXPORTER)))
        );
        vm.store(address(callerPortal), bytes32(uint256(0)), bytes32(uint256(uint160(address(callerSystemConfig)))));
        vm.store(address(callerSystemConfig), bytes32(uint256(0)), bytes32(uint256(uint160(address(caller)))));
        vm.store(address(aLockbox), keccak256(abi.encode(address(callerPortal), uint256(0))), bytes32(uint256(1)));
        assert(aSystemConfig.isFeatureEnabled(Features.INTEROP));
        assert(aLockbox.authorizedPortals(address(callerPortal)));

        (bool ok,) = caller.callRelay(address(aCdm), _messageHash, _undeliveredAt);

        assert(!ok);
    }
}
