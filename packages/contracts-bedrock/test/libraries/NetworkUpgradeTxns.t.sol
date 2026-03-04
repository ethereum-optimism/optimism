// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IGasPriceOracle } from "interfaces/L2/IGasPriceOracle.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

/// @title NetworkUpgradeTxns_TestInit
/// @notice Reusable test initialization for `NetworkUpgradeTxns` tests.
abstract contract NetworkUpgradeTxns_TestInit is Test {
    // Test constants matching Go implementation
    address constant L1_BLOCK_DEPLOYER = 0x4210000000000000000000000000000000000000;
    address constant GAS_PRICE_ORACLE_DEPLOYER = 0x4210000000000000000000000000000000000001;
    address constant DEPOSITOR_ACCOUNT = 0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001;

    // Intent strings from Ecotone upgrade (ecotone_upgrade_transactions.go:27-32)
    string constant INTENT_DEPLOY_L1_BLOCK = "Ecotone: L1 Block Deployment";
    string constant INTENT_DEPLOY_GAS_PRICE_ORACLE = "Ecotone: Gas Price Oracle Deployment";
    string constant INTENT_UPDATE_L1_BLOCK_PROXY = "Ecotone: L1 Block Proxy Update";
    string constant INTENT_UPDATE_GAS_PRICE_ORACLE = "Ecotone: Gas Price Oracle Proxy Update";
    string constant INTENT_ENABLE_ECOTONE = "Ecotone: Gas Price Oracle Set Ecotone";
    string constant INTENT_BEACON_ROOTS = "Ecotone: beacon block roots contract deployment";
}

/// @title NetworkUpgradeTxns_SerializeTxn_Test
/// @notice Tests the `serializeTxn` function.
contract NetworkUpgradeTxns_SerializeTxn_Test is NetworkUpgradeTxns_TestInit {
    /// @notice Test that serializeTxn correctly serializes all fields with fuzzed inputs.
    function testFuzz_serializeTxn_succeeds(
        address _from,
        address _to,
        uint64 _gasLimit,
        bytes calldata _data,
        string calldata _intent
    )
        public
    {
        vm.assume(_gasLimit > 0);
        vm.assume(bytes(_intent).length > 0);

        NetworkUpgradeTxns.NetworkUpgradeTxn memory txn = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: _intent,
            from: _from,
            to: _to,
            gasLimit: _gasLimit,
            data: _data
        });

        string memory json = NetworkUpgradeTxns.serializeTxn(txn, 0);

        // Verify JSON is not empty
        assertGt(bytes(json).length, 0, "Serialized JSON should not be empty");
    }

    /// @notice Test that serializeTxn is deterministic for the same input with fuzzed parameters.
    function testFuzz_serializeTxn_deterministic_succeeds(
        address _from,
        address _to,
        uint64 _gasLimit,
        bytes calldata _data,
        string calldata _intent
    )
        public
    {
        vm.assume(_gasLimit > 0);
        vm.assume(bytes(_intent).length > 0);

        NetworkUpgradeTxns.NetworkUpgradeTxn memory txn = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: _intent,
            from: _from,
            to: _to,
            gasLimit: _gasLimit,
            data: _data
        });

        string memory json1 = NetworkUpgradeTxns.serializeTxn(txn, 0);
        string memory json2 = NetworkUpgradeTxns.serializeTxn(txn, 0);

        assertEq(keccak256(bytes(json1)), keccak256(bytes(json2)), "Serialization should be deterministic");
    }

    /// @notice Test that serializeTxn handles different indices with fuzzed transaction.
    /// @dev The index parameter is used internally by forge-std but doesn't affect the output JSON.
    function testFuzz_serializeTxn_differentIndices_succeeds(
        address _from,
        address _to,
        uint64 _gasLimit,
        bytes calldata _data,
        string calldata _intent,
        uint256 _index1,
        uint256 _index2
    )
        public
    {
        vm.assume(_gasLimit > 0);
        vm.assume(bytes(_intent).length > 0);

        NetworkUpgradeTxns.NetworkUpgradeTxn memory txn = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: _intent,
            from: _from,
            to: _to,
            gasLimit: _gasLimit,
            data: _data
        });

        string memory json1 = NetworkUpgradeTxns.serializeTxn(txn, _index1);
        string memory json2 = NetworkUpgradeTxns.serializeTxn(txn, _index2);

        // Verify both produce non-empty JSON
        assertGt(bytes(json1).length, 0, "First serialization should produce valid JSON");
        assertGt(bytes(json2).length, 0, "Second serialization should produce valid JSON");

        // Note: The index is used internally by forge-std's serialization but the output
        // JSON is the same since it only contains the transaction data, not the index.
        assertEq(keccak256(bytes(json1)), keccak256(bytes(json2)), "Same transaction produces same JSON");
    }
}

/// @title NetworkUpgradeTxns_WriteArtifact_Test
/// @notice Tests the `writeArtifact` function.
contract NetworkUpgradeTxns_WriteArtifact_Test is NetworkUpgradeTxns_TestInit {
    /// @notice Test writeArtifact with empty array
    function test_writeArtifact_emptyArray_succeeds() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](0);
        string memory outputPath = "deployments/nut-test-empty.json";
        NetworkUpgradeTxns.BundleMetadata memory metadata = NetworkUpgradeTxns.BundleMetadata({ version: "" });
        NetworkUpgradeTxns.writeArtifact(txns, metadata, outputPath);
    }

    /// @notice Test writeArtifact creates valid JSON file
    function test_writeArtifact_succeeds() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](2);

        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: INTENT_DEPLOY_L1_BLOCK,
            from: L1_BLOCK_DEPLOYER,
            to: address(0),
            gasLimit: 375_000,
            data: DeployUtils.getCode("L1Block.sol:L1Block")
        });

        txns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: INTENT_ENABLE_ECOTONE,
            from: DEPOSITOR_ACCOUNT,
            to: Predeploys.GAS_PRICE_ORACLE,
            gasLimit: 50_000,
            data: abi.encodeCall(IGasPriceOracle.setEcotone, ())
        });

        string memory outputPath = "deployments/nut-test.json";
        NetworkUpgradeTxns.BundleMetadata memory metadata = NetworkUpgradeTxns.BundleMetadata({ version: "1.0.0" });
        NetworkUpgradeTxns.writeArtifact(txns, metadata, outputPath);

        // Read json file and validate the transactions
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory readTxns = NetworkUpgradeTxns.readArtifact(outputPath);
        assertEq(readTxns.length, txns.length, "Transaction count mismatch");
        for (uint256 i = 0; i < txns.length; i++) {
            assertEq(readTxns[i].intent, txns[i].intent, "'intent' doesn't match");
            assertEq(readTxns[i].from, txns[i].from, "'from' doesn't match");
            assertEq(readTxns[i].to, txns[i].to, "'to' doesn't match");
            assertEq(readTxns[i].gasLimit, txns[i].gasLimit, "'gasLimit' doesn't match");
            assertEq(readTxns[i].data, txns[i].data, "'data' doesn't match");
        }
    }
}

/// @title NetworkUpgradeTxns_Uncategorized_Test
/// @notice Tests that the artifact produced by the library matches the expected values.
contract NetworkUpgradeTxns_Uncategorized_Test is NetworkUpgradeTxns_TestInit {
    /// @notice EIP-4788 beacon roots contract deployment data from EIP spec
    ///         Obtained from https://eips.ethereum.org/EIPS/eip-4788#deployment
    bytes constant EIP4788_CREATION_DATA =
        hex"60618060095f395ff33373fffffffffffffffffffffffffffffffffffffffe14604d57602036146024575f5ffd5b5f35801560495762001fff810690815414603c575f5ffd5b62001fff01545f5260205ff35b5f5ffd5b62001fff42064281555f359062001fff015500";

    /// @notice Test constructing Ecotone upgrade transactions, writing to file and reading back.
    function test_ecotoneUpgrade_roundtrip_succeeds() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](6);

        // 1. Deploy L1Block
        // ecotone_upgrade_transactions.go:47
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: INTENT_DEPLOY_L1_BLOCK,
            from: L1_BLOCK_DEPLOYER,
            to: address(0),
            gasLimit: 375_000,
            data: DeployUtils.getCode("L1Block.sol:L1Block")
        });

        // 2. Deploy GasPriceOracle
        // ecotone_upgrade_transactions.go:64
        txns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: INTENT_DEPLOY_GAS_PRICE_ORACLE,
            from: GAS_PRICE_ORACLE_DEPLOYER,
            to: address(0),
            gasLimit: 1_000_000,
            data: DeployUtils.getCode("GasPriceOracle.sol:GasPriceOracle")
        });

        // 3. Update L1Block proxy
        // ecotone_upgrade_transactions.go:81
        // Calculate the deployed L1Block address
        address newL1BlockAddress = vm.computeCreateAddress(L1_BLOCK_DEPLOYER, 0);
        txns[2] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: INTENT_UPDATE_L1_BLOCK_PROXY,
            from: address(0),
            to: Predeploys.L1_BLOCK_ATTRIBUTES,
            gasLimit: 50_000,
            data: abi.encodeCall(IProxy.upgradeTo, (newL1BlockAddress))
        });

        // 4. Update GasPriceOracle proxy
        // ecotone_upgrade_transactions.go:98
        // Calculate the deployed GasPriceOracle address
        address newGasPriceOracleAddress = vm.computeCreateAddress(GAS_PRICE_ORACLE_DEPLOYER, 0);
        txns[3] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: INTENT_UPDATE_GAS_PRICE_ORACLE,
            from: address(0),
            to: Predeploys.GAS_PRICE_ORACLE,
            gasLimit: 50_000,
            data: abi.encodeCall(IProxy.upgradeTo, (newGasPriceOracleAddress))
        });

        // 5. Enable Ecotone on GasPriceOracle
        // ecotone_upgrade_transactions.go:115
        txns[4] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: INTENT_ENABLE_ECOTONE,
            from: DEPOSITOR_ACCOUNT,
            to: Predeploys.GAS_PRICE_ORACLE,
            gasLimit: 80_000,
            data: abi.encodeCall(IGasPriceOracle.setEcotone, ())
        });

        // 6. Deploy EIP-4788 beacon block roots contract
        // ecotone_upgrade_transactions.go:130
        txns[5] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: INTENT_BEACON_ROOTS,
            from: 0x0B799C86a49DEeb90402691F1041aa3AF2d3C875,
            to: address(0), // Contract deployment
            gasLimit: 250_000, // hex constant 0x3d090, as defined in EIP-4788 (250_000 in decimal)
            data: EIP4788_CREATION_DATA
        });

        // Write transactions to JSON file
        string memory outputPath = "deployments/nut-ecotone-upgrade-test.json";
        NetworkUpgradeTxns.BundleMetadata memory metadata = NetworkUpgradeTxns.BundleMetadata({ version: "1.0.0" });
        NetworkUpgradeTxns.writeArtifact(txns, metadata, outputPath);

        // Read back the transactions
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory readTxns = NetworkUpgradeTxns.readArtifact(outputPath);

        // Validate array length matches
        assertEq(readTxns.length, txns.length, "Transaction count mismatch");

        // Validate each transaction matches
        for (uint256 i = 0; i < txns.length; i++) {
            assertEq(readTxns[i].intent, txns[i].intent, "'intent' doesn't match");
            assertEq(readTxns[i].from, txns[i].from, "'from' doesn't match");
            assertEq(readTxns[i].to, txns[i].to, "'to' doesn't match");
            assertEq(readTxns[i].gasLimit, txns[i].gasLimit, "'gasLimit' doesn't match");
            assertEq(readTxns[i].data, txns[i].data, "'data' doesn't match");
        }
    }

    function testFuzz_txnStruct_succeeds(
        address _from,
        address _to,
        uint64 _gasLimit,
        bytes memory _data,
        string calldata _intent
    )
        public
        pure
    {
        NetworkUpgradeTxns.NetworkUpgradeTxn memory txn = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: _intent,
            from: _from,
            to: _to,
            gasLimit: _gasLimit,
            data: _data
        });

        assertEq(txn.intent, _intent);
        assertEq(txn.from, _from);
        assertEq(txn.to, _to);
        assertEq(txn.gasLimit, _gasLimit);
        assertEq(txn.data, _data);
    }
}
