// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { ExecuteNUTBundle } from "scripts/upgrade/ExecuteNUTBundle.s.sol";

// Libraries
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { PastNUTBundles } from "test/setup/PastNUTBundles.sol";

// Interfaces
import { IL2ProxyAdmin } from "interfaces/L2/IL2ProxyAdmin.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title PastNUTBundles_TestInit
/// @notice Shared test init for `PastNUTBundles` unit tests.
abstract contract PastNUTBundles_TestInit is Test {
    /// @notice Path to the committed Karst NUT bundle, relative to `packages/contracts-bedrock`.
    string internal constant KARST_BUNDLE_PATH = "../../op-core/nuts/bundles/karst_nut_bundle.json";

    /// @notice L2ContractsManager address encoded by the committed Karst NUT bundle.
    address internal constant KARST_L2CM = 0x5398A70Eb0929dd7bfc73c59E7137d8C7CDF6669;
}

/// @title PastNUTBundles_OrderTarget
/// @notice Minimal target contract used to observe wrapper dispatcher execution order.
contract PastNUTBundles_OrderTarget {
    uint256[] public records;
    address[] public senders;

    function record(uint256 _value) external {
        records.push(_value);
        senders.push(msg.sender);
    }

    function recordsLength() external view returns (uint256) {
        return records.length;
    }
}

/// @title PastNUTBundles_extractL2CM_Test
/// @notice Exercises the `extractL2CM` validation rules.
contract PastNUTBundles_extractL2CM_Test is PastNUTBundles_TestInit {
    /// @notice External wrapper so `vm.expectRevert` can catch the revert from the internal
    ///         library call.
    function _callExtractL2CM(
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _txns,
        string memory _path
    )
        external
        pure
        returns (address)
    {
        return PastNUTBundles.extractL2CM(_txns, _path);
    }

    /// @notice The Karst bundle's final tx decodes to the expected L2CM via the structural rule.
    function test_extractL2CM_karst_succeeds() public view {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = NetworkUpgradeTxns.readArtifact(KARST_BUNDLE_PATH);
        address l2cm = PastNUTBundles.extractL2CM(txns, KARST_BUNDLE_PATH);
        assertEq(l2cm, KARST_L2CM);
    }

    /// @notice Reverts when the bundle has no transactions.
    function test_extractL2CM_emptyBundle_reverts() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](0);
        vm.expectRevert(abi.encodeWithSelector(PastNUTBundles.PastNUTBundles_EmptyBundle.selector, "test-path"));
        this._callExtractL2CM(txns, "test-path");
    }

    /// @notice Reverts when the final tx targets something other than `Predeploys.PROXY_ADMIN`.
    function testFuzz_extractL2CM_wrongTarget_reverts(address _wrongTarget) public {
        vm.assume(_wrongTarget != Predeploys.PROXY_ADMIN);
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (address(0xCAFE))),
            from: address(0),
            gasLimit: 0,
            intent: "Invalid Final Upgrade Transaction",
            to: _wrongTarget
        });

        vm.expectRevert(
            abi.encodeWithSelector(PastNUTBundles.PastNUTBundles_WrongTarget.selector, "test-path", _wrongTarget)
        );
        this._callExtractL2CM(txns, "test-path");
    }

    /// @notice Reverts when the final tx selector is not `upgradePredeploys(address)`.
    function testFuzz_extractL2CM_wrongSelector_reverts(bytes4 _wrongSelector) public {
        vm.assume(_wrongSelector != IL2ProxyAdmin.upgradePredeploys.selector);
        bytes memory data = abi.encodePacked(_wrongSelector, bytes32(uint256(uint160(address(0xCAFE)))));
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: data,
            from: address(0),
            gasLimit: 0,
            intent: "Invalid Final Upgrade Transaction",
            to: Predeploys.PROXY_ADMIN
        });

        vm.expectRevert(
            abi.encodeWithSelector(PastNUTBundles.PastNUTBundles_WrongSelector.selector, "test-path", _wrongSelector)
        );
        this._callExtractL2CM(txns, "test-path");
    }

    /// @notice Reverts when the final tx calldata is not exactly 36 bytes.
    function test_extractL2CM_wrongDataLength_reverts() public {
        bytes memory shortData = abi.encodePacked(IL2ProxyAdmin.upgradePredeploys.selector, bytes31(0));
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: shortData,
            from: address(0),
            gasLimit: 0,
            intent: "Invalid Final Upgrade Transaction",
            to: Predeploys.PROXY_ADMIN
        });

        vm.expectRevert(
            abi.encodeWithSelector(PastNUTBundles.PastNUTBundles_WrongDataLength.selector, "test-path", uint256(35))
        );
        this._callExtractL2CM(txns, "test-path");
    }

    /// @notice Reverts when the decoded L2CM is `address(0)`.
    function test_extractL2CM_zeroL2CM_reverts() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (address(0))),
            from: address(0),
            gasLimit: 0,
            intent: "Invalid Final Upgrade Transaction",
            to: Predeploys.PROXY_ADMIN
        });

        vm.expectRevert(abi.encodeWithSelector(PastNUTBundles.PastNUTBundles_ZeroL2CM.selector, "test-path"));
        this._callExtractL2CM(txns, "test-path");
    }

    /// @notice Decodes the L2CM for any non-zero address when the final tx is well-formed.
    function testFuzz_extractL2CM_validFinalTxn_succeeds(address _l2cm) public pure {
        vm.assume(_l2cm != address(0));
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (_l2cm)),
            from: address(0),
            gasLimit: 0,
            intent: "L2ProxyAdmin Upgrade Predeploys",
            to: Predeploys.PROXY_ADMIN
        });

        assertEq(PastNUTBundles.extractL2CM(txns, "test-path"), _l2cm);
    }
}

/// @title PastNUTBundles_wrappersForFork_Test
/// @notice Exercises the empty default wrapper lookup.
contract PastNUTBundles_wrappersForFork_Test is PastNUTBundles_TestInit {
    /// @notice The default wrapper hook returns no pre or post wrappers for arbitrary fork names.
    function testFuzz_wrappersForFork_empty_succeeds(string memory _fork) public pure {
        PastNUTBundles.ForkWrappers memory wrappers = PastNUTBundles.wrappersForFork(_fork);

        assertEq(wrappers.pre.length, 0, "pre length");
        assertEq(wrappers.post.length, 0, "post length");
    }
}

/// @title PastNUTBundles_executeWithWrappers_Test
/// @notice Exercises the wrapper dispatcher.
contract PastNUTBundles_executeWithWrappers_Test is PastNUTBundles_TestInit {
    /// @dev High fixture gas for wrapper-order tests; executor subtracts intrinsic gas before forwarding.
    uint64 internal constant FIXTURE_GAS_LIMIT = 1_000_000;

    ExecuteNUTBundle internal script;
    PastNUTBundles_OrderTarget internal target;
    address internal alice;
    address internal bob;
    address internal carol;
    address internal dave;

    function setUp() public {
        script = new ExecuteNUTBundle();
        target = new PastNUTBundles_OrderTarget();
        alice = makeAddr("alice");
        bob = makeAddr("bob");
        carol = makeAddr("carol");
        dave = makeAddr("dave");
    }

    /// @notice Non-empty pre and post wrappers execute around the bundle in order.
    function test_executeWithWrappers_preBundlePostOrder_succeeds() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory pre = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        pre[0] = _txn(alice, 1, "pre");

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory bundle = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        bundle[0] = _txn(bob, 2, "bundle");

        ExecuteNUTBundle.PostWrapperTxn[] memory post = new ExecuteNUTBundle.PostWrapperTxn[](2);
        post[0] = _wrapper(carol, 3, "post 1");
        post[1] = _wrapper(dave, 4, "post 2");

        PastNUTBundles.executeWithWrappers(script, pre, bundle, post);

        assertEq(target.recordsLength(), 4, "records length");
        assertEq(target.records(0), 1, "records[0]");
        assertEq(target.records(1), 2, "records[1]");
        assertEq(target.records(2), 3, "records[2]");
        assertEq(target.records(3), 4, "records[3]");
        assertEq(target.senders(0), alice, "senders[0]");
        assertEq(target.senders(1), bob, "senders[1]");
        assertEq(target.senders(2), carol, "senders[2]");
        assertEq(target.senders(3), dave, "senders[3]");
    }

    function _txn(
        address _from,
        uint256 _value,
        string memory _intent
    )
        internal
        view
        returns (NetworkUpgradeTxns.NetworkUpgradeTxn memory txn_)
    {
        txn_ = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(PastNUTBundles_OrderTarget.record, (_value)),
            from: _from,
            gasLimit: FIXTURE_GAS_LIMIT,
            intent: _intent,
            to: address(target)
        });
    }

    function _wrapper(
        address _from,
        uint256 _value,
        string memory _intent
    )
        internal
        view
        returns (ExecuteNUTBundle.PostWrapperTxn memory wrapper_)
    {
        wrapper_ = ExecuteNUTBundle.PostWrapperTxn({
            from: _from,
            to: address(target),
            data: abi.encodeCall(PastNUTBundles_OrderTarget.record, (_value)),
            gasLimit: FIXTURE_GAS_LIMIT,
            mint: 0,
            value: 0,
            intent: _intent
        });
    }
}

/// @title PastNUTBundles_applyPastBundles_Test
/// @notice Exercises the apply loop with explicit current transactions.
contract PastNUTBundles_applyPastBundles_Test is PastNUTBundles_TestInit {
    /// @notice When the current L2CM matches a non-Karst prior bundle's L2CM, the apply loop
    ///         must not invoke the executor for that bundle.
    function test_applyPastBundles_skipsWhenL2CMMatches_succeeds() public {
        ExecuteNUTBundle script = new ExecuteNUTBundle();

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory karstTxns = NetworkUpgradeTxns.readArtifact(KARST_BUNDLE_PATH);
        PastNUTBundles.NUTBundle[] memory entries = new PastNUTBundles.NUTBundle[](1);
        entries[0] = PastNUTBundles.NUTBundle({ fork: "future-fork", path: KARST_BUNDLE_PATH });

        // No call to executeAll should occur when the current L2CM matches the prior bundle's.
        vm.expectCall(address(script), abi.encodePacked(ExecuteNUTBundle.executeAll.selector), 0);
        PastNUTBundles.applyPastBundles(karstTxns, script, entries);
    }

    /// @notice When the current bundle already contains a direct deterministic deployment from a
    ///         prior bundle, applying that prior bundle would make the current bundle collide
    ///         when it later reaches the same CREATE2 deployment.
    /// @dev TODO(#19369): Remove with the Karst direct CREATE2 bootstrap skip.
    function testFuzz_applyPastBundles_skipsWhenCurrentContainsDirectCreate2_succeeds(address _l2cm) public {
        vm.assume(_l2cm != address(0) && _l2cm != KARST_L2CM);
        ExecuteNUTBundle script = new ExecuteNUTBundle();

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory karstTxns = NetworkUpgradeTxns.readArtifact(KARST_BUNDLE_PATH);
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory currentTxns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](2);
        currentTxns[0] = karstTxns[0];
        currentTxns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (_l2cm)),
            from: address(0),
            gasLimit: 0,
            intent: "L2ProxyAdmin Upgrade Predeploys",
            to: Predeploys.PROXY_ADMIN
        });
        PastNUTBundles.NUTBundle[] memory entries = new PastNUTBundles.NUTBundle[](1);
        entries[0] = PastNUTBundles.NUTBundle({ fork: "karst", path: KARST_BUNDLE_PATH });

        // No call to executeAll should occur even though the current L2CM differs, because the
        // current bundle owns the same non-idempotent direct CREATE2 deployment.
        vm.expectCall(address(script), abi.encodePacked(ExecuteNUTBundle.executeAll.selector), 0);
        PastNUTBundles.applyPastBundles(currentTxns, script, entries);
    }

    /// @notice The direct CREATE2 overlap skip is a Karst bootstrap exception, not a generic
    ///         "matching transaction means skip the whole prior bundle" rule.
    /// @dev TODO(#19369): Remove with the Karst direct CREATE2 bootstrap skip.
    function testFuzz_applyPastBundles_nonKarstDirectCreate2Overlap_succeeds(address _l2cm) public {
        vm.assume(_l2cm != address(0) && _l2cm != KARST_L2CM);
        ExecuteNUTBundle script = new ExecuteNUTBundle();

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory karstTxns = NetworkUpgradeTxns.readArtifact(KARST_BUNDLE_PATH);
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory currentTxns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](2);
        currentTxns[0] = karstTxns[0];
        currentTxns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (_l2cm)),
            from: address(0),
            gasLimit: 0,
            intent: "L2ProxyAdmin Upgrade Predeploys",
            to: Predeploys.PROXY_ADMIN
        });
        PastNUTBundles.NUTBundle[] memory entries = new PastNUTBundles.NUTBundle[](1);
        entries[0] = PastNUTBundles.NUTBundle({ fork: "future-fork", path: KARST_BUNDLE_PATH });

        vm.mockCall(address(script), abi.encodePacked(ExecuteNUTBundle.executeAll.selector), "");
        vm.expectCall(address(script), abi.encodeCall(ExecuteNUTBundle.executeAll, (karstTxns)));

        PastNUTBundles.applyPastBundles(currentTxns, script, entries);
    }

    /// @notice When the current L2CM differs from a prior bundle's L2CM, the apply loop must
    ///         invoke the executor with the parsed prior transactions.
    function testFuzz_applyPastBundles_executesWhenL2CMDiffers_succeeds(address _l2cm) public {
        vm.assume(_l2cm != address(0) && _l2cm != KARST_L2CM);
        ExecuteNUTBundle script = new ExecuteNUTBundle();

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory karstTxns = NetworkUpgradeTxns.readArtifact(KARST_BUNDLE_PATH);
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory currentTxns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        currentTxns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (_l2cm)),
            from: address(0),
            gasLimit: 0,
            intent: "L2ProxyAdmin Upgrade Predeploys",
            to: Predeploys.PROXY_ADMIN
        });
        PastNUTBundles.NUTBundle[] memory entries = new PastNUTBundles.NUTBundle[](1);
        entries[0] = PastNUTBundles.NUTBundle({ fork: "karst", path: KARST_BUNDLE_PATH });

        // Mock the actual execution so the test does not require a forked L2 environment.
        vm.mockCall(address(script), abi.encodePacked(ExecuteNUTBundle.executeAll.selector), "");
        vm.expectCall(address(script), abi.encodeCall(ExecuteNUTBundle.executeAll, (karstTxns)));

        PastNUTBundles.applyPastBundles(currentTxns, script, entries);
    }

    /// @notice Karst is still applied on live forks that are missing the ConditionalDeployer
    ///         bootstrap, even when Karst's L2CM matches the current bundle's L2CM.
    /// @dev TODO(#19369): Remove with the Karst same-L2CM pre-bootstrap exception.
    function test_applyPastBundles_karstSameL2CMExecutesWhenConditionalDeployerMissing_succeeds() public {
        ExecuteNUTBundle script = new ExecuteNUTBundle();

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory karstTxns = NetworkUpgradeTxns.readArtifact(KARST_BUNDLE_PATH);
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory currentTxns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        currentTxns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (KARST_L2CM)),
            from: address(0),
            gasLimit: 0,
            intent: "L2ProxyAdmin Upgrade Predeploys",
            to: Predeploys.PROXY_ADMIN
        });
        PastNUTBundles.NUTBundle[] memory entries = new PastNUTBundles.NUTBundle[](1);
        entries[0] = PastNUTBundles.NUTBundle({ fork: "karst", path: KARST_BUNDLE_PATH });

        vm.mockCallRevert(
            Predeploys.CONDITIONAL_DEPLOYER, abi.encodePacked(ISemver.version.selector), bytes("not initialized")
        );
        vm.mockCall(address(script), abi.encodePacked(ExecuteNUTBundle.executeAll.selector), "");
        vm.expectCall(address(script), abi.encodeCall(ExecuteNUTBundle.executeAll, (karstTxns)));

        PastNUTBundles.applyPastBundles(currentTxns, script, entries);
    }

    /// @notice Karst is skipped when the current L2CM matches Karst's L2CM and the live fork
    ///         already has a usable ConditionalDeployer implementation. The current txns must
    ///         not include the Karst direct CREATE2 bootstrap or the bootstrap-overlap rule would
    ///         mask the same-L2CM rule under test.
    /// @dev TODO(#19369): Fold into the generic same-L2CM skip test once the Karst exception is removed.
    function test_applyPastBundles_karstSameL2CMSkipsWhenConditionalDeployerInitialized_succeeds() public {
        ExecuteNUTBundle script = new ExecuteNUTBundle();

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory currentTxns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        currentTxns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (KARST_L2CM)),
            from: address(0),
            gasLimit: 0,
            intent: "L2ProxyAdmin Upgrade Predeploys",
            to: Predeploys.PROXY_ADMIN
        });
        PastNUTBundles.NUTBundle[] memory entries = new PastNUTBundles.NUTBundle[](1);
        entries[0] = PastNUTBundles.NUTBundle({ fork: "karst", path: KARST_BUNDLE_PATH });

        vm.mockCall(Predeploys.CONDITIONAL_DEPLOYER, abi.encodePacked(ISemver.version.selector), abi.encode("1.0.0"));

        vm.expectCall(address(script), abi.encodePacked(ExecuteNUTBundle.executeAll.selector), 0);
        PastNUTBundles.applyPastBundles(currentTxns, script, entries);
    }

    /// @notice An empty entries array is a no-op: the loop must not invoke the executor.
    function test_applyPastBundles_emptyEntries_succeeds() public {
        ExecuteNUTBundle script = new ExecuteNUTBundle();

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory currentTxns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](0);
        PastNUTBundles.NUTBundle[] memory entries = new PastNUTBundles.NUTBundle[](0);

        vm.expectCall(address(script), abi.encodePacked(ExecuteNUTBundle.executeAll.selector), 0);
        PastNUTBundles.applyPastBundles(currentTxns, script, entries);
    }

    /// @notice Multiple entries are evaluated independently: Karst is bootstrap-skipped because the
    ///         current bundle owns the same direct CREATE2 deployment, while a non-Karst entry
    ///         pointing at the same artifact applies because the bootstrap rule is Karst-only and
    ///         the current L2CM differs from the bundle's L2CM.
    /// @dev TODO(#19369): Remove with the Karst direct CREATE2 bootstrap skip.
    function testFuzz_applyPastBundles_multipleEntriesSkipOneApplyOther_succeeds(address _l2cm) public {
        vm.assume(_l2cm != address(0) && _l2cm != KARST_L2CM);
        ExecuteNUTBundle script = new ExecuteNUTBundle();

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory karstTxns = NetworkUpgradeTxns.readArtifact(KARST_BUNDLE_PATH);
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory currentTxns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](2);
        currentTxns[0] = karstTxns[0];
        currentTxns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (_l2cm)),
            from: address(0),
            gasLimit: 0,
            intent: "L2ProxyAdmin Upgrade Predeploys",
            to: Predeploys.PROXY_ADMIN
        });

        PastNUTBundles.NUTBundle[] memory entries = new PastNUTBundles.NUTBundle[](2);
        entries[0] = PastNUTBundles.NUTBundle({ fork: "karst", path: KARST_BUNDLE_PATH });
        entries[1] = PastNUTBundles.NUTBundle({ fork: "future-fork", path: KARST_BUNDLE_PATH });

        vm.mockCall(address(script), abi.encodePacked(ExecuteNUTBundle.executeAll.selector), "");
        vm.expectCall(address(script), abi.encodeCall(ExecuteNUTBundle.executeAll, (karstTxns)), 1);

        PastNUTBundles.applyPastBundles(currentTxns, script, entries);
    }
}
