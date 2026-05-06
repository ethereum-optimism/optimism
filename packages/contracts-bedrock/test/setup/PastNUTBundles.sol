// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Forge
import { console2 as console } from "forge-std/console2.sol";

// Scripts
import { ExecuteNUTBundle } from "scripts/upgrade/ExecuteNUTBundle.s.sol";
import { Process } from "scripts/libraries/Process.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Bytes } from "src/libraries/Bytes.sol";
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";

// Interfaces
import { IL2ProxyAdmin } from "interfaces/L2/IL2ProxyAdmin.sol";

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

    /// @notice Thrown when a bundle has no transactions and an L2CM cannot be extracted.
    error EmptyBundle(string path);

    /// @notice Thrown when the final transaction does not target `Predeploys.PROXY_ADMIN`.
    error WrongTarget(string path, address to);

    /// @notice Thrown when the final transaction calldata is not `upgradePredeploys(address)`.
    error WrongSelector(string path, bytes4 selector);

    /// @notice Thrown when the final transaction calldata is not the expected 36 bytes
    ///         (4-byte selector + 32-byte ABI-encoded address).
    error WrongDataLength(string path, uint256 length);

    /// @notice Thrown when the decoded L2CM address is the zero address.
    error ZeroL2CM(string path);

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
        if (_txns.length == 0) revert EmptyBundle(_path);

        NetworkUpgradeTxns.NetworkUpgradeTxn memory finalTxn = _txns[_txns.length - 1];

        if (finalTxn.to != Predeploys.PROXY_ADMIN) revert WrongTarget(_path, finalTxn.to);
        if (finalTxn.data.length != 36) revert WrongDataLength(_path, finalTxn.data.length);

        bytes4 selector = bytes4(finalTxn.data);
        if (selector != IL2ProxyAdmin.upgradePredeploys.selector) revert WrongSelector(_path, selector);

        l2cm_ = abi.decode(Bytes.slice(finalTxn.data, 4, 32), (address));
        if (l2cm_ == address(0)) revert ZeroL2CM(_path);
    }

    /// @notice Stages discovered prior NUT bundles in front of the current bundle's expected state.
    /// @dev Accepts the already-generated current bundle transactions so parallel fork-test setup
    ///      does not re-read `current-upgrade-bundle.json` while another test is writing it.
    function stagePastBundles(
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _currentTxns,
        ExecuteNUTBundle _executeScript
    )
        internal
    {
        stagePastBundlesAgainst(_currentTxns, _executeScript);
    }

    /// @notice Fetches and stages prior NUT bundles.
    /// @param _currentTxns Parsed current bundle transactions.
    /// @param _executeScript Live `ExecuteNUTBundle` script used to apply prior bundles.
    function stagePastBundlesAgainst(
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _currentTxns,
        ExecuteNUTBundle _executeScript
    )
        internal
    {
        NUTBundle[] memory entries = fetchPastBundles();
        stagePastBundlesAgainst(_currentTxns, _executeScript, entries);
    }

    /// @notice Stages the given prior NUT bundles.
    /// @param _currentTxns Parsed current bundle transactions.
    /// @param _executeScript Live `ExecuteNUTBundle` script used to apply prior bundles.
    /// @param _entries Ordered prior bundle entries to stage.
    function stagePastBundlesAgainst(
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

            (bool skip, string memory reason) = _shouldSkipPriorBundle(entry, priorTxns, _currentTxns);
            if (skip) {
                console.log("PastNUTBundles: skipping fork=%s path=%s (%s)", entry.fork, entry.path, reason);
                continue;
            }

            address priorL2CM = extractL2CM(priorTxns, entry.path);
            console.log("PastNUTBundles: staging fork=%s path=%s L2CM=%s", entry.fork, entry.path, priorL2CM);
            _executeScript.executeAll(priorTxns);
        }
    }

    /// @notice Returns whether a prior bundle should be skipped before staging.
    /// @dev Future fork-specific staging exceptions should be added here rather than by
    ///      de-duplicating arbitrary transactions from the prior bundle.
    function _shouldSkipPriorBundle(
        NUTBundle memory _entry,
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _priorTxns,
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _currentTxns
    )
        internal
        pure
        returns (bool skip_, string memory reason_)
    {
        address priorL2CM = extractL2CM(_priorTxns, _entry.path);
        address currentL2CM = extractL2CM(_currentTxns, Constants.CURRENT_BUNDLE_PATH);
        if (priorL2CM == currentL2CM) return (true, "L2CM matches current");

        if (_shouldSkipKarstBootstrap(_entry, _priorTxns, _currentTxns)) {
            return (true, "current bundle contains same Karst direct CREATE2 bootstrap");
        }

        return (false, "");
    }

    /// @notice Returns true when Karst staging should be skipped because the current bundle owns
    ///         the same non-idempotent direct CREATE2 bootstrap transaction.
    function _shouldSkipKarstBootstrap(
        NUTBundle memory _entry,
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _priorTxns,
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory _currentTxns
    )
        internal
        pure
        returns (bool)
    {
        if (keccak256(bytes(_entry.fork)) != keccak256(bytes("karst"))) return false;

        for (uint256 i = 0; i < _priorTxns.length; i++) {
            if (_priorTxns[i].to != Preinstalls.DeterministicDeploymentProxy) continue;

            bytes32 priorTxnHash = keccak256(abi.encode(_priorTxns[i].to, _priorTxns[i].data));
            for (uint256 j = 0; j < _currentTxns.length; j++) {
                if (_currentTxns[j].to != Preinstalls.DeterministicDeploymentProxy) continue;
                if (keccak256(abi.encode(_currentTxns[j].to, _currentTxns[j].data)) == priorTxnHash) return true;
            }
        }

        return false;
    }
}
