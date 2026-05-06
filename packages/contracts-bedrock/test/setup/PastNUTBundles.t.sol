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

/// @title PastNUTBundles_TestInit
/// @notice Reusable harness for `PastNUTBundles` unit tests.
abstract contract PastNUTBundles_TestInit is Test {
    /// @notice Path to the committed Karst NUT bundle, relative to `packages/contracts-bedrock`.
    string internal constant KARST_BUNDLE_PATH = "../../op-core/nuts/bundles/karst_nut_bundle.json";

    /// @notice L2ContractsManager address encoded by the committed Karst NUT bundle.
    address internal constant KARST_L2CM = 0x5398A70Eb0929dd7bfc73c59E7137d8C7CDF6669;
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
        vm.expectRevert(abi.encodeWithSelector(PastNUTBundles.EmptyBundle.selector, "test-path"));
        this._callExtractL2CM(txns, "test-path");
    }

    /// @notice Reverts when the final tx targets something other than `Predeploys.PROXY_ADMIN`.
    function test_extractL2CM_wrongTarget_reverts() public {
        address wrongTarget = address(0xBEEF);
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (address(0xCAFE))),
            from: address(0),
            gasLimit: 0,
            intent: "test",
            to: wrongTarget
        });

        vm.expectRevert(abi.encodeWithSelector(PastNUTBundles.WrongTarget.selector, "test-path", wrongTarget));
        this._callExtractL2CM(txns, "test-path");
    }

    /// @notice Reverts when the final tx selector is not `upgradePredeploys(address)`.
    function test_extractL2CM_wrongSelector_reverts() public {
        bytes4 wrongSelector = 0xdeadbeef;
        bytes memory data = abi.encodePacked(wrongSelector, bytes32(uint256(uint160(address(0xCAFE)))));
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: data, from: address(0), gasLimit: 0, intent: "test", to: Predeploys.PROXY_ADMIN
        });

        vm.expectRevert(abi.encodeWithSelector(PastNUTBundles.WrongSelector.selector, "test-path", wrongSelector));
        this._callExtractL2CM(txns, "test-path");
    }

    /// @notice Reverts when the final tx calldata is not exactly 36 bytes.
    function test_extractL2CM_wrongDataLength_reverts() public {
        bytes memory shortData = abi.encodePacked(IL2ProxyAdmin.upgradePredeploys.selector, bytes31(0));
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: shortData, from: address(0), gasLimit: 0, intent: "test", to: Predeploys.PROXY_ADMIN
        });

        vm.expectRevert(abi.encodeWithSelector(PastNUTBundles.WrongDataLength.selector, "test-path", uint256(35)));
        this._callExtractL2CM(txns, "test-path");
    }

    /// @notice Reverts when the decoded L2CM is `address(0)`.
    function test_extractL2CM_zeroL2CM_reverts() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (address(0))),
            from: address(0),
            gasLimit: 0,
            intent: "test",
            to: Predeploys.PROXY_ADMIN
        });

        vm.expectRevert(abi.encodeWithSelector(PastNUTBundles.ZeroL2CM.selector, "test-path"));
        this._callExtractL2CM(txns, "test-path");
    }
}

/// @title PastNUTBundles_stagePastBundlesAgainst_Test
/// @notice Exercises the staging loop using the test-friendly variant that accepts an explicit current transaction
///         array rather than reading `Constants.CURRENT_BUNDLE_PATH`.
contract PastNUTBundles_stagePastBundlesAgainst_Test is PastNUTBundles_TestInit {
    /// @notice When the current L2CM matches a prior bundle's L2CM, the staging loop must not
    ///         invoke the executor for that bundle.
    function test_stagePastBundlesAgainst_skipsWhenL2CMMatches_succeeds() public {
        ExecuteNUTBundle script = new ExecuteNUTBundle();

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory karstTxns = NetworkUpgradeTxns.readArtifact(KARST_BUNDLE_PATH);
        PastNUTBundles.NUTBundle[] memory entries = new PastNUTBundles.NUTBundle[](1);
        entries[0] = PastNUTBundles.NUTBundle({ fork: "karst", path: KARST_BUNDLE_PATH });

        // No call to executeAll should occur when the current L2CM matches Karst's.
        vm.expectCall(address(script), abi.encodeWithSelector(ExecuteNUTBundle.executeAll.selector), 0);
        PastNUTBundles.stagePastBundlesAgainst(karstTxns, script, entries);
    }

    /// @notice When the current bundle already contains a direct deterministic deployment from a
    ///         prior bundle, staging that prior bundle would make the current bundle collide when
    ///         it later reaches the same CREATE2 deployment.
    function test_stagePastBundlesAgainst_skipsWhenCurrentContainsDirectCreate2_succeeds() public {
        ExecuteNUTBundle script = new ExecuteNUTBundle();

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory karstTxns = NetworkUpgradeTxns.readArtifact(KARST_BUNDLE_PATH);
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory currentTxns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](2);
        currentTxns[0] = karstTxns[0];
        currentTxns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (address(1))),
            from: address(0),
            gasLimit: 0,
            intent: "test",
            to: Predeploys.PROXY_ADMIN
        });
        PastNUTBundles.NUTBundle[] memory entries = new PastNUTBundles.NUTBundle[](1);
        entries[0] = PastNUTBundles.NUTBundle({ fork: "karst", path: KARST_BUNDLE_PATH });

        // No call to executeAll should occur even though the current L2CM differs, because the
        // current bundle owns the same non-idempotent direct CREATE2 deployment.
        vm.expectCall(address(script), abi.encodeWithSelector(ExecuteNUTBundle.executeAll.selector), 0);
        PastNUTBundles.stagePastBundlesAgainst(currentTxns, script, entries);
    }

    /// @notice The direct CREATE2 overlap skip is a Karst bootstrap exception, not a generic
    ///         "matching transaction means skip the whole prior bundle" rule.
    function test_stagePastBundlesAgainst_nonKarstDirectCreate2Overlap_executes() public {
        ExecuteNUTBundle script = new ExecuteNUTBundle();

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory karstTxns = NetworkUpgradeTxns.readArtifact(KARST_BUNDLE_PATH);
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory currentTxns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](2);
        currentTxns[0] = karstTxns[0];
        currentTxns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (address(1))),
            from: address(0),
            gasLimit: 0,
            intent: "test",
            to: Predeploys.PROXY_ADMIN
        });
        PastNUTBundles.NUTBundle[] memory entries = new PastNUTBundles.NUTBundle[](1);
        entries[0] = PastNUTBundles.NUTBundle({ fork: "future-fork", path: KARST_BUNDLE_PATH });

        vm.mockCall(address(script), abi.encodeWithSelector(ExecuteNUTBundle.executeAll.selector), "");
        vm.expectCall(address(script), abi.encodeCall(ExecuteNUTBundle.executeAll, (karstTxns)));

        PastNUTBundles.stagePastBundlesAgainst(currentTxns, script, entries);
    }

    /// @notice When the current L2CM differs from a prior bundle's L2CM, the staging loop must
    ///         invoke the executor with the parsed prior transactions.
    function test_stagePastBundlesAgainst_executesWhenL2CMDiffers_succeeds() public {
        ExecuteNUTBundle script = new ExecuteNUTBundle();

        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory karstTxns = NetworkUpgradeTxns.readArtifact(KARST_BUNDLE_PATH);
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory currentTxns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        currentTxns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (address(1))),
            from: address(0),
            gasLimit: 0,
            intent: "test",
            to: Predeploys.PROXY_ADMIN
        });
        PastNUTBundles.NUTBundle[] memory entries = new PastNUTBundles.NUTBundle[](1);
        entries[0] = PastNUTBundles.NUTBundle({ fork: "karst", path: KARST_BUNDLE_PATH });

        // Mock the actual execution so the test does not require a forked L2 environment.
        vm.mockCall(address(script), abi.encodeWithSelector(ExecuteNUTBundle.executeAll.selector), "");
        vm.expectCall(address(script), abi.encodeCall(ExecuteNUTBundle.executeAll, (karstTxns)));

        // address(1) cannot collide with any CREATE2-derived L2CM, so Karst will be staged.
        PastNUTBundles.stagePastBundlesAgainst(currentTxns, script, entries);
    }
}
