// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Phase 2: whole-contract reachability properties of the REAL L2ToL2CrossDomainMessenger, deployed AS ON CHAIN: the
// real Proxy (src/universal/Proxy.sol, admin = the L2 ProxyAdmin) at 0x4200..0023 delegating to the real
// implementation, initialized by the ProxyAdmin with the production period as L2ContractsManager does. See README.md
// "Reachability (phase 2)".
//
// A STEP is one call to 0x..23 from a symbolic caller (not the ProxyAdmin or its owner: governance, see README; not
// address(0), which only eth_call can be) with symbolic msg.value, at a symbolic non-decreasing block.timestamp,
// choosing symbolically among every state-changing entry point and an UNKNOWN selector:
//   SEND    sendMessage(dest, target, message)
//   RELAY   relayMessage(id, canonical SentMessage payload(dest, target, nonce, sender, message)), target one of a
//           codeless account, 0x..07, 0x..16
//   EXPIRE  expireMessage(h, t)          (0x..07 answers xDomainMessageSender() with a per-step symbolic value)
//   INIT    initialize(t)
//   OTHER   a symbolic 4-byte selector that is none of the contract's selectors, plus 32 symbolic bytes (one word: a
//           second symbolic word would be a symbolic ABI offset for the proxy's upgradeToAndCall(address,bytes))
// View functions are not steps: the compiler forbids state writes in them (and the proxy forwards them unchanged).
//
// Properties, at a symbolic message hash K, checked after EVERY step:
//   (E) expiredMessages[K] never goes true->false; it goes false->true only in an EXPIRE step with msg.sender ==
//       0x..07, xDomainMessageSender() == otherMessenger(), h == K, sentMessageTimestamps[K] != 0 and
//       t > sentMessageTimestamps[K] + expiryPeriod() (all as they were before the step).
//   (T) sentMessageTimestamps[K] changes only in a SEND step that sends K, from 0 to block.timestamp.
//   (S) successfulMessages[K] changes only in a RELAY step that relays K, from false to true.
//   (P) expiryPeriod() never changes: only the ProxyAdmin or its owner may initialize, and only once.
// check_reach_sequence2: two steps from the deployed (fresh) state, so every state is reachable.
// check_reach_step_symbolicStorage: one step from FULLY symbolic storage (every state, reachable or not); there (T) is
// weakened to "changes only in a SEND step that sends K, to block.timestamp" (an unreachable state may already hold a
// value for the next nonce's hash).

import { Test } from "test/setup/Test.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { L2ToL2CrossDomainMessenger } from "src/L2/L2ToL2CrossDomainMessenger.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { SVM, SVM_ADDRESS, MockCrossL2Inbox, MockL2CDMGetters, MockProxyAdmin } from "./HalmosMocks.sol";

contract ReachL2ToL2Halmos is Test {
    SVM internal constant svm = SVM(SVM_ADDRESS);
    address internal constant L2_TO_L2 = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;
    address internal constant L2CDM = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
    address internal constant PASSER = Predeploys.L2_TO_L1_MESSAGE_PASSER;
    address internal constant INBOX = Predeploys.CROSS_L2_INBOX;
    address internal constant PROXY_ADMIN = Predeploys.PROXY_ADMIN;
    /// @notice The L2 ProxyAdmin's owner in this harness (governance, like the ProxyAdmin itself).
    address internal constant PROXY_ADMIN_OWNER = address(0x0A11CE);
    /// @notice Storage slot of `expiryPeriod` and OpenZeppelin v5 Initializable's namespaced slot.
    bytes32 internal constant EXPIRY_PERIOD_SLOT = bytes32(uint256(5));
    bytes32 internal constant INITIALIZABLE_SLOT = 0xf0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a00;

    uint256 internal constant SEND = 0;
    uint256 internal constant RELAY = 1;
    uint256 internal constant EXPIRE = 2;
    uint256 internal constant OTHER = 3;
    uint256 internal constant INIT = 4;

    L2ToL2CrossDomainMessenger internal m = L2ToL2CrossDomainMessenger(L2_TO_L2);
    address internal impl;
    address[3] internal harness;

    struct Step {
        uint256 kind;
        address caller;
        uint256 value;
        uint256 ts;
        // SEND / RELAY
        uint256 dest;
        address target;
        uint256 nonce;
        address sender;
        Identifier id;
        // EXPIRE (t is also INIT's period argument)
        bytes32 h;
        uint256 t;
        address xSender;
        // OTHER
        bytes4 sel;
        bytes32 w1;
    }

    struct Obs {
        bool exp;
        uint256 ts;
        bool succ;
        uint256 nonce;
        uint256 period;
    }

    function setUp() public {
        impl = address(new L2ToL2CrossDomainMessenger());
        bytes memory code = abi.encodePacked(
            DeployUtils.getCode("test/formal/expiry/halmos/out/Proxy.sol/Proxy.json"), abi.encode(PROXY_ADMIN)
        );
        address proxy;
        assembly {
            proxy := create(0, add(code, 32), mload(code))
        }
        vm.etch(L2_TO_L2, proxy.code);
        vm.store(L2_TO_L2, Constants.PROXY_OWNER_ADDRESS, bytes32(uint256(uint160(PROXY_ADMIN))));
        vm.store(L2_TO_L2, Constants.PROXY_IMPLEMENTATION_ADDRESS, bytes32(uint256(uint160(impl))));
        vm.etch(INBOX, address(new MockCrossL2Inbox()).code);
        vm.etch(L2CDM, address(new MockL2CDMGetters()).code);
        vm.etch(PROXY_ADMIN, address(new MockProxyAdmin()).code);
        MockProxyAdmin(PROXY_ADMIN).setOwner(PROXY_ADMIN_OWNER);
        harness[0] = impl;
        harness[1] = proxy;
        harness[2] = address(this);
        // The ProxyAdmin initializes the proxy with the production period, as L2ContractsManager does.
        vm.prank(PROXY_ADMIN);
        m.initialize(Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD);
        assert(m.expiryPeriod() == Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD); // the proxy delegates
    }

    function _obs(bytes32 _k) internal view returns (Obs memory o_) {
        o_.exp = m.expiredMessages(_k);
        o_.ts = m.sentMessageTimestamps(_k);
        o_.succ = m.successfulMessages(_k);
        o_.nonce = m.messageNonce();
        o_.period = m.expiryPeriod();
    }

    /// @notice The implementation's whole ABI (from its artifact's methodIdentifiers). Proxy selectors (upgradeTo, ...)
    /// are NOT here: for a non-admin caller the proxy forwards them, and they must revert like any unknown selector.
    function _isKnownSelector(bytes4 _s) internal pure returns (bool) {
        return _s == bytes4(keccak256("expiryPeriod()")) || _s == bytes4(keccak256("crossDomainMessageContext()"))
            || _s == bytes4(keccak256("crossDomainMessageSender()"))
            || _s == bytes4(keccak256("crossDomainMessageSource()"))
            || _s == bytes4(keccak256("expireMessage(bytes32,uint256)"))
            || _s == bytes4(keccak256("expiredMessages(bytes32)")) || _s == bytes4(keccak256("initialize(uint256)"))
            || _s == bytes4(keccak256("messageNonce()")) || _s == bytes4(keccak256("messageVersion()"))
            || _s == bytes4(keccak256("proxyAdmin()")) || _s == bytes4(keccak256("proxyAdminOwner()"))
            || _s == bytes4(keccak256("relayMessage((address,uint256,uint256,uint256,uint256),bytes)"))
            || _s == bytes4(keccak256("sendMessage(uint256,address,bytes)"))
            || _s == bytes4(keccak256("sentMessageTimestamps(bytes32)"))
            || _s == bytes4(keccak256("sentMessages(uint256)"))
            || _s == bytes4(keccak256("successfulMessages(bytes32)")) || _s == bytes4(keccak256("version()"));
    }

    function _payload(Step memory _s, bytes calldata _message) internal pure returns (bytes memory) {
        return abi.encodePacked(
            abi.encode(L2ToL2CrossDomainMessenger.SentMessage.selector, _s.dest, _s.target, _s.nonce),
            abi.encode(_s.sender, _message)
        );
    }

    /// @notice The step's kind from its low three bits (a bit mask is cheap for the solver, unlike `% 5`); the values
    ///         above INIT are OTHER steps too.
    function _kind(Step memory _s) internal pure returns (uint256 kind_) {
        kind_ = _s.kind & 7;
        if (kind_ > INIT) kind_ = OTHER;
    }

    function _calldata(Step memory _s, bytes calldata _message) internal pure returns (bytes memory) {
        uint256 kind = _kind(_s);
        if (kind == SEND) {
            return abi.encodeCall(L2ToL2CrossDomainMessenger.sendMessage, (_s.dest, _s.target, _message));
        }
        if (kind == RELAY) {
            return abi.encodeCall(L2ToL2CrossDomainMessenger.relayMessage, (_s.id, _payload(_s, _message)));
        }
        if (kind == EXPIRE) return abi.encodeCall(L2ToL2CrossDomainMessenger.expireMessage, (_s.h, _s.t));
        if (kind == INIT) return abi.encodeCall(L2ToL2CrossDomainMessenger.initialize, (_s.t));
        return abi.encodePacked(_s.sel, _s.w1);
    }

    /// @notice Runs one step and checks (E), (T), (S), (P) at `_k`. `_fresh`: the state is reachable (strong (T)).
    function _step(Step memory _s, bytes calldata _message, bytes32 _k, address _other, bool _fresh) internal {
        uint256 kind = _kind(_s);
        vm.assume(_s.caller != PROXY_ADMIN && _s.caller != PROXY_ADMIN_OWNER && _s.caller != address(0));
        vm.assume(_s.ts >= block.timestamp);
        vm.assume(_s.value <= 1 << 128);
        if (kind == OTHER) vm.assume(!_isKnownSelector(_s.sel));
        if (kind == RELAY) {
            // Relay targets: a codeless account, 0x..07 or 0x..16 (the two blocked targets). Relayed calls into
            // arbitrary code are covered by L2ToL2ExpiryHalmos (re-entrant and observing targets); letting the symbolic
            // target alias every account here made two steps exceed 40 minutes.
            uint256 choice = uint160(_s.target) % 3;
            _s.target = choice == 0 ? address(0xC0DE) : (choice == 1 ? L2CDM : PASSER);
        }
        vm.warp(_s.ts);
        MockL2CDMGetters(L2CDM).setGetters(_s.xSender, _other);

        Obs memory pre = _obs(_k);
        uint256 period = pre.period;
        bytes memory data = _calldata(_s, _message);
        vm.deal(_s.caller, _s.value);
        vm.prank(_s.caller);
        (bool ok,) = L2_TO_L2.call{ value: _s.value }(data);
        Obs memory post = _obs(_k);

        // (E)
        if (pre.exp) assert(post.exp);
        if (!pre.exp && post.exp) {
            assert(ok && kind == EXPIRE);
            assert(_s.caller == L2CDM && _s.xSender == _other && _s.h == _k);
            // Bounded before adding, so a wrapped sentAt + EXPIRY_PERIOD is a counterexample, not a 0x11 panic
            // (which halmos would not count as a failure).
            assert(pre.ts != 0 && pre.ts <= type(uint256).max - period && _s.t > pre.ts + period);
        }
        // (T)
        if (post.ts != pre.ts) {
            assert(ok && kind == SEND);
            assert(post.ts == block.timestamp);
            assert(_k == keccak256(abi.encode(_s.dest, block.chainid, pre.nonce, _s.caller, _s.target, _message)));
            if (_fresh) assert(pre.ts == 0);
        }
        // (S)
        if (post.succ != pre.succ) {
            assert(ok && kind == RELAY && !pre.succ);
            assert(_k == keccak256(abi.encode(block.chainid, _s.id.chainId, _s.nonce, _s.sender, _s.target, _message)));
        }
        // (P)
        assert(post.period == pre.period);
        if (kind == OTHER) assert(!ok); // no fallback: unknown selectors always revert
    }

    /// @notice Two symbolic steps from the deployed state (every state reachable; three exceeded 40 minutes). Two
    ///         steps already cover send-then-expire and send-then-relay.
    /// @custom:halmos --default-bytes-lengths 0,33
    function check_reach_sequence2(
        uint256 _chainId,
        bytes32 _k,
        address _other,
        Step memory _s1,
        bytes calldata _m1,
        Step memory _s2,
        bytes calldata _m2
    )
        public
    {
        vm.chainId(_chainId);
        _step(_s1, _m1, _k, _other, true);
        _step(_s2, _m2, _k, _other, true);
    }

    /// @notice Fully symbolic messenger storage, except the proxy's admin and implementation slots. Symbolic storage
    ///         leaves the slots setUp wrote concrete, so the period and the Initializable word get fresh symbolic
    ///         values here.
    function _symbolicStorage() internal {
        svm.enableSymbolicStorage(L2_TO_L2);
        vm.store(L2_TO_L2, Constants.PROXY_OWNER_ADDRESS, bytes32(uint256(uint160(PROXY_ADMIN))));
        vm.store(L2_TO_L2, Constants.PROXY_IMPLEMENTATION_ADDRESS, bytes32(uint256(uint160(impl))));
        vm.store(L2_TO_L2, EXPIRY_PERIOD_SLOT, svm.createBytes32("expiryPeriod"));
        vm.store(L2_TO_L2, INITIALIZABLE_SLOT, svm.createBytes32("initializable"));
    }

    /// @notice One symbolic step from FULLY symbolic messenger storage (proxy admin/implementation slots kept).
    function check_reach_step_symbolicStorage(
        uint256 _chainId,
        bytes32 _k,
        address _other,
        Step memory _s,
        bytes calldata _m
    )
        public
    {
        vm.chainId(_chainId);
        _symbolicStorage();
        _step(_s, _m, _k, _other, false);
    }

    /// @notice NON-VACUITY (expected FAIL): from fully symbolic storage, one step never changes any of the three
    ///         observations at K (witness for check_reach_step_symbolicStorage: a transition is reachable there).
    ///         With a symbolic period and Initializable word, finding a counterexample for every failing path takes
    ///         over an hour, so this witness stops at the first one.
    /// @custom:halmos --early-exit
    function check_FALSE_reach_step_noTransition(
        uint256 _chainId,
        bytes32 _k,
        address _other,
        Step memory _s,
        bytes calldata _m
    )
        public
    {
        vm.chainId(_chainId);
        _symbolicStorage();
        Obs memory pre = _obs(_k);
        _step(_s, _m, _k, _other, false);
        Obs memory post = _obs(_k);
        assert(post.exp == pre.exp && post.ts == pre.ts && post.succ == pre.succ);
    }

    /// @notice NON-VACUITY (expected FAIL) for (P): from fully symbolic storage, governance (the ProxyAdmin or its
    ///         owner) never changes the period with initialize. The counterexample is an initialize on an uninitialized
    ///         state, so the observation (P) reads can change.
    function check_FALSE_reach_periodNeverChanges(bool _byOwner, uint256 _p) public {
        _symbolicStorage();
        uint256 before = m.expiryPeriod();
        vm.prank(_byOwner ? PROXY_ADMIN_OWNER : PROXY_ADMIN);
        (bool ok,) = L2_TO_L2.call(abi.encodeCall(L2ToL2CrossDomainMessenger.initialize, (_p)));
        assert(!ok || m.expiryPeriod() == before);
    }

    /// @notice NON-VACUITY (expected FAIL): expiredMessages[K] never becomes true within three steps. The
    ///         counterexample is send K, then (later) expire K from 0x..07 with the right xDomainMessageSender.
    /// @custom:halmos --default-bytes-lengths 0
    function check_FALSE_reach_expiredNeverSet(
        uint256 _chainId,
        bytes32 _k,
        address _other,
        Step memory _s1,
        bytes calldata _m1,
        Step memory _s2,
        bytes calldata _m2
    )
        public
    {
        vm.chainId(_chainId);
        _step(_s1, _m1, _k, _other, true);
        _step(_s2, _m2, _k, _other, true);
        assert(!m.expiredMessages(_k));
    }
}
