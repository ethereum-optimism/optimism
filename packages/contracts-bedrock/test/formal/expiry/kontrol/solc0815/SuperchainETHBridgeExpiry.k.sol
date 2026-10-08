// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Kontrol (KEVM) proofs for SuperchainETHBridge.refundETH (property 5).
//
// Model: the REAL SuperchainETHBridge is etched at 0x4200..0024 with fully symbolic storage
// (refunded[] arbitrary), the REAL ETHLiquidity is etched at 0x4200..0025 with balance 2^128 wei,
// and a mock with a fully symbolic expiredMessages mapping (ANY set of expired hashes) is etched at
// 0x4200..0023. block.chainid is symbolic. Not runnable with plain forge (Kontrol-only cheat
// codes).

// Contracts
import { SuperchainETHBridge } from "src/L2/SuperchainETHBridge.sol";
import { ETHLiquidity } from "src/L2/ETHLiquidity.sol";
import { ExpiryKontrolBaseL1, MockExpiredMessages } from "test/formal/expiry/kontrol/solc0815/ExpiryMocks0815.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

contract SuperchainETHBridgeExpiryKontrol is ExpiryKontrolBaseL1 {
    /// @notice Predeploy addresses (solc 0.8.15 cannot initialise constants from library constants;
    ///         setUp asserts they equal Predeploys.SUPERCHAIN_ETH_BRIDGE, ETH_LIQUIDITY and
    ///         L2_TO_L2_CROSS_DOMAIN_MESSENGER).
    address internal constant BRIDGE = 0x4200000000000000000000000000000000000024;
    address internal constant LIQUIDITY = 0x4200000000000000000000000000000000000025;
    address internal constant L2TOL2 = 0x4200000000000000000000000000000000000023;
    uint256 internal constant LIQUIDITY_BALANCE = 2 ** 128;

    SuperchainETHBridge internal constant bridge = SuperchainETHBridge(BRIDGE);
    MockExpiredMessages internal constant l2tol2 = MockExpiredMessages(L2TOL2);

    address internal bridgeImpl;
    address internal liquidityImpl;
    address internal l2tol2Impl;

    function setUp() public {
        assert(BRIDGE == Predeploys.SUPERCHAIN_ETH_BRIDGE);
        assert(LIQUIDITY == Predeploys.ETH_LIQUIDITY);
        assert(L2TOL2 == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        bridgeImpl = address(new SuperchainETHBridge());
        liquidityImpl = address(new ETHLiquidity());
        l2tol2Impl = address(new MockExpiredMessages());
        _etch(BRIDGE, bridgeImpl);
        _etch(LIQUIDITY, liquidityImpl);
        _etch(L2TOL2, l2tol2Impl);
        vm.deal(LIQUIDITY, LIQUIDITY_BALANCE);
    }

    function _etch(address _at, address _impl) internal {
        vm.etch(_at, _impl.code);
        vm.etch(_impl, hex"");
    }

    /// @notice The message hash refundETH must bind, written out independently of the Hashing
    ///         library: keccak256(abi.encode(destination, source = chainid, nonce, sender = bridge,
    ///         target = bridge, relayETH(from, to, amount))).
    function _expectedHash(
        uint256 _destination,
        uint256 _chainId,
        uint256 _nonce,
        address _from,
        address _to,
        uint256 _amount
    )
        internal
        pure
        returns (bytes32)
    {
        return keccak256(
            abi.encode(
                _destination,
                _chainId,
                _nonce,
                BRIDGE,
                BRIDGE,
                abi.encodeCall(SuperchainETHBridge.relayETH, (_from, _to, _amount))
            )
        );
    }

    function _refund(
        uint256 _destination,
        uint256 _nonce,
        address _from,
        address _to,
        uint256 _amount
    )
        internal
        returns (bool ok_, bytes memory ret_)
    {
        vm.prank(kevm.freshAddress());
        (ok_, ret_) =
            BRIDGE.call(abi.encodeCall(SuperchainETHBridge.refundETH, (_destination, _nonce, _from, _to, _amount)));
    }

    /// @notice Address CREATE gives the first contract deployed by `_deployer` (nonce 0, the nonce
    ///         of every etched predeploy in this harness).
    function _firstCreate(address _deployer) internal pure returns (address) {
        return
            address(uint160(uint256(keccak256(abi.encodePacked(bytes1(0xd6), bytes1(0x94), _deployer, bytes1(0x80))))));
    }

    /// @notice Recipient exclusion: `from` is not an account of this harness (the test contract,
    ///         the cheat-code address, the three etched predeploys, the wiped implementation
    ///         addresses, and the SafeSends that ETHLiquidity.mint and refundETH deploy). KEVM's
    ///         account map cannot alias a symbolic SELFDESTRUCT beneficiary with an existing
    ///         account, so without this the proof gets stuck. Refunds pay `from` by SELFDESTRUCT,
    ///         which runs no code, so the identity of `from` does not affect success or `refunded`.
    function _assumeRecipient(address _from) internal view {
        vm.assume(_from != address(this) && _from != address(vm));
        vm.assume(_from != BRIDGE && _from != LIQUIDITY && _from != L2TOL2);
        vm.assume(_from != bridgeImpl && _from != liquidityImpl && _from != l2tol2Impl);
        vm.assume(_from != _firstCreate(LIQUIDITY) && _from != _firstCreate(BRIDGE));
    }

    function _setup() internal returns (uint256 chainId_) {
        chainId_ = kevm.freshUInt(32);
        vm.chainId(chainId_);
        kevm.symbolicStorage(BRIDGE);
        kevm.symbolicStorage(L2TOL2);
    }

    /// @notice PREIMAGE BINDING: for ALL destination, nonce, from, to, amount <= liquidity,
    ///         chainid, refunded[] and expiredMessages[]: refundETH succeeds IFF expiredMessages[H]
    ///         && !refunded[H] for H = keccak256(abi.encode(destination, chainid, nonce, bridge,
    ///         bridge, relayETH(from, to, amount))); after it, refunded[H] == (old || success).
    function prove_refundETH_preimageBinding(
        uint256 _destination,
        uint256 _nonce,
        address _from,
        address _to,
        uint256 _amount
    )
        external
    {
        uint256 chainId = _setup();
        vm.assume(_amount <= LIQUIDITY_BALANCE);
        _assumeRecipient(_from);

        bytes32 h = _expectedHash(_destination, chainId, _nonce, _from, _to, _amount);
        bool expired = l2tol2.expiredMessages(h);
        bool refundedBefore = bridge.refunded(h);

        (bool ok,) = _refund(_destination, _nonce, _from, _to, _amount);

        assert(ok == (expired && !refundedBefore));
        assert(bridge.refunded(h) == (refundedBefore || ok));
    }

    /// @notice WITNESS (expected to FAIL): refundETH can succeed in this harness, so the two proofs
    ///         here are not vacuous (in particular the vm.assume(ok1) in prove_refundETH_singleUse
    ///         is satisfiable).
    function prove_refundETH_canSucceed_WITNESS(
        uint256 _destination,
        uint256 _nonce,
        address _from,
        address _to,
        uint256 _amount
    )
        external
    {
        _setup();
        vm.assume(_amount <= LIQUIDITY_BALANCE / 2);
        _assumeRecipient(_from);
        (bool ok,) = _refund(_destination, _nonce, _from, _to, _amount);
        assert(!ok);
    }

    /// @notice SINGLE USE: after ANY successful refundETH, the same call reverts with
    ///         SuperchainETHBridge_AlreadyRefunded.
    function prove_refundETH_singleUse(
        uint256 _destination,
        uint256 _nonce,
        address _from,
        address _to,
        uint256 _amount
    )
        external
    {
        _setup();
        vm.assume(_amount <= LIQUIDITY_BALANCE / 2);
        _assumeRecipient(_from);

        (bool ok1,) = _refund(_destination, _nonce, _from, _to, _amount);
        vm.assume(ok1);
        (bool ok2, bytes memory ret2) = _refund(_destination, _nonce, _from, _to, _amount);

        assert(!ok2);
        assert(ret2.length == 4);
        assert(bytes4(ret2) == SuperchainETHBridge.SuperchainETHBridge_AlreadyRefunded.selector);
    }

    /// @notice SINGLE USE, second half on its own (cheaper than the two-call proof above): from ANY
    ///         state in which refunded[H] is set (all other storage symbolic), refundETH for the
    ///         arguments hashing to H reverts with SuperchainETHBridge_AlreadyRefunded or
    ///         SuperchainETHBridge_MessageNotExpired, and never pays. With
    ///         prove_refundETH_preimageBinding (a success sets refunded[H]) this gives single use.
    function prove_refundETH_alreadyRefundedReverts(
        uint256 _destination,
        uint256 _nonce,
        address _from,
        address _to,
        uint256 _amount
    )
        external
    {
        uint256 chainId = _setup();
        bytes32 h = _expectedHash(_destination, chainId, _nonce, _from, _to, _amount);
        vm.assume(bridge.refunded(h));

        (bool ok, bytes memory ret) = _refund(_destination, _nonce, _from, _to, _amount);

        assert(!ok);
        assert(ret.length == 4);
        assert(
            bytes4(ret) == SuperchainETHBridge.SuperchainETHBridge_AlreadyRefunded.selector
                || bytes4(ret) == SuperchainETHBridge.SuperchainETHBridge_MessageNotExpired.selector
        );
    }

    /// @notice WITNESS (expected to FAIL): under the assumptions of the proof above, refundETH runs
    ///         to its AlreadyRefunded revert (the asserted outcome is reachable).
    function prove_refundETH_alreadyRefundedReachable_WITNESS(
        uint256 _destination,
        uint256 _nonce,
        address _from,
        address _to,
        uint256 _amount
    )
        external
    {
        uint256 chainId = _setup();
        bytes32 h = _expectedHash(_destination, chainId, _nonce, _from, _to, _amount);
        vm.assume(bridge.refunded(h));
        (, bytes memory ret) = _refund(_destination, _nonce, _from, _to, _amount);
        assert(bytes4(ret) != SuperchainETHBridge.SuperchainETHBridge_AlreadyRefunded.selector);
    }

    // ---------------------------------------------------------------------------------------------
    // Phase 2: whole-contract reachability (any selector)
    // ---------------------------------------------------------------------------------------------

    /// @notice For ANY 4-byte selector, any caller (possibly 0x..23), any five argument words and
    ///         ANY hash H0: if refunded[H0] goes from false to true, then the selector is
    ///         refundETH's, H0 is the refundETH preimage hash of the arguments, and
    ///         expiredMessages[H0] holds. So no other function (sendETH, relayETH, getters, unknown
    ///         selectors) sets `refunded`. The arguments are passed as five words; for functions
    ///         with fewer or narrower parameters the leading words are decoded (and must be in
    ///         range, or the call reverts). The same recipient exclusion as above applies to the
    ///         address that each path pays.
    function prove_bridge_anySelector_refundedOnlyByRefundETH(
        bytes4 _selector,
        uint256 _w0,
        uint256 _w1,
        address _w2,
        address _w3,
        uint256 _w4
    )
        external
    {
        uint256 chainId = _setup();
        bytes32 h0 = bytes32(kevm.freshUInt(32));
        // Recipients: refundETH pays _w2; relayETH pays its second word.
        _assumeRecipient(_w2);
        if (_w1 < 2 ** 160) _assumeRecipient(address(uint160(_w1)));
        bool refundedBefore = bridge.refunded(h0);

        vm.prank(kevm.freshAddress());
        (bool ok,) = BRIDGE.call(abi.encodePacked(_selector, abi.encode(_w0, _w1, _w2, _w3, _w4)));
        ok; // success or failure, the property below must hold

        if (!refundedBefore && bridge.refunded(h0)) {
            assert(_selector == SuperchainETHBridge.refundETH.selector);
            assert(h0 == _expectedHash(_w0, chainId, _w1, _w2, _w3, _w4));
            assert(l2tol2.expiredMessages(h0));
        }
    }

    /// @notice WITNESS (expected to FAIL): in the any-selector setting, `refunded[H0]` can flip.
    function prove_bridge_anySelectorRefunds_WITNESS(
        bytes4 _selector,
        uint256 _w0,
        uint256 _w1,
        address _w2,
        address _w3,
        uint256 _w4
    )
        external
    {
        _setup();
        bytes32 h0 = bytes32(kevm.freshUInt(32));
        // Recipients: refundETH pays _w2; relayETH pays its second word.
        _assumeRecipient(_w2);
        if (_w1 < 2 ** 160) _assumeRecipient(address(uint160(_w1)));
        bool refundedBefore = bridge.refunded(h0);

        vm.prank(kevm.freshAddress());
        (bool ok,) = BRIDGE.call(abi.encodePacked(_selector, abi.encode(_w0, _w1, _w2, _w3, _w4)));
        ok; // success or failure, the property below must hold

        assert(!(!refundedBefore && bridge.refunded(h0)));
    }
}
