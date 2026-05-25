// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { console2 as console } from "forge-std/console2.sol";
import { Vm } from "forge-std/Vm.sol";

// Scripts
import { Config } from "scripts/libraries/Config.sol";

// Libraries
import { LibString } from "@solady/utils/LibString.sol";
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { Process } from "scripts/libraries/Process.sol";

// Reuse all test logic from L2ForkUpgrade — only setUp differs
import {
    L2ForkUpgrade_TestInit,
    L2ForkUpgrade_Versions_Test,
    L2ForkUpgrade_Initialization_Test,
    L2ForkUpgrade_Implementations_Test,
    L2ForkUpgrade_Events_Test
} from "test/L2/fork/L2ForkUpgrade.t.sol";

/// @title L2VerifyBetanetForkUpgrade_TestInit
/// @notice Provides a setUp for the L2 verify betanet fork upgrade tests.
///         The tests are supposed to run on forks where the last L2CM activation block
///         used the current NUT bundle from the repository.
contract L2VerifyBetanetForkUpgrade_TestInit is L2ForkUpgrade_TestInit {
    function setUp() public virtual override(L2ForkUpgrade_TestInit) {
        vm.skip(!Config.l2CMActivationTest());
        super.setUp();
    }

    /// @notice instead of executing the bundle on the fork, it overrides the execution
    ///         by going to a block after the activation block.
    function _executeCurrentBundle() internal virtual override {
        uint256 l2BlockAfterFork = Config.l2BlockAfterFork();
        if (l2BlockAfterFork == 0) {
            vm.createSelectFork(Config.l2ForkRpcUrl());
        } else {
            vm.createSelectFork(Config.l2ForkRpcUrl(), l2BlockAfterFork);
        }
        console.log("Setup: L2 fork switched to after the fork!");
    }
}

/// @title L2VerifyBetanetForkUpgrade_Versions_Test
/// @notice Tests that all predeploy versions were updated during the betanet activation.
contract L2VerifyBetanetForkUpgrade_Versions_Test is
    L2VerifyBetanetForkUpgrade_TestInit,
    L2ForkUpgrade_Versions_Test
{
    function setUp() public override(L2VerifyBetanetForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2VerifyBetanetForkUpgrade_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2VerifyBetanetForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2VerifyBetanetForkUpgrade_TestInit._executeCurrentBundle();
    }
}

/// @title L2VerifyBetanetForkUpgrade_Initialization_Test
/// @notice Tests that all initialization configurations were preserved during the betanet activation.
contract L2VerifyBetanetForkUpgrade_Initialization_Test is
    L2VerifyBetanetForkUpgrade_TestInit,
    L2ForkUpgrade_Initialization_Test
{
    function setUp() public override(L2VerifyBetanetForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2VerifyBetanetForkUpgrade_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2VerifyBetanetForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2VerifyBetanetForkUpgrade_TestInit._executeCurrentBundle();
    }
}

/// @title L2VerifyBetanetForkUpgrade_Implementations_Test
/// @notice Tests that all predeploy implementations were correctly upgraded during the betanet activation.
contract L2VerifyBetanetForkUpgrade_Implementations_Test is
    L2VerifyBetanetForkUpgrade_TestInit,
    L2ForkUpgrade_Implementations_Test
{
    function setUp() public override(L2VerifyBetanetForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2VerifyBetanetForkUpgrade_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2VerifyBetanetForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2VerifyBetanetForkUpgrade_TestInit._executeCurrentBundle();
    }

    /// @notice Tests that all predeploy implementations match expected addresses and have code.
    function test_l2ForkUpgrade_implementationsMatch_succeeds() public override(L2ForkUpgrade_Implementations_Test) {
        // Skip if running with an unoptimized Foundry profile
        skipIfUnoptimized();

        // Pre-capture expected implementations before any fork switch:
        // in activation mode vm.createSelectFork creates a new fork where generateScript
        // (deployed on fork 0) is not accessible.
        PredeployState[] memory predeploys = _getPreUpgradePredeploys();
        address[] memory expectedImpls = _getExpectedImpls(predeploys);

        // Execute bundle on forked L2
        _executeCurrentBundle();

        // Verify each predeploy's implementation
        _verifyImplementations(predeploys, expectedImpls);
    }
}

/// @title L2VerifyBetanetForkUpgrade_Events_Test
/// @notice Tests that all predeploy proxies emit the Upgraded event during the betanet activation.
contract L2VerifyBetanetForkUpgrade_Events_Test is L2VerifyBetanetForkUpgrade_TestInit, L2ForkUpgrade_Events_Test {
    function setUp() public override(L2VerifyBetanetForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2VerifyBetanetForkUpgrade_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2VerifyBetanetForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2VerifyBetanetForkUpgrade_TestInit._executeCurrentBundle();
    }

    function test_l2ForkUpgrade_upgradeEventsEmitted_succeeds() public override(L2ForkUpgrade_Events_Test) {
        // Skip if running with an unoptimized Foundry profile
        skipIfUnoptimized();

        // Pre-capture everything from generateScript before any fork switch:
        // in activation mode vm.createSelectFork creates a new fork where generateScript
        // (deployed on fork 0) is not accessible.
        address storageSetterImpl = generateScript.findImplByName("StorageSetter");
        PredeployState[] memory predeploys = _getPreUpgradePredeploys();
        address[] memory expectedImpls = _getExpectedImpls(predeploys);

        // Execute upgrade bundle
        _executeCurrentBundle();

        // Get all recorded logs
        Vm.Log[] memory logs = _getLogs(predeploys);

        // Verify each predeploy emitted the Upgraded event
        _verifyEvents(predeploys, logs, expectedImpls, storageSetterImpl);
    }

    function _getLogs(PredeployState[] memory predeploys) internal returns (Vm.Log[] memory logs_) {
        bytes32[] memory topics = new bytes32[](1);
        uint256 activationBlockNumber = Config.l2ForkBlockNumber() + 1;
        topics[0] = UPGRADED_EVENT_TOPIC;
        // vm.eth_getLogs serializes address(0) as the literal zero address rather than null,
        // so passing address(0) returns no events. Query each predeploy address individually.
        uint256 totalCount = 0;
        Vm.EthGetLogs[][] memory perDeployLogs = new Vm.EthGetLogs[][](predeploys.length);
        for (uint256 p = 0; p < predeploys.length; p++) {
            perDeployLogs[p] =
                vm.eth_getLogs(activationBlockNumber, activationBlockNumber, predeploys[p].predeploy, topics);
            totalCount += perDeployLogs[p].length;
        }
        logs_ = new Vm.Log[](totalCount);
        uint256 flatIdx = 0;
        for (uint256 p = 0; p < predeploys.length; p++) {
            for (uint256 q = 0; q < perDeployLogs[p].length; q++) {
                Vm.EthGetLogs memory ethLog = perDeployLogs[p][q];
                logs_[flatIdx].topics = ethLog.topics;
                logs_[flatIdx].data = ethLog.data;
                logs_[flatIdx].emitter = ethLog.emitter;
                flatIdx++;
            }
        }
    }
}

/// @title L2VerifyBetanetForkUpgrade_ActivationBlockTxns_Test
/// @notice Verifies the activation block contains the expected NUT bundle transactions.
contract L2VerifyBetanetForkUpgrade_ActivationBlockTxns_Test is L2VerifyBetanetForkUpgrade_TestInit {
    function setUp() public override(L2VerifyBetanetForkUpgrade_TestInit) {
        L2VerifyBetanetForkUpgrade_TestInit.setUp();
    }

    /// @notice Fetches the activation block via RPC and asserts that the NUT bundle transactions
    ///         are present in the correct order with the correct from, to, and calldata.
    function test_l2VerifyBetanetForkUpgrade_activationBlockContainsNUTBundle_succeeds() public {
        // Skip if running with an unoptimized Foundry profile
        skipIfUnoptimized();

        // Execute bundle on forked L2
        _executeCurrentBundle();

        // activationBlock is the first block after the pre-fork snapshot where the NUT bundle runs.
        uint256 activationBlock = Config.l2ForkBlockNumber() + 1;
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory bundleTxns = _currentBundleTxns();

        string memory blockHex = string.concat("0x", LibString.toHexStringNoPrefix(activationBlock));
        string memory rpcUrl = Config.l2ForkRpcUrl();

        // vm.rpc ABI-encodes block objects; use FFI (cast) to fetch JSON for parseJson.
        string memory blockJsonHashes = _ffiGetBlockByNumber(blockHex, rpcUrl, false);
        uint256 txCount = _activationBlockTransactionCount(blockJsonHashes);

        string memory blockJson = _ffiGetBlockByNumber(blockHex, rpcUrl, true);
        (address[] memory froms, address[] memory tos, bytes[] memory inputs) =
            _parseActivationBlockTransactions(blockJson, txCount);

        // The activation block also contains the L1 attributes deposit (and potentially other
        // system transactions) before the NUT bundle. Find the bundle start by matching the
        // first bundle transaction's from+to.
        uint256 bundleStart = _findBundleStart(froms, tos, inputs, bundleTxns[0]);

        assertGe(
            froms.length - bundleStart,
            bundleTxns.length,
            "Activation block does not contain enough transactions for the full NUT bundle"
        );

        for (uint256 i = 0; i < bundleTxns.length; i++) {
            uint256 blockIdx = bundleStart + i;
            assertEq(
                froms[blockIdx],
                bundleTxns[i].from,
                string.concat("from mismatch at bundle index ", vm.toString(i), ": ", bundleTxns[i].intent)
            );
            assertEq(
                tos[blockIdx],
                bundleTxns[i].to,
                string.concat("to mismatch at bundle index ", vm.toString(i), ": ", bundleTxns[i].intent)
            );
            assertEq(
                keccak256(inputs[blockIdx]),
                keccak256(bundleTxns[i].data),
                string.concat("data mismatch at bundle index ", vm.toString(i), ": ", bundleTxns[i].intent)
            );
        }
    }

    /// @notice Fetches a block via `cast rpc` (FFI). Returns JSON suitable for `vm.parseJson`.
    function _ffiGetBlockByNumber(
        string memory _blockHex,
        string memory _rpcUrl,
        bool _fullTxs
    )
        internal
        returns (string memory blockJson_)
    {
        string[] memory cmds = new string[](7);
        cmds[0] = "cast";
        cmds[1] = "rpc";
        cmds[2] = "eth_getBlockByNumber";
        cmds[3] = _blockHex;
        cmds[4] = _fullTxs ? "true" : "false";
        cmds[5] = "--rpc-url";
        cmds[6] = _rpcUrl;
        blockJson_ = string(Process.run(cmds));
    }

    /// @notice Returns the number of transactions in an activation block JSON payload (hash-only).
    function _activationBlockTransactionCount(string memory _blockJson) internal pure returns (uint256 count_) {
        string[] memory hashes = vm.parseJsonStringArray(_blockJson, ".transactions");
        return hashes.length;
    }

    /// @notice Parses from/to/input for each transaction in a full activation block JSON payload.
    function _parseActivationBlockTransactions(
        string memory _blockJson,
        uint256 _txCount
    )
        internal
        pure
        returns (address[] memory froms_, address[] memory tos_, bytes[] memory inputs_)
    {
        froms_ = new address[](_txCount);
        tos_ = new address[](_txCount);
        inputs_ = new bytes[](_txCount);

        for (uint256 i = 0; i < _txCount; i++) {
            string memory base = string.concat(".transactions[", vm.toString(i), "]");
            froms_[i] = vm.parseJsonAddress(_blockJson, string.concat(base, ".from"));
            tos_[i] = vm.parseJsonAddress(_blockJson, string.concat(base, ".to"));
            inputs_[i] = vm.parseJsonBytes(_blockJson, string.concat(base, ".input"));
        }
    }

    /// @notice Returns the block transaction index where the NUT bundle starts, identified by
    ///         matching the first bundle transaction's from+to pair.
    function _findBundleStart(
        address[] memory _froms,
        address[] memory _tos,
        bytes[] memory _inputs,
        NetworkUpgradeTxns.NetworkUpgradeTxn memory _firstBundleTxn
    )
        internal
        pure
        returns (uint256)
    {
        for (uint256 i = 0; i < _froms.length; i++) {
            if (
                _froms[i] == _firstBundleTxn.from && _tos[i] == _firstBundleTxn.to
                    && keccak256(_inputs[i]) == keccak256(_firstBundleTxn.data)
            ) {
                return i;
            }
        }
        revert("L2VerifyBetanetForkUpgrade_ActivationBlockTxns_Test: NUT bundle start not found in activation block");
    }
}
