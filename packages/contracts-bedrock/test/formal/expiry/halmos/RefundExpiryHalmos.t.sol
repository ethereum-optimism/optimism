// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Halmos symbolic checks on the REAL SuperchainETHBridge.refundETH (etched at 0x4200..0024) with the REAL
// ETHLiquidity (etched at 0x4200..0025) and the REAL SafeSend (created by both). See README.md for the exact
// statements, every assumption and bound, and the SELFDESTRUCT patch these checks need.
//
// Group (6): refundETH(destination, nonce, from, to, amount), with
//     H = keccak256(abi.encode(destination, block.chainid, nonce, bridge, bridge,
//                              abi.encodeCall(relayETH, (from, to, amount))))
//   - succeeds iff expiredMessages[H] && !refunded[H] && amount <= ETHLiquidity's balance;
//   - on success: refunded[H] becomes true; `from` gains amount + p2, the bridge gains p1, ETHLiquidity loses
//     amount, where p1 / p2 are whatever was pre-sent to the two SafeSend addresses (ETHLiquidity's, the bridge's)
//     before the call (SafeSend forwards its whole balance); on revert no balance or flag changes;
//   - single use: the identical call then reverts; refunded[H2] unchanged for every H2 != H;
//   - composed with the real L2ToL2CrossDomainMessenger: the hash sendETH's message gets is H for the same
//     arguments, sendMessage records block.timestamp for it, and once it is expired the refund pays `from`.

import { Test } from "test/setup/Test.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { SuperchainETHBridge } from "src/L2/SuperchainETHBridge.sol";
import { ETHLiquidity } from "src/L2/ETHLiquidity.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

interface SVM {
    function enableSymbolicStorage(address) external;
}

contract MockExpired {
    mapping(bytes32 => bool) public expiredMessages; // symbolic (symbolic storage)
}

contract Probe { }

contract RefundExpiryHalmos is Test {
    SVM internal constant svm = SVM(0xF3993A62377BCd56AE39D773740A5390411E8BC9);
    address internal constant L2_TO_L2 = 0x4200000000000000000000000000000000000023;
    address internal constant BRIDGE = 0x4200000000000000000000000000000000000024;
    address internal constant LIQUIDITY = 0x4200000000000000000000000000000000000025;

    /// @notice Bound on every symbolic balance and amount. Halmos 0.3.3 prunes any path that READS a balance above
    ///         MAX_ETH (2^128 stock, 2^200 with halmos-selfdestruct.patch); with inputs <= 2^198, every sum of at most
    ///         four of them stays <= 2^200, so no path is pruned by that cap. (Total ETH supply is ~2^87 wei.)
    uint256 internal constant BAL_BOUND = 1 << 198;

    /// @notice Halmos allocates CREATE addresses sequentially from 0xaaaa0000 (+1, +2, ...) and CREATE2 addresses from
    ///         0xbbbb0000. ASSUMPTION: `from` is not in those ranges, i.e. not one of the SafeSend helpers created
    /// during the check (on a real chain those are CREATE(bridge|liquidity, nonce) addresses, which only the bridge /
    ///         ETHLiquidity can deploy to, so a sender there is not a realistic `from`).
    uint160 internal constant FRESH_LO = 0xaaaa0000;
    uint160 internal constant FRESH2_LO = 0xbbbb0000;

    SuperchainETHBridge internal bridge = SuperchainETHBridge(BRIDGE);
    MockExpired internal msgr = MockExpired(L2_TO_L2);

    struct Env {
        uint256 chainId;
        uint256 liq; // ETHLiquidity balance
        uint256 b0; // bridge balance
        uint256 p1; // pre-sent to ETHLiquidity's SafeSend (forwards to the bridge)
        uint256 p2; // pre-sent to the bridge's SafeSend (forwards to `from`)
    }

    struct Args {
        uint256 destination;
        uint256 nonce;
        address from;
        address to;
        uint256 amount;
    }

    function setUp() public {
        assert(L2_TO_L2 == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        assert(BRIDGE == Predeploys.SUPERCHAIN_ETH_BRIDGE);
        assert(LIQUIDITY == Predeploys.ETH_LIQUIDITY);
        vm.etch(BRIDGE, address(new SuperchainETHBridge()).code);
        vm.etch(LIQUIDITY, address(new ETHLiquidity()).code);
        vm.etch(L2_TO_L2, address(new MockExpired()).code);
    }

    // ---------------------------------------------------------------- helpers

    function _refundHash(Args memory _a) internal view returns (bytes32) {
        return keccak256(
            abi.encode(
                _a.destination,
                block.chainid,
                _a.nonce,
                BRIDGE,
                BRIDGE,
                abi.encodeCall(SuperchainETHBridge.relayETH, (_a.from, _a.to, _a.amount))
            )
        );
    }

    function _assumeRealisticFrom(address _from) internal view {
        // address(0) is never a real sender of sendETH (msg.sender != 0 on chain).
        vm.assume(_from != BRIDGE && _from != LIQUIDITY && _from != address(0));
        vm.assume(uint160(_from) < FRESH_LO || uint160(_from) > FRESH_LO + 0xffff);
        vm.assume(uint160(_from) < FRESH2_LO || uint160(_from) > FRESH2_LO + 0xffff);
        vm.assume(_from.balance <= BAL_BOUND);
    }

    /// @notice Symbolic expired set and refunded map; symbolic balances; pre-sent ETH at the two SafeSend addresses the
    ///         refund will create (the next two CREATE addresses after a probe deployment).
    function _world(Env memory _e, Args memory _a) internal {
        vm.chainId(_e.chainId);
        svm.enableSymbolicStorage(BRIDGE);
        svm.enableSymbolicStorage(L2_TO_L2);
        vm.assume(_e.liq <= BAL_BOUND && _e.b0 <= BAL_BOUND && _e.p1 <= BAL_BOUND && _e.p2 <= BAL_BOUND);
        vm.assume(_a.amount <= BAL_BOUND);
        _assumeRealisticFrom(_a.from);
        vm.deal(LIQUIDITY, _e.liq);
        vm.deal(BRIDGE, _e.b0);
        uint160 next = uint160(address(new Probe())) + 1;
        vm.deal(address(next), _e.p1); // ETHLiquidity.mint's SafeSend
        vm.deal(address(next + 1), _e.p2); // refundETH's SafeSend
    }

    function _refund(address _caller, Args memory _a) internal returns (bool ok_) {
        vm.prank(_caller);
        (ok_,) = BRIDGE.call(abi.encodeCall(bridge.refundETH, (_a.destination, _a.nonce, _a.from, _a.to, _a.amount)));
    }

    // ================================================================ (6) refundETH

    /// @notice iff + effects (with pre-sent ETH) + single use (see header).
    function check_refund_iff_effects_singleUse(address _caller, Env memory _e, Args memory _a) public {
        _world(_e, _a);
        bytes32 h = _refundHash(_a);
        bool expired = msgr.expiredMessages(h);
        bool refundedBefore = bridge.refunded(h);
        uint256 fromBefore = _a.from.balance;

        bool ok = _refund(_caller, _a);

        assert(ok == (expired && !refundedBefore && _a.amount <= _e.liq));
        if (ok) {
            assert(bridge.refunded(h));
            assert(_a.from.balance == fromBefore + _a.amount + _e.p2);
            assert(LIQUIDITY.balance == _e.liq - _a.amount);
            assert(BRIDGE.balance == _e.b0 + _e.p1);
            assert(!_refund(_caller, _a)); // single use
        } else {
            assert(bridge.refunded(h) == refundedBefore);
            assert(_a.from.balance == fromBefore);
            assert(LIQUIDITY.balance == _e.liq);
            assert(BRIDGE.balance == _e.b0);
        }
    }

    /// @notice A refund touches only its own hash: refunded[H2] unchanged for every H2 != H.
    function check_refund_frame(Env memory _e, Args memory _a, bytes32 _h2) public {
        _world(_e, _a);
        vm.assume(_h2 != _refundHash(_a));
        bool before2 = bridge.refunded(_h2);
        _refund(address(this), _a);
        assert(bridge.refunded(_h2) == before2);
    }

    /// @notice NON-VACUITY (expected FAIL): an expired, unrefunded, funded refund still reverts. The counterexample
    ///         is a successful refund (the success side of the iff is reachable).
    function check_FALSE_refund_failsWhenExpired(Env memory _e, Args memory _a) public {
        _world(_e, _a);
        bytes32 h = _refundHash(_a);
        vm.assume(msgr.expiredMessages(h) && !bridge.refunded(h) && _a.amount <= _e.liq);
        assert(!_refund(address(this), _a));
    }

    /// @notice NON-VACUITY (expected FAIL): the refund pays the destination-side recipient `to` (it must pay `from`).
    function check_FALSE_refund_paysTo(Env memory _e, Args memory _a) public {
        _world(_e, _a);
        // `to` is a realistic recipient distinct from `from` (not a SafeSend helper, bridge or liquidity) and the
        // amount is positive, so the only way this can fail is that the refund pays `from`, not `to`.
        _assumeRealisticFrom(_a.to);
        vm.assume(_a.to != _a.from && _a.amount > 0);
        uint256 toBefore = _a.to.balance;
        bool ok = _refund(address(this), _a);
        if (ok) assert(_a.to.balance == toBefore + _a.amount);
    }

    /// @notice NON-VACUITY (expected FAIL): `from` gains exactly `amount` even when ETH was pre-sent to the SafeSend
    ///         address. The counterexample has p2 > 0 (documents why the effect statement includes p2).
    function check_FALSE_refund_ignoresPresentETH(Env memory _e, Args memory _a) public {
        _world(_e, _a);
        uint256 fromBefore = _a.from.balance;
        bool ok = _refund(address(this), _a);
        if (ok) assert(_a.from.balance == fromBefore + _a.amount);
    }

    // ================================================================ send -> expire -> refund (composed)

    /// @notice With the REAL L2ToL2CrossDomainMessenger at 0x..23 (fresh storage except a SYMBOLIC prior nonce
    ///         msgNonce < 2^240 - 1): sendETH from `from` succeeds, its message hash equals refundETH's recomputed H
    /// for (destination, nonce = messageNonce() before, from, to, amount), sentMessageTimestamps[H] == block.timestamp,
    /// refundETH reverts before expiry, and once
    ///         expiredMessages[H] is set (written directly: expireMessage itself is checked in L2ToL2ExpiryHalmos)
    ///         the refund succeeds and pays `from` exactly `amount`.
    function check_sendETH_then_refund(uint256 _chainId, uint256 _ts, uint256 _liq0, Args memory _a) public {
        bytes memory code = DeployUtils.getCode(
            "test/formal/expiry/halmos/out/L2ToL2CrossDomainMessenger.sol/L2ToL2CrossDomainMessenger.json"
        );
        address real;
        assembly {
            real := create(0, add(code, 32), mload(code))
        }
        vm.etch(L2_TO_L2, real.code);
        IL2ToL2CrossDomainMessenger l2tol2 = IL2ToL2CrossDomainMessenger(L2_TO_L2);

        vm.chainId(_chainId);
        vm.warp(_ts);
        vm.assume(_a.destination != _chainId && _a.to != address(0));
        vm.assume(_a.amount <= BAL_BOUND && _liq0 <= BAL_BOUND);
        _assumeRealisticFrom(_a.from);
        vm.deal(LIQUIDITY, _liq0);
        vm.deal(_a.from, _a.from.balance + _a.amount);
        vm.deal(address(this), _a.amount);

        vm.assume(_a.nonce < type(uint240).max); // symbolic prior nonce; the send increments it (checked arithmetic)
        vm.store(L2_TO_L2, bytes32(uint256(1)), bytes32(_a.nonce)); // msgNonce (slot 1)
        assert(l2tol2.messageNonce() == _a.nonce); // slot check (message version 0)
        vm.prank(_a.from);
        (bool sent, bytes memory ret) =
            BRIDGE.call{ value: _a.amount }(abi.encodeCall(bridge.sendETH, (_a.to, _a.destination)));
        assert(sent);
        bytes32 h = abi.decode(ret, (bytes32));
        assert(h == _refundHash(_a));
        assert(l2tol2.sentMessageTimestamps(h) == _ts);

        assert(!_refund(address(this), _a)); // not expired yet
        vm.store(L2_TO_L2, keccak256(abi.encode(h, uint256(4))), bytes32(uint256(1))); // expiredMessages[h] = true
        assert(l2tol2.expiredMessages(h)); // slot check

        uint256 fromBefore = _a.from.balance;
        assert(_refund(address(this), _a));
        assert(_a.from.balance == fromBefore + _a.amount);
    }
}
