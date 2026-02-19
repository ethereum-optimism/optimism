// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IGasPriceOracle } from "interfaces/L2/IGasPriceOracle.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

// Testing
import { Test } from "forge-std/Test.sol";

// Libraries
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title NetworkUpgradeTxns_TestInit
/// @notice Reusable test initialization for `NetworkUpgradeTxns` tests.
abstract contract NetworkUpgradeTxns_TestInit is Test {
    // Test constants matching Go implementation
    address constant L1_BLOCK_DEPLOYER = 0x4210000000000000000000000000000000000000;
    address constant GAS_PRICE_ORACLE_DEPLOYER = 0x4210000000000000000000000000000000000001;
    address constant DEPOSITOR_ACCOUNT = 0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001;

    // Known source hashes from Ecotone upgrade (ecotone_upgrade_transactions_test.go:14-45)
    bytes32 constant DEPLOY_L1_BLOCK_HASH = 0x877a6077205782ea15a6dc8699fa5ebcec5e0f4389f09cb8eda09488231346f8;
    bytes32 constant DEPLOY_GAS_PRICE_ORACLE_HASH = 0xa312b4510adf943510f05fcc8f15f86995a5066bd83ce11384688ae20e6ecf42;
    bytes32 constant UPDATE_L1_BLOCK_PROXY_HASH = 0x18acb38c5ff1c238a7460ebc1b421fa49ec4874bdf1e0a530d234104e5e67dbc;
    bytes32 constant UPDATE_GAS_PRICE_ORACLE_HASH = 0xee4f9385eceef498af0be7ec5862229f426dec41c8d42397c7257a5117d9230a;
    bytes32 constant ENABLE_ECOTONE_HASH = 0x0c1cb38e99dbc9cbfab3bb80863380b0905290b37eb3d6ab18dc01c1f3e75f93;
    bytes32 constant BEACON_ROOTS_HASH = 0x69b763c48478b9dc2f65ada09b3d92133ec592ea715ec65ad6e7f3dc519dc00c;

    // Intent strings from Ecotone upgrade (ecotone_upgrade_transactions.go:27-32)
    string constant INTENT_DEPLOY_L1_BLOCK = "Ecotone: L1 Block Deployment";
    string constant INTENT_DEPLOY_GAS_PRICE_ORACLE = "Ecotone: Gas Price Oracle Deployment";
    string constant INTENT_UPDATE_L1_BLOCK_PROXY = "Ecotone: L1 Block Proxy Update";
    string constant INTENT_UPDATE_GAS_PRICE_ORACLE = "Ecotone: Gas Price Oracle Proxy Update";
    string constant INTENT_ENABLE_ECOTONE = "Ecotone: Gas Price Oracle Set Ecotone";
    string constant INTENT_BEACON_ROOTS = "Ecotone: beacon block roots contract deployment";
}

/// @title NetworkUpgradeTxns_SourceHash_Test
/// @notice Tests the `sourceHash` function matches Go implementation.
contract NetworkUpgradeTxns_SourceHash_Test is NetworkUpgradeTxns_TestInit {
    /// @notice Test sourceHash for L1Block deployment matches Go test vector
    function test_sourceHash_deployL1Block_succeeds() public pure {
        bytes32 hash = NetworkUpgradeTxns.sourceHash(INTENT_DEPLOY_L1_BLOCK);
        assertEq(hash, DEPLOY_L1_BLOCK_HASH, "L1Block deployment hash mismatch");
    }

    /// @notice Test sourceHash for GasPriceOracle deployment matches Go test vector
    function test_sourceHash_deployGasPriceOracle_succeeds() public pure {
        bytes32 hash = NetworkUpgradeTxns.sourceHash(INTENT_DEPLOY_GAS_PRICE_ORACLE);
        assertEq(hash, DEPLOY_GAS_PRICE_ORACLE_HASH, "GasPriceOracle deployment hash mismatch");
    }

    /// @notice Test sourceHash for L1Block proxy update matches Go test vector
    function test_sourceHash_updateL1BlockProxy_succeeds() public pure {
        bytes32 hash = NetworkUpgradeTxns.sourceHash(INTENT_UPDATE_L1_BLOCK_PROXY);
        assertEq(hash, UPDATE_L1_BLOCK_PROXY_HASH, "L1Block proxy update hash mismatch");
    }

    /// @notice Test sourceHash for GasPriceOracle proxy update matches Go test vector
    function test_sourceHash_updateGasPriceOracleProxy_succeeds() public pure {
        bytes32 hash = NetworkUpgradeTxns.sourceHash(INTENT_UPDATE_GAS_PRICE_ORACLE);
        assertEq(hash, UPDATE_GAS_PRICE_ORACLE_HASH, "GasPriceOracle proxy update hash mismatch");
    }

    /// @notice Test sourceHash for enable Ecotone matches Go test vector
    function test_sourceHash_enableEcotone_succeeds() public pure {
        bytes32 hash = NetworkUpgradeTxns.sourceHash(INTENT_ENABLE_ECOTONE);
        assertEq(hash, ENABLE_ECOTONE_HASH, "Enable Ecotone hash mismatch");
    }

    /// @notice Test sourceHash for beacon roots matches Go test vector
    function test_sourceHash_beaconRoots_succeeds() public pure {
        bytes32 hash = NetworkUpgradeTxns.sourceHash(INTENT_BEACON_ROOTS);
        assertEq(hash, BEACON_ROOTS_HASH, "Beacon roots hash mismatch");
    }
}

/// @title NetworkUpgradeTxns_WriteArtifact_Test
/// @notice Tests the `writeArtifact` function.
contract NetworkUpgradeTxns_WriteArtifact_Test is NetworkUpgradeTxns_TestInit {
    /// @notice Test writeArtifact with empty array
    function test_writeArtifact_emptyArray() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](0);
        string memory outputPath = "deployments/nut-test-empty.json";
        NetworkUpgradeTxns.writeArtifact(txns, outputPath);
    }

    /// @notice Test writeArtifact with single Predeploy deployment
    function test_writeArtifact_singleDeployment() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);

        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            sourceHash: NetworkUpgradeTxns.sourceHash(INTENT_DEPLOY_L1_BLOCK),
            from: L1_BLOCK_DEPLOYER,
            to: address(0),
            mint: 0,
            value: 0,
            gas: 375_000,
            isSystemTransaction: false,
            data: vm.getCode("L1Block.sol:L1Block")
        });
        string memory outputPath = "deployments/nut-test-single.json";
        NetworkUpgradeTxns.writeArtifact(txns, outputPath);
    }

    /// @notice Test writeArtifact creates valid JSON file
    function test_writeArtifact_succeeds() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](2);

        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            sourceHash: NetworkUpgradeTxns.sourceHash(INTENT_DEPLOY_L1_BLOCK),
            from: L1_BLOCK_DEPLOYER,
            to: address(0),
            mint: 0,
            value: 0,
            gas: 375_000,
            isSystemTransaction: false,
            data: vm.getCode("L1Block.sol:L1Block")
        });

        txns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            sourceHash: NetworkUpgradeTxns.sourceHash(INTENT_ENABLE_ECOTONE),
            from: DEPOSITOR_ACCOUNT,
            to: Predeploys.GAS_PRICE_ORACLE,
            mint: 0,
            value: 0,
            gas: 50_000,
            isSystemTransaction: false,
            data: abi.encodeCall(IGasPriceOracle.setEcotone, ())
        });

        string memory outputPath = "deployments/nut-test.json";
        NetworkUpgradeTxns.writeArtifact(txns, outputPath);

        // Read json file and validate the transactions
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory readTxns = NetworkUpgradeTxns.readArtifact(outputPath);
        assertEq(readTxns.length, txns.length, "Transaction count mismatch");
        for (uint256 i = 0; i < txns.length; i++) {
            assertEq(readTxns[i].sourceHash, txns[i].sourceHash, "'sourceHash' doesn't match");
            assertEq(readTxns[i].from, txns[i].from, "'from' doesn't match");
            assertEq(readTxns[i].to, txns[i].to, "'to' doesn't match");
            assertEq(readTxns[i].mint, txns[i].mint, "'mint' doesn't match");
        }
    }
}

/// @title NetworkUpgradeTxns_EcotoneUpgrade_Test
/// @notice Tests that the artifact produced by the library matches the expected values.
contract NetworkUpgradeTxns_EcotoneUpgrade_Test is NetworkUpgradeTxns_TestInit {
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
            sourceHash: NetworkUpgradeTxns.sourceHash(INTENT_DEPLOY_L1_BLOCK),
            from: L1_BLOCK_DEPLOYER,
            to: address(0),
            mint: 0,
            value: 0,
            gas: 375_000,
            isSystemTransaction: false,
            data: vm.getCode("L1Block.sol:L1Block")
        });

        // 2. Deploy GasPriceOracle
        // ecotone_upgrade_transactions.go:64
        txns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            sourceHash: NetworkUpgradeTxns.sourceHash(INTENT_DEPLOY_GAS_PRICE_ORACLE),
            from: GAS_PRICE_ORACLE_DEPLOYER,
            to: address(0),
            mint: 0,
            value: 0,
            gas: 1_000_000,
            isSystemTransaction: false,
            data: vm.getCode("GasPriceOracle.sol:GasPriceOracle")
        });

        // 3. Update L1Block proxy
        // ecotone_upgrade_transactions.go:81
        // Calculate the deployed L1Block address
        address newL1BlockAddress = vm.computeCreateAddress(L1_BLOCK_DEPLOYER, 0);
        txns[2] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            sourceHash: NetworkUpgradeTxns.sourceHash(INTENT_UPDATE_L1_BLOCK_PROXY),
            from: address(0),
            to: Predeploys.L1_BLOCK_ATTRIBUTES,
            mint: 0,
            value: 0,
            gas: 50_000,
            isSystemTransaction: false,
            data: abi.encodeCall(IProxy.upgradeTo, (newL1BlockAddress))
        });

        // 4. Update GasPriceOracle proxy
        // ecotone_upgrade_transactions.go:98
        // Calculate the deployed GasPriceOracle address
        address newGasPriceOracleAddress = vm.computeCreateAddress(GAS_PRICE_ORACLE_DEPLOYER, 0);
        txns[3] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            sourceHash: NetworkUpgradeTxns.sourceHash(INTENT_UPDATE_GAS_PRICE_ORACLE),
            from: address(0),
            to: Predeploys.GAS_PRICE_ORACLE,
            mint: 0,
            value: 0,
            gas: 50_000,
            isSystemTransaction: false,
            data: abi.encodeCall(IProxy.upgradeTo, (newGasPriceOracleAddress))
        });

        // 5. Enable Ecotone on GasPriceOracle
        // ecotone_upgrade_transactions.go:115
        txns[4] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            sourceHash: NetworkUpgradeTxns.sourceHash(INTENT_ENABLE_ECOTONE),
            from: DEPOSITOR_ACCOUNT,
            to: Predeploys.GAS_PRICE_ORACLE,
            mint: 0,
            value: 0,
            gas: 80_000,
            isSystemTransaction: false,
            data: abi.encodeCall(IGasPriceOracle.setEcotone, ())
        });

        // 6. Deploy EIP-4788 beacon block roots contract
        // ecotone_upgrade_transactions.go:130
        txns[5] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            sourceHash: NetworkUpgradeTxns.sourceHash(INTENT_BEACON_ROOTS),
            from: 0x0B799C86a49DEeb90402691F1041aa3AF2d3C875,
            to: address(0), // Contract deployment
            mint: 0,
            value: 0,
            gas: 250_000, // hex constant 0x3d090, as defined in EIP-4788 (250_000 in decimal)
            isSystemTransaction: false,
            data: EIP4788_CREATION_DATA
        });

        // Write transactions to JSON file
        string memory outputPath = "deployments/nut-ecotone-upgrade-test.json";
        NetworkUpgradeTxns.writeArtifact(txns, outputPath);

        // Read back the transactions
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory readTxns = NetworkUpgradeTxns.readArtifact(outputPath);

        // Validate array length matches
        assertEq(readTxns.length, txns.length, "Transaction count mismatch");

        // Validate each transaction matches
        for (uint256 i = 0; i < txns.length; i++) {
            assertEq(readTxns[i].sourceHash, txns[i].sourceHash, "'sourceHash' doesn't match");
            assertEq(readTxns[i].from, txns[i].from, "'from' doesn't match");
            assertEq(readTxns[i].to, txns[i].to, "'to' doesn't match");
            assertEq(readTxns[i].mint, txns[i].mint, "'mint' doesn't match");
            assertEq(readTxns[i].value, txns[i].value, "'value' doesn't match");
            assertEq(readTxns[i].gas, txns[i].gas, "'gas' doesn't match");
            assertEq(
                readTxns[i].isSystemTransaction, txns[i].isSystemTransaction, "'isSystemTransaction' doesn't match"
            );
            assertEq(readTxns[i].data, txns[i].data, "'data' doesn't match");
        }
    }
}
