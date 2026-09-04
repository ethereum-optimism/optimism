// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { Vm } from "forge-std/Vm.sol";

// Contracts
import { Proxy } from "src/universal/Proxy.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { ClaimRegistry } from "src/private-interop/ClaimRegistry.sol";

// Interfaces
import { IClaimRegistry, RangeClaim } from "interfaces/private-interop/IClaimRegistry.sol";

/// @title ClaimRegistry_TestInit
/// @notice Reusable test initialization for `ClaimRegistry` tests.
abstract contract ClaimRegistry_TestInit is Test {
    /// @notice Registry under test, behind a proxy.
    IClaimRegistry internal registry;

    /// @notice ProxyAdmin that owns the proxy.
    ProxyAdmin internal proxyAdmin;

    /// @notice Representative batcher address used by tests; the outer batch authenticates it.
    address internal operator;

    /// @notice Test setup.
    function setUp() public virtual {
        operator = makeAddr("operator");

        proxyAdmin = new ProxyAdmin(makeAddr("proxyAdminOwner"));
        registry = _freshRegistry();
    }

    /// @notice Deploys a fresh registry behind its own proxy.
    function _freshRegistry() internal returns (IClaimRegistry registry_) {
        Proxy proxy = new Proxy(address(proxyAdmin));
        ClaimRegistry impl = new ClaimRegistry();

        vm.prank(address(proxyAdmin));
        proxy.upgradeTo(address(impl));

        registry_ = IClaimRegistry(address(proxy));
    }

    /// @notice Builds a well-formed v1 claim covering the given range. Every field gets a distinct
    ///         value derived from the range, so a test comparing commitments cannot pass by
    ///         accident on two fields that happen to hold the same value.
    function _claim(uint64 _firstBlock, uint64 _lastBlock) internal pure returns (RangeClaim memory claim_) {
        claim_ = RangeClaim({
            version: 1,
            firstBlock: _firstBlock,
            lastBlock: _lastBlock,
            privateTerminalBlockHash: keccak256(abi.encode("privateTerminal", _lastBlock)),
            privateTerminalParentHash: keccak256(abi.encode("privateTerminalParent", _lastBlock)),
            l1Head: keccak256(abi.encode("l1Head", _lastBlock)),
            rollupConfigHash: keccak256("rollupConfig"),
            depSetHash: keccak256("depSet"),
            privateDataHash: keccak256(abi.encode("privateData", _firstBlock, _lastBlock)),
            proof: hex""
        });
    }
}

/// @title ClaimRegistry_PostClaim_Test
/// @notice Tests the `postClaim` function of the `ClaimRegistry` contract.
contract ClaimRegistry_PostClaim_Test is ClaimRegistry_TestInit {
    /// @notice Tests that the first post is accepted at any starting block and records the range.
    function testFuzz_postClaim_firstPost_succeeds(uint64 _firstBlock, uint64 _length) external {
        _firstBlock = uint64(bound(_firstBlock, 0, type(uint32).max));
        _length = uint64(bound(_length, 0, type(uint16).max));
        uint64 lastBlock = _firstBlock + _length;

        RangeClaim memory claim = _claim(_firstBlock, lastBlock);

        vm.prank(operator);
        registry.postClaim(claim);

        assertEq(registry.rangeCount(), 1);
        assertEq(registry.lastPostedLastBlock(), lastBlock);
        assertEq(registry.lastClaimHash(), keccak256(abi.encode(bytes32(0), abi.encode(claim))));
    }

    /// @notice Tests that both private terminal hashes are inside what the registry commits to.
    ///         Each variation posts to a fresh registry, so a differing `lastClaimHash` can only
    ///         come from the mutated field. This is the guard against a struct or codec change
    ///         silently dropping the parent hash: the supernode relies on it to serve complete
    ///         follow references, and a field that fell out of the commitment would still decode
    ///         fine, just uncommitted.
    function test_postClaim_terminalHashesAreCommitted_succeeds() external {
        IClaimRegistry baseRegistry = _freshRegistry();
        vm.prank(operator);
        baseRegistry.postClaim(_claim(100, 399));
        bytes32 baseHash = baseRegistry.lastClaimHash();

        RangeClaim memory mutatedBlockHash = _claim(100, 399);
        mutatedBlockHash.privateTerminalBlockHash = bytes32(uint256(0xa11ce));
        IClaimRegistry blockHashRegistry = _freshRegistry();
        vm.prank(operator);
        blockHashRegistry.postClaim(mutatedBlockHash);
        assertTrue(blockHashRegistry.lastClaimHash() != baseHash);

        RangeClaim memory mutatedParentHash = _claim(100, 399);
        mutatedParentHash.privateTerminalParentHash = bytes32(uint256(0xb0b));
        IClaimRegistry parentHashRegistry = _freshRegistry();
        vm.prank(operator);
        parentHashRegistry.postClaim(mutatedParentHash);
        assertTrue(parentHashRegistry.lastClaimHash() != baseHash);
        assertTrue(parentHashRegistry.lastClaimHash() != blockHashRegistry.lastClaimHash());
    }

    /// @notice Tests that posting a claim emits no logs at all. The claim is the first transaction
    ///         of a range-opening block, so any log here would sit ahead of every message the block
    ///         renders and shift each one's log index. On this chain a message's log position is
    ///         its identity, so a rendering-only log would break the canonical-position rule.
    function test_postClaim_emitsNoLogs_succeeds() external {
        vm.recordLogs();
        vm.prank(operator);
        registry.postClaim(_claim(100, 399));

        Vm.Log[] memory logs = vm.getRecordedLogs();
        assertEq(logs.length, 0);
    }

    /// @notice Tests that a contiguous second range is accepted and folds into the claim chain.
    function test_postClaim_contiguous_succeeds() external {
        RangeClaim memory first = _claim(100, 399);
        vm.prank(operator);
        registry.postClaim(first);
        bytes32 firstHash = registry.lastClaimHash();

        RangeClaim memory second = _claim(400, 699);
        vm.prank(operator);
        registry.postClaim(second);

        assertEq(registry.lastClaimHash(), keccak256(abi.encode(firstHash, abi.encode(second))));
        assertEq(registry.rangeCount(), 2);
        assertEq(registry.lastPostedLastBlock(), 699);
    }

    /// @notice Tests that a range skipping forward over a gap is accepted. A range whose opening
    ///         block is invalidated and replaced never executes its claim transaction, so the
    ///         registry cannot advance for it; a strict `+ 1` rule would let that voided range
    ///         wedge every honest claim after it. The gap is the mark the voided range leaves.
    function test_postClaim_forwardGap_succeeds() external {
        vm.prank(operator);
        registry.postClaim(_claim(100, 399));
        bytes32 firstHash = registry.lastClaimHash();

        RangeClaim memory afterGap = _claim(700, 999);
        vm.prank(operator);
        registry.postClaim(afterGap);

        assertEq(registry.rangeCount(), 2);
        assertEq(registry.lastPostedLastBlock(), 999);
        assertEq(registry.lastClaimHash(), keccak256(abi.encode(firstHash, abi.encode(afterGap))));
    }

    /// @notice Tests that a range overlapping the last posted range is refused, gap rule or not.
    function test_postClaim_overlap_reverts() external {
        vm.prank(operator);
        registry.postClaim(_claim(100, 399));

        vm.expectRevert(IClaimRegistry.ClaimRegistry_OverlappingRange.selector);
        vm.prank(operator);
        registry.postClaim(_claim(399, 699));
    }

    /// @notice Tests that a range running backwards behind the last posted range is refused.
    function test_postClaim_regression_reverts() external {
        vm.prank(operator);
        registry.postClaim(_claim(100, 399));

        vm.expectRevert(IClaimRegistry.ClaimRegistry_OverlappingRange.selector);
        vm.prank(operator);
        registry.postClaim(_claim(50, 99));
    }

    /// @notice Tests that re-posting the same range is refused.
    function test_postClaim_duplicate_reverts() external {
        RangeClaim memory claim = _claim(100, 399);

        vm.prank(operator);
        registry.postClaim(claim);

        vm.expectRevert(IClaimRegistry.ClaimRegistry_OverlappingRange.selector);
        vm.prank(operator);
        registry.postClaim(claim);
    }

    /// @notice Tests that an inverted range is refused.
    function test_postClaim_invertedRange_reverts() external {
        vm.expectRevert(IClaimRegistry.ClaimRegistry_InvalidRange.selector);
        vm.prank(operator);
        registry.postClaim(_claim(400, 399));
    }

    /// @notice Tests that a claim of an unsupported version is refused.
    function testFuzz_postClaim_unsupportedVersion_reverts(uint8 _version) external {
        vm.assume(_version != 1);

        RangeClaim memory claim = _claim(100, 399);
        claim.version = _version;

        vm.expectRevert(IClaimRegistry.ClaimRegistry_UnsupportedClaimVersion.selector);
        vm.prank(operator);
        registry.postClaim(claim);
    }

    /// @notice Tests that a claim carrying proof bytes is refused, so nobody can publish something
    ///         that looks proven while v1 verifies nothing.
    function testFuzz_postClaim_nonEmptyProof_reverts(bytes calldata _proof) external {
        vm.assume(_proof.length != 0);

        RangeClaim memory claim = _claim(100, 399);
        claim.proof = _proof;

        vm.expectRevert(IClaimRegistry.ClaimRegistry_ProofNotSupported.selector);
        vm.prank(operator);
        registry.postClaim(claim);
    }

    /// @notice Tests that a rejected post leaves the range state untouched.
    function test_postClaim_rejectedPostLeavesStateUnchanged_succeeds() external {
        vm.prank(operator);
        registry.postClaim(_claim(100, 399));
        bytes32 firstHash = registry.lastClaimHash();

        vm.expectRevert(IClaimRegistry.ClaimRegistry_OverlappingRange.selector);
        vm.prank(operator);
        registry.postClaim(_claim(200, 799));

        assertEq(registry.rangeCount(), 1);
        assertEq(registry.lastPostedLastBlock(), 399);
        assertEq(registry.lastClaimHash(), firstHash);
    }
}
