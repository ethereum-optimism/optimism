// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Vm } from "forge-std/Vm.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { console } from "forge-std/console.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

/// @title NetworkUpgradeTxns
/// @notice Standard library for generating Network Upgrade Transaction (NUT) artifacts.
///         Provides minimal interface to create DepositTx-compatible transaction metadata.
library NetworkUpgradeTxns {
    using stdJson for string;

    Vm private constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    /// @notice Source domain for upgrade transactions
    uint64 internal constant UPGRADE_DEPOSIT_SOURCE_DOMAIN = 2;

    /// @notice Represents a single Network Upgrade Transaction
    ///         Maps to the fields of the `DepositTx` struct defined in
    ///         https://github.com/ethereum-optimism/op-geth/blob/optimism/core/types/deposit_tx.go
    struct NetworkUpgradeTxn {
        bytes data;
        address from;
        uint64 gas;
        bool isSystemTransaction;
        uint256 mint;
        bytes32 sourceHash;
        address to;
        uint256 value;
    }

    /// @notice Create an upgrade transaction
    /// @param intent Human-readable intent
    /// @param from Sender address
    /// @param to Target address
    /// @param mint Mint amount
    /// @param value Value to send
    /// @param gas Gas limit
    /// @param isSystemTransaction Whether this is a system transaction
    /// @param data Transaction data
    /// @return Upgrade transaction struct
    function newTx(
        string memory intent,
        address from,
        address to,
        uint256 mint,
        uint256 value,
        uint64 gas,
        bool isSystemTransaction,
        bytes memory data
    )
        internal
        pure
        returns (NetworkUpgradeTxn memory)
    {
        return NetworkUpgradeTxn({
            sourceHash: sourceHash(intent),
            from: from,
            to: to,
            mint: mint,
            value: value,
            gas: gas,
            isSystemTransaction: isSystemTransaction,
            data: data
        });
    }

    /// @notice Calculate source hash for an upgrade transaction
    /// @param intent Human-readable intent string
    /// @return Source hash
    function sourceHash(string memory intent) internal pure returns (bytes32) {
        bytes32 intentHash = keccak256(bytes(intent));
        bytes memory domainInput = new bytes(64);

        assembly {
            mstore(add(domainInput, 56), shl(192, UPGRADE_DEPOSIT_SOURCE_DOMAIN))
            mstore(add(domainInput, 64), intentHash)
        }

        return keccak256(domainInput);
    }

    /// @notice Write transactions array to JSON file
    /// @param txns Array of upgrade transactions
    /// @param outputPath File path for output JSON
    function writeArtifact(NetworkUpgradeTxn[] memory txns, string memory outputPath) internal {
        string memory finalJson = "[";

        for (uint256 i = 0; i < txns.length; i++) {
            string memory txnJson = serializeTxn(txns[i], i);
            finalJson = string.concat(finalJson, txnJson);
            if (i < txns.length - 1) {
                finalJson = string.concat(finalJson, ",");
            }
        }

        finalJson = string.concat(finalJson, "]");

        // Write the final serialized JSON array to file
        vm.writeJson(finalJson, outputPath);
    }

    /// @notice Serialize a single transaction to JSON
    /// @param txn Transaction to serialize
    /// @param index Transaction index
    /// @return JSON string
    function serializeTxn(NetworkUpgradeTxn memory txn, uint256 index) internal returns (string memory) {
        string memory key = vm.toString(index);

        vm.serializeBytes32(key, "sourceHash", txn.sourceHash);
        vm.serializeAddress(key, "from", txn.from);
        vm.serializeAddress(key, "to", txn.to);
        vm.serializeUint(key, "mint", txn.mint);
        vm.serializeUint(key, "value", txn.value);
        vm.serializeUint(key, "gas", uint256(txn.gas));
        vm.serializeBool(key, "isSystemTransaction", txn.isSystemTransaction);
        return vm.serializeBytes(key, "data", txn.data);
    }

    /// @notice Helper function to read upgrade transactions from JSON file
    /// @param _inputPath File path for input JSON
    /// @return Array of upgrade transactions
    function readArtifact(string memory _inputPath)
        internal
        view
        returns (NetworkUpgradeTxns.NetworkUpgradeTxn[] memory)
    {
        string memory json = vm.readFile(_inputPath);
        bytes memory parsedData = vm.parseJson(json);
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns =
            abi.decode(parsedData, (NetworkUpgradeTxns.NetworkUpgradeTxn[]));
        return txns;
    }
}
