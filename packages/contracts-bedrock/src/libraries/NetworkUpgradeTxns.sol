// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Utilities
import { Vm } from "forge-std/Vm.sol";
import { stdJson } from "forge-std/StdJson.sol";

/// @title NetworkUpgradeTxns
/// @notice Standard library for generating Network Upgrade Transaction (NUT) artifacts.
///         Generates simplified JSON format that is converted to deposit transactions by op-node.
library NetworkUpgradeTxns {
    using stdJson for string;

    Vm private constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    /// @notice Represents a single Network Upgrade Transaction
    ///         This struct is serialized to JSON and later converted to a DepositTx by op-node.
    ///         See op-node/rollup/derive/parse_upgrade_transactions.go for conversion logic.
    /// @dev Fields MUST be in alphabetical order for JSON parseJson/abi.decode to work.
    /// @param data The calldata for the transaction.
    /// @param from The address of the sender of the transaction.
    /// @param gasLimit The gas limit for the transaction.
    /// @param intent Human-readable description of the transaction's purpose.
    /// @param to The address of the recipient of the transaction.
    struct NetworkUpgradeTxn {
        bytes data;
        address from;
        uint64 gasLimit;
        string intent;
        address to;
    }

    /// @notice Writes the transactions array to a JSON file.
    /// @param _txns The array of upgrade transactions.
    /// @param _outputPath The file path for the output JSON.
    function writeArtifact(NetworkUpgradeTxn[] memory _txns, string memory _outputPath) internal {
        string memory finalJson = "[";

        for (uint256 i = 0; i < _txns.length; i++) {
            string memory txnJson = serializeTxn(_txns[i], i);
            finalJson = string.concat(finalJson, txnJson);
            if (i < _txns.length - 1) {
                finalJson = string.concat(finalJson, ",");
            }
        }

        finalJson = string.concat(finalJson, "]");

        // Writes the final serialized JSON array to file.
        vm.writeJson(finalJson, _outputPath);
    }

    /// @notice Serializes a single transaction to JSON.
    /// @param _txn The transaction to serialize.
    /// @param _index The transaction index.
    /// @return serializedJson_ The serialized JSON string.
    function serializeTxn(
        NetworkUpgradeTxn memory _txn,
        uint256 _index
    )
        internal
        returns (string memory serializedJson_)
    {
        string memory key = vm.toString(_index);

        vm.serializeBytes(key, "data", _txn.data);
        vm.serializeAddress(key, "from", _txn.from);
        vm.serializeUint(key, "gasLimit", uint256(_txn.gasLimit));
        vm.serializeString(key, "intent", _txn.intent);
        serializedJson_ = vm.serializeAddress(key, "to", _txn.to);
    }

    /// @notice Reads upgrade transactions from a JSON file.
    /// @param _inputPath The file path for the input JSON.
    /// @return txns_ The array of upgrade transactions.
    function readArtifact(string memory _inputPath) internal view returns (NetworkUpgradeTxn[] memory txns_) {
        string memory json = vm.readFile(_inputPath);
        bytes memory parsedData = vm.parseJson(json);
        txns_ = abi.decode(parsedData, (NetworkUpgradeTxns.NetworkUpgradeTxn[]));
    }
}
