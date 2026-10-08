// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Phase 2 (3): whole-contract reachability on the REAL L1CrossDomainMessenger. For sequences of calls to every
// non-view entry point (sendMessage, relayMessage, relayUndeliveredMessage, initialize) with symbolic arguments, plus
// unknown selectors, from symbolic callers with symbolic msg.value: A's L1CDM is the SENDER in the relayMessage
// envelope of a deposit it makes ONLY in a relayUndeliveredMessage step; a deposit made in a sendMessage step carries
// that step's caller as the envelope sender;
// nothing else deposits; at most one deposit per step. (On chain the L1CDM sits behind a ResolvedDelegateProxy, which
// only forwards; it is not modelled here: the implementation is called directly.)

import { Test } from "test/setup/Test.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { Constants } from "src/libraries/Constants.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { MockSystemConfig, MockCallerPortal, MockCallerMessenger, MockLockbox } from "./L1CDMExpiryHalmos.t.sol";

interface SVMReachL1 {
    function enableSymbolicStorage(address) external;
}

/// @notice A's portal: records every depositTransaction and the envelope SENDER (relayMessage's 2nd argument, the word
///         at byte offset 36 of _data). l2Sender() is symbolic (symbolic storage).
contract PortalRecorder {
    address internal immutable lockbox;
    address internal immutable sysCfg;
    address public l2Sender; // symbolic
    uint256 public deposits;
    uint256 public selfSenderDeposits; // envelope sender == msg.sender (the L1CDM itself)
    address public lastEnvelopeSender;
    address public lastFrom;

    constructor(address _lockbox, address _sysCfg) {
        lockbox = _lockbox;
        sysCfg = _sysCfg;
    }

    function ethLockbox() external view returns (address) {
        return lockbox;
    }

    function systemConfig() external view returns (address) {
        return sysCfg;
    }

    function depositTransaction(address, uint256, uint64, bool, bytes memory _data) external payable {
        address envSender;
        assembly {
            envSender := and(mload(add(_data, 68)), 0xffffffffffffffffffffffffffffffffffffffff)
        }
        deposits++;
        lastFrom = msg.sender;
        lastEnvelopeSender = envSender;
        if (envSender == msg.sender) selfSenderDeposits++;
    }
}

contract ReachL1CDMHalmos is Test {
    SVMReachL1 internal constant svm = SVMReachL1(0xF3993A62377BCd56AE39D773740A5390411E8BC9);
    address internal constant L2CDM = 0x4200000000000000000000000000000000000007;

    L1CrossDomainMessenger internal l1cdm;
    MockSystemConfig internal sysCfg; // caller chain's SystemConfig
    MockCallerPortal internal callerPortal;
    MockCallerMessenger internal caller;
    MockLockbox internal lockbox;
    MockSystemConfig internal sysCfgA;
    PortalRecorder internal portalA;

    function setUp() public {
        sysCfg = new MockSystemConfig();
        callerPortal = new MockCallerPortal(address(sysCfg));
        caller = new MockCallerMessenger(address(callerPortal));
        lockbox = new MockLockbox();
        sysCfgA = new MockSystemConfig();
        portalA = new PortalRecorder(address(lockbox), address(sysCfgA));
        l1cdm = new L1CrossDomainMessenger();
        vm.store(address(l1cdm), bytes32(uint256(252)), bytes32(uint256(uint160(address(portalA)))));
        vm.store(address(l1cdm), bytes32(uint256(254)), bytes32(uint256(uint160(address(sysCfgA)))));
        vm.store(address(l1cdm), bytes32(uint256(207)), bytes32(uint256(uint160(L2CDM))));
        vm.store(address(l1cdm), bytes32(uint256(204)), bytes32(uint256(uint160(Constants.DEFAULT_L2_SENDER))));
        sysCfgA.set(address(l1cdm), false);
        assert(address(l1cdm.portal()) == address(portalA) && address(l1cdm.systemConfig()) == address(sysCfgA));
    }

    struct Step {
        uint256 kind; // 0 sendMessage, 1 relayMessage, 2 relayUndeliveredMessage, 3 initialize, 4 unknown selector
        address from;
        uint256 value;
        address target;
        uint32 minGas;
        uint256 nonce;
        address sender;
        uint256 msgValue; // relayMessage's _value
        bytes32 h;
        uint256 t;
        address a1;
        address a2;
        bytes4 sel;
        bytes32 w1;
    }

    /// @notice The whole ABI (from the artifact's methodIdentifiers).
    function _known(bytes4 _s) internal pure returns (bool) {
        return _s == bytes4(keccak256("ENCODING_OVERHEAD()")) || _s == bytes4(keccak256("FLOOR_CALLDATA_OVERHEAD()"))
            || _s == bytes4(keccak256("MESSAGE_VERSION()")) || _s == bytes4(keccak256("MIN_GAS_CALLDATA_OVERHEAD()"))
            || _s == bytes4(keccak256("MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR()"))
            || _s == bytes4(keccak256("MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR()"))
            || _s == bytes4(keccak256("OTHER_MESSENGER()")) || _s == bytes4(keccak256("PORTAL()"))
            || _s == bytes4(keccak256("RELAY_CALL_OVERHEAD()")) || _s == bytes4(keccak256("RELAY_CONSTANT_OVERHEAD()"))
            || _s == bytes4(keccak256("RELAY_GAS_CHECK_BUFFER()")) || _s == bytes4(keccak256("RELAY_RESERVED_GAS()"))
            || _s == bytes4(keccak256("TX_BASE_GAS()")) || _s == bytes4(keccak256("baseGas(bytes,uint32)"))
            || _s == bytes4(keccak256("failedMessages(bytes32)")) || _s == bytes4(keccak256("initVersion()"))
            || _s == bytes4(keccak256("initialize(address,address)")) || _s == bytes4(keccak256("messageNonce()"))
            || _s == bytes4(keccak256("otherMessenger()")) || _s == bytes4(keccak256("paused()"))
            || _s == bytes4(keccak256("portal()")) || _s == bytes4(keccak256("proxyAdmin()"))
            || _s == bytes4(keccak256("proxyAdminOwner()"))
            || _s == bytes4(keccak256("relayMessage(uint256,address,address,uint256,uint256,bytes)"))
            || _s == bytes4(keccak256("relayUndeliveredMessage(bytes32,uint256)"))
            || _s == bytes4(keccak256("sendMessage(address,bytes,uint32)"))
            || _s == bytes4(keccak256("successfulMessages(bytes32)")) || _s == bytes4(keccak256("superchainConfig()"))
            || _s == bytes4(keccak256("systemConfig()")) || _s == bytes4(keccak256("version()"))
            || _s == bytes4(keccak256("xDomainMessageSender()"));
    }

    function _calldata(Step memory _s, bytes calldata _message) internal view returns (bytes memory) {
        uint256 k = _s.kind % 5;
        if (k == 0) return abi.encodeCall(l1cdm.sendMessage, (_s.target, _message, _s.minGas));
        if (k == 1) {
            return abi.encodeCall(
                l1cdm.relayMessage, (_s.nonce, _s.sender, _s.target, _s.msgValue, uint256(_s.minGas), _message)
            );
        }
        if (k == 2) return abi.encodeCall(l1cdm.relayUndeliveredMessage, (_s.h, _s.t));
        if (k == 3) return abi.encodeCall(l1cdm.initialize, (ISystemConfig(_s.a1), IOptimismPortal2(payable(_s.a2))));
        return abi.encodePacked(_s.sel, _s.w1);
    }

    function _step(Step memory _s, bytes calldata _message) internal {
        uint256 k = _s.kind % 5;
        // The L1CDM acts only through its own code; address(0) is not a real caller.
        vm.assume(_s.from != address(l1cdm) && _s.from != address(0) && _s.value <= 1 << 128);
        if (k == 4) vm.assume(!_known(_s.sel));
        if (k == 1) {
            // Relay targets: a codeless account, A's L1CDM itself or A's portal (the two blocked targets). Relayed
            // calls into arbitrary code are covered by check_L1_relayGate_and_delivery; aliasing every mock here made
            // one
            // step exceed 40 minutes.
            uint256 choice = uint160(_s.target) % 3;
            _s.target = choice == 0 ? address(0xC0DE) : (choice == 1 ? address(l1cdm) : address(portalA));
        }
        uint256 depBefore = portalA.deposits();
        uint256 selfBefore = portalA.selfSenderDeposits();

        vm.deal(_s.from, _s.value);
        vm.prank(_s.from);
        (bool ok,) = address(l1cdm).call{ value: _s.value }(_calldata(_s, _message));

        uint256 newDeps = portalA.deposits() - depBefore;
        assert(newDeps <= 1);
        if (portalA.selfSenderDeposits() != selfBefore) assert(ok && k == 2);
        if (newDeps == 1) {
            assert(ok);
            assert(portalA.lastFrom() == address(l1cdm));
            assert(k == 2 || (k == 0 && portalA.lastEnvelopeSender() == _s.from));
        }
        if (k == 4) assert(!ok);
    }

    /// @notice Two steps, each one of: sendMessage (any caller, msg.value, target, message, gas), relayMessage (any
    ///         caller including A's portal, any fields), relayUndeliveredMessage (any caller, including the mock
    ///         messengers through symbolic aliasing), initialize, or an unknown selector. Every mock answer is symbolic
    ///         (caller-side getters and revert flags, lockbox, A's SystemConfig: INTEROP flag and paused). A's
    ///         portal.l2Sender() is 0, so no first delivery passes the relay gate; relayed calls are covered by
    ///         check_L1_relayGate_and_delivery. L1CDM storage starts as deployed.
    /// @custom:halmos --default-bytes-lengths 0,68
    function check_reach_l1cdm_sequence2(
        Step memory _s1,
        bytes calldata _m1,
        Step memory _s2,
        bytes calldata _m2
    )
        public
    {
        _symbolicMocks();
        _step(_s1, _m1);
        _step(_s2, _m2);
    }

    function _symbolicMocks() internal {
        svm.enableSymbolicStorage(address(sysCfg));
        svm.enableSymbolicStorage(address(callerPortal));
        svm.enableSymbolicStorage(address(caller));
        svm.enableSymbolicStorage(address(lockbox));
        svm.enableSymbolicStorage(address(sysCfgA));
    }

    /// @notice NON-VACUITY (expected FAIL): A's L1CDM is never the envelope sender in one step (the
    ///         relayUndeliveredMessage path reaches it).
    /// @custom:halmos --default-bytes-lengths 0
    function check_FALSE_reach_l1cdmNeverSelfSender(Step memory _s1, bytes calldata _m1) public {
        _symbolicMocks();
        _step(_s1, _m1);
        assert(portalA.selfSenderDeposits() == 0);
    }
}
