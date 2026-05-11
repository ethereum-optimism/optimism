// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Forge
import { console2 as console } from "forge-std/console2.sol";

// Scripts
import { ExecuteNUTBundle } from "scripts/upgrade/ExecuteNUTBundle.s.sol";
import { Process } from "scripts/libraries/Process.sol";

// Libraries
import { LibString } from "@solady/utils/LibString.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Bytes } from "src/libraries/Bytes.sol";
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";

// Interfaces
import { IL2ProxyAdmin } from "interfaces/L2/IL2ProxyAdmin.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title PastNUTBundles
/// @notice Test-only library for discovering, validating, skipping, and executing prior NUT
///         bundles before live L2 fork upgrade tests run the current bundle. The L2 fork
///         setUp uses this so chains that have not yet applied prior committed NUTs (e.g.
///         Karst on a not-yet-Karst chain) are seeded with the predeploy state the current
///         bundle expects.
library PastNUTBundles {
    /// @notice One ordered NUT bundle entry returned by the `nut-bundles` Go FFI command.
    /// @param fork The fork name from `op-core/nuts/fork_lock.toml`.
    /// @param path A path resolvable from `packages/contracts-bedrock` via `vm.readFile`.
    struct NUTBundle {
        string fork;
        string path;
    }

    /// @notice Wrapper transactions to execute around a NUT bundle.
    /// @param pre Standard NUT transactions to execute before the bundle.
    /// @param post Wrapper transactions to execute after the bundle.
    struct ForkWrappers {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] pre;
        ExecuteNUTBundle.PostWrapperTxn[] post;
    }

    /// @notice Thrown when a bundle has no transactions and an L2CM cannot be extracted.
    error PastNUTBundles_EmptyBundle(string path);

    /// @notice Thrown when the final transaction does not target `Predeploys.PROXY_ADMIN`.
    error PastNUTBundles_WrongTarget(string path, address to);

    /// @notice Thrown when the final transaction calldata is not `upgradePredeploys(address)`.
    error PastNUTBundles_WrongSelector(string path, bytes4 selector);

    /// @notice Thrown when the final transaction calldata is not the expected 36 bytes
    ///         (4-byte selector + 32-byte ABI-encoded address).
    error PastNUTBundles_WrongDataLength(string path, uint256 length);

    /// @notice Thrown when the decoded L2CM address is the zero address.
    error PastNUTBundles_ZeroL2CM(string path);

    /// @notice Fetches committed NUT bundles in chronological fork order via the Go FFI.
    /// @return bundles_ Ordered list of (fork, path) entries with `path` relative to
    ///                  `packages/contracts-bedrock`.
    function fetchPastBundles() internal returns (NUTBundle[] memory bundles_) {
        string[] memory command = new string[](2);
        command[0] = "scripts/go-ffi/go-ffi";
        command[1] = "nut-bundles";

        bytes memory result = Process.run(command, true);
        bundles_ = abi.decode(result, (NUTBundle[]));
    }

    /// @notice Returns activation wrappers for a fork. Empty for every fork in this PR.
    ///         PRs that introduce wrapped NUT activations should populate the matching
    ///         fork branch and document the production trigger predicate.
    function wrappersForFork(string memory) internal pure returns (ForkWrappers memory w_) {
        return w_;
    }

    /// @notice Executes a NUT bundle with optional pre-bundle transactions and post-bundle wrappers.
    /// @param _executeScript Live `ExecuteNUTBundle` script used to apply the bundle.
    /// @param _pre Standard NUT transactions to run before the bundle.
    /// @param _bundle Standard NUT transactions in the bundle.
    /// @param _post Wrapper transactions to run after the bundle.
    function executeWithWrappers(
        ExecuteNUTBundle _executeScript,
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _pre,
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _bundle,
        ExecuteNUTBundle.PostWrapperTxn[] memory _post
    )
        internal
    {
        if (_pre.length > 0) _executeScript.executeAll(_pre);
        _executeScript.executeAll(_bundle);
        for (uint256 i = 0; i < _post.length; i++) {
            _executeScript.executeWrapper(_post[i]);
        }
    }

    /// @notice Extracts the L2ContractsManager address from a parsed NUT bundle.
    /// @dev Relies on the `GenerateNUTBundle` invariant that the final phase pushes exactly one
    ///      `L2ProxyAdmin.upgradePredeploys(l2cm)` call as the last transaction.
    /// @param _txns Parsed NUT bundle transactions.
    /// @param _path Bundle path used in revert messages so failures point at the offending file.
    /// @return l2cm_ The decoded L2ContractsManager address.
    function extractL2CM(
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _txns,
        string memory _path
    )
        internal
        pure
        returns (address l2cm_)
    {
        if (_txns.length == 0) revert PastNUTBundles_EmptyBundle(_path);

        NetworkUpgradeTxns.NetworkUpgradeTxn memory finalTxn = _txns[_txns.length - 1];

        if (finalTxn.to != Predeploys.PROXY_ADMIN) revert PastNUTBundles_WrongTarget(_path, finalTxn.to);
        if (finalTxn.data.length != 36) revert PastNUTBundles_WrongDataLength(_path, finalTxn.data.length);

        bytes4 selector = bytes4(finalTxn.data);
        if (selector != IL2ProxyAdmin.upgradePredeploys.selector) {
            revert PastNUTBundles_WrongSelector(_path, selector);
        }

        l2cm_ = abi.decode(Bytes.slice(finalTxn.data, 4, 32), (address));
        if (l2cm_ == address(0)) revert PastNUTBundles_ZeroL2CM(_path);
    }

    /// @notice Fetches committed prior NUT bundles via the Go FFI and applies them in front of the
    ///         current bundle's expected state.
    /// @dev Accepts the already-generated current bundle transactions so parallel fork-test setup
    ///      does not re-read `current-upgrade-bundle.json` while another test is writing it.
    /// @param _currentTxns Parsed current bundle transactions.
    /// @param _executeScript Live `ExecuteNUTBundle` script used to apply prior bundles.
    function applyPastBundles(
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _currentTxns,
        ExecuteNUTBundle _executeScript
    )
        internal
    {
        applyPastBundles(_currentTxns, _executeScript, fetchPastBundles());
    }

    /// @notice Applies the given prior NUT bundles in front of the current bundle's expected state.
    /// @param _currentTxns Parsed current bundle transactions.
    /// @param _executeScript Live `ExecuteNUTBundle` script used to apply prior bundles.
    /// @param _entries Ordered prior bundle entries to apply.
    function applyPastBundles(
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _currentTxns,
        ExecuteNUTBundle _executeScript,
        NUTBundle[] memory _entries
    )
        internal
    {
        console.log("PastNUTBundles: %d prior bundle entry/entries discovered", _entries.length);

        for (uint256 i = 0; i < _entries.length; i++) {
            NUTBundle memory entry = _entries[i];
            NetworkUpgradeTxns.NetworkUpgradeTxn[] memory priorTxns = NetworkUpgradeTxns.readArtifact(entry.path);

            (bool skip, string memory reason, address priorL2CM) =
                _shouldSkipPriorBundle(entry, priorTxns, _currentTxns);
            if (skip) {
                console.log("PastNUTBundles: skipping fork=%s path=%s (%s)", entry.fork, entry.path, reason);
                continue;
            }

            console.log("PastNUTBundles: applying fork=%s path=%s L2CM=%s", entry.fork, entry.path, priorL2CM);
            ForkWrappers memory w = wrappersForFork(entry.fork);
            executeWithWrappers(_executeScript, w.pre, priorTxns, w.post);
        }
    }

    /// @notice Returns whether a prior bundle should be skipped before applying, along with the
    ///         prior bundle's L2CM when extraction was performed.
    /// @dev Future fork-specific exceptions should be added here rather than by de-duplicating
    ///      arbitrary transactions from the prior bundle.
    /// @return skip_ Whether the prior bundle should be skipped.
    /// @return reason_ Human-readable reason logged when `skip_` is true.
    /// @return priorL2CM_ The prior bundle's L2CM when not skipped via the Karst bootstrap rule;
    ///                    `address(0)` when bootstrap-skipped, since the caller does not need it.
    function _shouldSkipPriorBundle(
        NUTBundle memory _entry,
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _priorTxns,
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _currentTxns
    )
        internal
        view
        returns (bool skip_, string memory reason_, address priorL2CM_)
    {
        if (_shouldSkipKarstBootstrap(_entry, _priorTxns, _currentTxns)) {
            return (true, "current bundle contains same Karst direct CREATE2 bootstrap", address(0));
        }

        priorL2CM_ = extractL2CM(_priorTxns, _entry.path);
        address currentL2CM = extractL2CM(_currentTxns, Constants.CURRENT_BUNDLE_PATH);

        if (priorL2CM_ == currentL2CM) {
            // TODO(#19369): Remove this Karst exception once Karst is deployed on all chains.
            // Pre-Karst chains lack ConditionalDeployer. Matching L2CM does not prove the
            // bootstrap predeploy state exists, so they still need the Karst bundle.
            if (LibString.eq(_entry.fork, "karst") && !_isConditionalDeployerInitialized()) {
                return (false, "", priorL2CM_);
            }
            return (true, "L2CM matches current", priorL2CM_);
        }

        return (false, "", priorL2CM_);
    }

    /// @notice Returns true when the live fork has a usable ConditionalDeployer implementation.
    function _isConditionalDeployerInitialized() internal view returns (bool) {
        try ISemver(Predeploys.CONDITIONAL_DEPLOYER).version() returns (string memory) {
            return true;
        } catch {
            return false;
        }
    }

    /// @notice Returns true when the Karst bundle should be skipped because the current bundle
    ///         owns the same non-idempotent direct CREATE2 bootstrap transaction.
    /// @dev TODO(#19369): Remove this exception once Karst upgrade is deployed in all chains.
    function _shouldSkipKarstBootstrap(
        NUTBundle memory _entry,
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _priorTxns,
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _currentTxns
    )
        internal
        pure
        returns (bool)
    {
        if (!LibString.eq(_entry.fork, "karst")) return false;

        for (uint256 i = 0; i < _priorTxns.length; i++) {
            if (_priorTxns[i].to != Preinstalls.DeterministicDeploymentProxy) continue;

            bytes32 priorDataHash = keccak256(_priorTxns[i].data);
            for (uint256 j = 0; j < _currentTxns.length; j++) {
                if (_currentTxns[j].to != Preinstalls.DeterministicDeploymentProxy) continue;
                if (keccak256(_currentTxns[j].data) == priorDataHash) return true;
            }
        }

        return false;
    }
}
