// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Phase 2: whole-contract reachability properties of the REAL UndeliveredMessageExporter and the REAL
// SuperchainETHBridge + ETHLiquidity, each deployed AS ON CHAIN behind the real Proxy (admin = the L2 ProxyAdmin) at
// its predeploy address (exporter: Predeploys.UNDELIVERED_MESSAGE_EXPORTER, never hardcoded). See README.md
// "Reachability (phase 2)". Steps come from symbolic callers that are not the ProxyAdmin (governance) nor address(0)
// (eth_call only), nor any harness or predeploy contract address unless stated.

import { Test } from "test/setup/Test.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { UndeliveredMessageExporter } from "src/L2/UndeliveredMessageExporter.sol";
import { SuperchainETHBridge } from "src/L2/SuperchainETHBridge.sol";
import { ETHLiquidity } from "src/L2/ETHLiquidity.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { Constants } from "src/libraries/Constants.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { CallRecorder, MockSuccessful } from "./ExporterExpiryHalmos.t.sol";

interface SVMReach {
    function enableSymbolicStorage(address) external;
}

/// @notice Installs the real Proxy at `_at` (admin = L2 ProxyAdmin) delegating to `_impl`.
abstract contract ProxyHarness is Test {
    function _installProxy(address _at, address _impl) internal {
        vm.etch(_at, address(new Proxy(Predeploys.PROXY_ADMIN)).code);
        vm.store(_at, Constants.PROXY_OWNER_ADDRESS, bytes32(uint256(uint160(Predeploys.PROXY_ADMIN))));
        vm.store(_at, Constants.PROXY_IMPLEMENTATION_ADDRESS, bytes32(uint256(uint160(_impl))));
    }

    /// @notice The arguments after the selector, as a bytes view into `_data` (overwrites its length word and
    /// selector).
    function _tail(bytes memory _data) internal pure returns (bytes memory t_) {
        assembly {
            let len := mload(_data)
            t_ := add(_data, 4)
            mstore(t_, sub(len, 4))
        }
    }
}

/// @notice (2) The exporter, across sequences of calls (each: export with symbolic arguments and message, version(),
///         an unknown selector incl. the proxy's own, or empty calldata; symbolic caller, msg.value, timestamp): every
///         call it makes to 0x..07 is exactly sendMessage(sm, relayUndeliveredMessage(H, block.timestamp), g) for the
/// step's decoded export arguments, with H computed with block.chainid, and only when !successfulMessages[H] held
/// before the step; it
///         never calls 0x..16; the proxy slots and slots 0..3 never change (the implementation has no state variables).
contract ReachExporterHalmos is ProxyHarness {
    SVMReach internal constant svm = SVMReach(0xF3993A62377BCd56AE39D773740A5390411E8BC9);
    address internal immutable EXPORTER = Predeploys.UNDELIVERED_MESSAGE_EXPORTER;
    address internal constant L2_TO_L2 = 0x4200000000000000000000000000000000000023;
    address internal constant L2CDM = 0x4200000000000000000000000000000000000007;
    address internal constant PASSER = 0x4200000000000000000000000000000000000016;

    function setUp() public {
        _installProxy(EXPORTER, address(new UndeliveredMessageExporter()));
        vm.etch(L2_TO_L2, address(new MockSuccessful()).code);
        vm.etch(L2CDM, address(new CallRecorder()).code);
        vm.etch(PASSER, address(new CallRecorder()).code);
    }

    struct Step {
        uint256 kind; // 0: exportUndeliveredMessage(args, message); 1: version(); 2: unknown selector; 3: empty
        // calldata
        address caller;
        uint256 value;
        uint256 ts;
        address sm;
        uint256 src;
        uint256 nonce;
        address snd;
        address tgt;
        uint32 g;
        bytes4 sel;
        bytes32 w1;
    }

    /// @notice Storage slots observed for the "writes nothing" frame: the proxy's two EIP-1967 slots and slots 0..3
    /// (the implementation declares no state variables: its storage layout is empty).
    function _slots() internal pure returns (bytes32[6] memory s_) {
        s_ = [
            Constants.PROXY_IMPLEMENTATION_ADDRESS,
            Constants.PROXY_OWNER_ADDRESS,
            bytes32(0),
            bytes32(uint256(1)),
            bytes32(uint256(2)),
            bytes32(uint256(3))
        ];
    }

    function _calldata(Step memory _s, bytes calldata _message) internal pure returns (bytes memory) {
        uint256 k = _s.kind % 4;
        if (k == 0) {
            return abi.encodeCall(
                UndeliveredMessageExporter.exportUndeliveredMessage,
                (_s.sm, _s.src, _s.nonce, _s.snd, _s.tgt, _message, _s.g)
            );
        }
        if (k == 1) return abi.encodeCall(ISemver.version, ());
        if (k == 2) return abi.encodePacked(_s.sel, _s.w1);
        return "";
    }

    function _expectedPayload(
        Step memory _s,
        bytes calldata _message,
        uint256 _chainId
    )
        internal
        view
        returns (bytes32 h_, bytes32 payloadHash_)
    {
        h_ = keccak256(abi.encode(_chainId, _s.src, _s.nonce, _s.snd, _s.tgt, _message));
        payloadHash_ = keccak256(
            abi.encodeCall(
                ICrossDomainMessenger.sendMessage,
                (_s.sm, abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (h_, block.timestamp)), _s.g)
            )
        );
    }

    function _step(Step memory _s, bytes calldata _message, uint256 _chainId) internal {
        uint256 k = _s.kind % 4;
        vm.assume(_s.caller != Predeploys.PROXY_ADMIN && _s.caller != address(0) && _s.value <= 1 << 128);
        vm.assume(_s.ts >= block.timestamp);
        vm.warp(_s.ts);
        if (k == 2) {
            // Proxy selectors are NOT excluded: for a non-admin caller the proxy forwards them, and they must revert.
            vm.assume(_s.sel != UndeliveredMessageExporter.exportUndeliveredMessage.selector);
            vm.assume(_s.sel != bytes4(keccak256("version()")));
        }
        uint256 callsBefore = uint256(vm.load(L2CDM, bytes32(0)));
        uint256 totalBefore = uint256(vm.load(L2CDM, bytes32(uint256(1))));
        bytes32[6] memory slots = _slots();
        bytes32[6] memory before;
        for (uint256 i = 0; i < 6; i++) {
            before[i] = vm.load(EXPORTER, slots[i]);
        }
        (bytes32 h, bytes32 expected) = _expectedPayload(_s, _message, _chainId);
        bool relayedBefore = MockSuccessful(L2_TO_L2).successfulMessages(h);

        vm.deal(_s.caller, _s.value);
        vm.prank(_s.caller);
        (bool ok,) = EXPORTER.call{ value: _s.value }(_calldata(_s, _message));

        uint256 newCalls = uint256(vm.load(L2CDM, bytes32(0))) - callsBefore;
        assert(newCalls <= 1);
        assert(uint256(vm.load(L2CDM, bytes32(uint256(1)))) - totalBefore == newCalls); // every call to 0x..07 is ours
        assert(uint256(vm.load(PASSER, bytes32(uint256(1)))) == 0);
        for (uint256 i = 0; i < 6; i++) {
            assert(vm.load(EXPORTER, slots[i]) == before[i]);
        }
        if (newCalls == 1) {
            assert(ok && k == 0 && !relayedBefore);
            assert(vm.load(L2CDM, bytes32(uint256(2))) == expected);
        }
        if (k >= 2) assert(!ok);
        if (k == 0) assert(ok == (!relayedBefore && _s.value == 0)); // export is not payable
    }

    /// @notice Two steps (three exceeded 40 minutes); the messenger's successfulMessages is an arbitrary symbolic
    ///         mapping, so the first step already starts from every messenger state.
    /// @custom:halmos --default-bytes-lengths 0,33
    function check_reach_exporter_sequence2(
        uint256 _chainId,
        Step memory _s1,
        bytes calldata _m1,
        Step memory _s2,
        bytes calldata _m2
    )
        public
    {
        vm.chainId(_chainId);
        svm.enableSymbolicStorage(L2_TO_L2);
        _step(_s1, _m1, _chainId);
        _step(_s2, _m2, _chainId);
    }

    /// @notice NON-VACUITY (expected FAIL): in one step the exporter never calls the L2CrossDomainMessenger.
    /// @custom:halmos --default-bytes-lengths 0
    function check_FALSE_reach_exporterNeverCalls(uint256 _chainId, Step memory _s1, bytes calldata _m1) public {
        vm.chainId(_chainId);
        svm.enableSymbolicStorage(L2_TO_L2);
        _step(_s1, _m1, _chainId);
        assert(uint256(vm.load(L2CDM, bytes32(0))) == 0);
    }
}

/// @notice Mock L2ToL2CrossDomainMessenger for the bridge: expiredMessages and the relay context are symbolic (the
///         test sets the context per step); sendMessage accepts anything (its own checks are proved elsewhere).
contract MockL2ToL2ForBridge {
    mapping(bytes32 => bool) public expiredMessages; // symbolic (symbolic storage)
    address internal ctxSender;
    uint256 internal ctxSource;

    function setContext(address _sender, uint256 _source) external {
        ctxSender = _sender;
        ctxSource = _source;
    }

    function crossDomainMessageContext() external view returns (address, uint256) {
        return (ctxSender, ctxSource);
    }

    function sendMessage(uint256, address, bytes calldata) external pure returns (bytes32) {
        return bytes32(0);
    }
}

/// @notice (4) Bridge + ETHLiquidity, across sequences of calls to EITHER contract (every state-changing function, plus
///         unknown selectors): refunded[K] never goes true->false and goes false->true only in refundETH whose
///         arguments hash to K and with expiredMessages[K]; ETHLiquidity's balance decreases (a mint) only in relayETH
///         called by 0x..23 with context sender == the bridge, or in refundETH, and by exactly the amount.
contract ReachBridgeHalmos is ProxyHarness {
    SVMReach internal constant svm = SVMReach(0xF3993A62377BCd56AE39D773740A5390411E8BC9);
    address internal constant L2_TO_L2 = 0x4200000000000000000000000000000000000023;
    address internal constant BRIDGE = 0x4200000000000000000000000000000000000024;
    address internal constant LIQUIDITY = 0x4200000000000000000000000000000000000025;
    uint256 internal constant BOUND = 1 << 198;

    uint256 internal constant SEND_ETH = 0;
    uint256 internal constant RELAY_ETH = 1;
    uint256 internal constant REFUND_ETH = 2;
    uint256 internal constant BURN = 3;
    uint256 internal constant FUND = 4;
    uint256 internal constant MINT = 5;
    uint256 internal constant UNKNOWN_BRIDGE = 6;
    uint256 internal constant UNKNOWN_LIQUIDITY = 7;

    address internal bridgeImpl;
    address internal liquidityImpl;

    function setUp() public {
        assert(BRIDGE == Predeploys.SUPERCHAIN_ETH_BRIDGE && LIQUIDITY == Predeploys.ETH_LIQUIDITY);
        bridgeImpl = address(new SuperchainETHBridge());
        liquidityImpl = address(new ETHLiquidity());
        _installProxy(BRIDGE, bridgeImpl);
        _installProxy(LIQUIDITY, liquidityImpl);
        vm.etch(L2_TO_L2, address(new MockL2ToL2ForBridge()).code);
    }

    struct Step {
        uint256 kind;
        address caller;
        uint256 value;
        uint256 dest;
        uint256 nonce;
        address from;
        address to;
        uint256 amount;
        address ctxSender;
        uint256 ctxSource;
        bytes4 sel;
        bytes32 w1;
    }

    function _calldata(Step memory _s) internal pure returns (address to_, bytes memory data_) {
        uint256 k = _s.kind % 8;
        if (k == SEND_ETH) return (BRIDGE, abi.encodeCall(SuperchainETHBridge.sendETH, (_s.to, _s.dest)));
        if (k == RELAY_ETH) return (BRIDGE, abi.encodeCall(SuperchainETHBridge.relayETH, (_s.from, _s.to, _s.amount)));
        if (k == REFUND_ETH) {
            return
                (BRIDGE, abi.encodeCall(SuperchainETHBridge.refundETH, (_s.dest, _s.nonce, _s.from, _s.to, _s.amount)));
        }
        if (k == BURN) return (LIQUIDITY, abi.encodeCall(ETHLiquidity.burn, ()));
        if (k == FUND) return (LIQUIDITY, abi.encodeCall(ETHLiquidity.fund, ()));
        if (k == MINT) return (LIQUIDITY, abi.encodeCall(ETHLiquidity.mint, (_s.amount)));
        return (k == UNKNOWN_BRIDGE ? BRIDGE : LIQUIDITY, abi.encodePacked(_s.sel, _s.w1));
    }

    /// @notice The union of both ABIs (from the artifacts' methodIdentifiers).
    function _known(bytes4 _s) internal pure returns (bool) {
        return _s == bytes4(keccak256("refundETH(uint256,uint256,address,address,uint256)"))
            || _s == bytes4(keccak256("refunded(bytes32)"))
            || _s == bytes4(keccak256("relayETH(address,address,uint256)"))
            || _s == bytes4(keccak256("sendETH(address,uint256)")) || _s == bytes4(keccak256("version()"))
            || _s == bytes4(keccak256("burn()")) || _s == bytes4(keccak256("fund()"))
            || _s == bytes4(keccak256("mint(uint256)"));
    }

    function _refundHash(Step memory _s) internal view returns (bytes32) {
        return keccak256(
            abi.encode(
                _s.dest,
                block.chainid,
                _s.nonce,
                BRIDGE,
                BRIDGE,
                abi.encodeCall(SuperchainETHBridge.relayETH, (_s.from, _s.to, _s.amount))
            )
        );
    }

    function _step(Step memory _s, bytes32 _k) internal {
        uint256 k = _s.kind % 8;
        // Callers: anyone but the ProxyAdmin, address(0), the two contracts themselves (they act only through their
        // code) and their implementations. 0x..23 IS allowed: that is how relayETH is reached.
        vm.assume(_s.caller != Predeploys.PROXY_ADMIN && _s.caller != address(0));
        vm.assume(_s.caller != BRIDGE && _s.caller != LIQUIDITY);
        vm.assume(_s.caller != bridgeImpl && _s.caller != liquidityImpl);
        vm.assume(_s.value <= BOUND && _s.amount <= BOUND && _s.caller.balance <= BOUND);
        // `from` / `to` are not halmos-fresh addresses (the SafeSend helpers created during the step; see README).
        vm.assume(uint160(_s.from) < 0xaaaa0000 || uint160(_s.from) > 0xaaaaffff);
        vm.assume(uint160(_s.to) < 0xaaaa0000 || uint160(_s.to) > 0xaaaaffff);
        if (k >= UNKNOWN_BRIDGE) vm.assume(!_known(_s.sel));
        MockL2ToL2ForBridge(L2_TO_L2).setContext(_s.ctxSender, _s.ctxSource);

        bool refundedBefore = SuperchainETHBridge(BRIDGE).refunded(_k);
        bool expiredBefore = MockL2ToL2ForBridge(L2_TO_L2).expiredMessages(_k);
        uint256 liqBefore = LIQUIDITY.balance;
        (address to, bytes memory data) = _calldata(_s);

        vm.deal(_s.caller, _s.caller.balance + _s.value);
        vm.prank(_s.caller);
        (bool ok,) = to.call{ value: _s.value }(data);

        bool refundedAfter = SuperchainETHBridge(BRIDGE).refunded(_k);
        if (refundedBefore) assert(refundedAfter);
        if (!refundedBefore && refundedAfter) {
            assert(ok && k == REFUND_ETH && _k == _refundHash(_s) && expiredBefore);
        }
        if (LIQUIDITY.balance < liqBefore) {
            assert(ok);
            assert((k == RELAY_ETH && _s.caller == L2_TO_L2 && _s.ctxSender == BRIDGE) || k == REFUND_ETH);
            assert(liqBefore - LIQUIDITY.balance == _s.amount);
        }
        if (k >= UNKNOWN_BRIDGE) assert(!ok);
    }

    /// @notice One step over both contracts from FULLY symbolic bridge storage (refunded) and messenger storage
    ///         (expiredMessages, context), with a symbolic liquidity balance: so the transition properties hold from
    ///         every state, reachable or not. (Two- and three-step sequences exceeded 40 minutes; since every
    ///         property is a one-step transition property, symbolic initial state subsumes the sequence.)
    function check_reach_bridge_step(uint256 _chainId, bytes32 _k, uint256 _liq, Step memory _s1) public {
        _bridgeWorld(_chainId, _liq);
        _step(_s1, _k);
    }

    function _bridgeWorld(uint256 _chainId, uint256 _liq) internal {
        vm.chainId(_chainId);
        svm.enableSymbolicStorage(L2_TO_L2);
        svm.enableSymbolicStorage(BRIDGE);
        vm.store(BRIDGE, Constants.PROXY_OWNER_ADDRESS, bytes32(uint256(uint160(Predeploys.PROXY_ADMIN))));
        vm.store(BRIDGE, Constants.PROXY_IMPLEMENTATION_ADDRESS, bytes32(uint256(uint160(bridgeImpl))));
        vm.assume(_liq <= BOUND);
        vm.deal(LIQUIDITY, _liq);
    }

    /// @notice NON-VACUITY (expected FAIL): ETHLiquidity's balance never decreases in one step (same world).
    function check_FALSE_reach_liquidityNeverMints(uint256 _chainId, bytes32 _k, uint256 _liq, Step memory _s1) public {
        _bridgeWorld(_chainId, _liq);
        _step(_s1, _k);
        assert(LIQUIDITY.balance >= _liq);
    }
}
