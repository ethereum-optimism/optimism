// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Utilities
import { Vm } from "forge-std/Vm.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { console } from "forge-std/console.sol";

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
    /// @dev Fields MUST be in alphabetical order for JSON parseJson/abi.decode to work.
    /// @param data The data of the transaction.
    /// @param from The address of the sender of the transaction.
    /// @param gas The gas limit for the transaction.
    /// @param isSystemTransaction Whether this transaction is exempt from the L2 gas limit.
    /// @param mint The amount of ETH to mint on L2, locked on L1, 0 if no minting.
    /// @param sourceHash The hash that uniquely identifies the source of the deposit.
    /// @param to The address of the recipient of the transaction, address zero means contract creation.
    /// @param value The amount of ETH to transfer from L2 balance, executed after mint (if any).
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

    /// @notice Creates an upgrade transaction.
    /// @param _intent Human-readable intent.
    /// @param _from The sender address.
    /// @param _to The target address.
    /// @param _mint The mint amount.
    /// @param _value The value to send.
    /// @param _gas The gas limit.
    /// @param _isSystemTransaction Whether this is a system transaction.
    /// @param _data The transaction data.
    /// @return nut_ The upgrade transaction struct.
    function newTx(
        string memory _intent,
        address _from,
        address _to,
        uint256 _mint,
        uint256 _value,
        uint64 _gas,
        bool _isSystemTransaction,
        bytes memory _data
    )
        internal
        pure
        returns (NetworkUpgradeTxn memory nut_)
    {
        nut_ = NetworkUpgradeTxn({
            data: _data,
            from: _from,
            gas: _gas,
            isSystemTransaction: _isSystemTransaction,
            mint: _mint,
            sourceHash: sourceHash(_intent),
            to: _to,
            value: _value
        });
    }

    /// @notice Calculates the source hash for an upgrade transaction.
    /// @param _intent Human-readable intent string.
    /// @return sourceHash_ The computed source hash.
    function sourceHash(string memory _intent) internal pure returns (bytes32 sourceHash_) {
        bytes32 intentHash = keccak256(bytes(_intent));
        bytes memory domainInput = new bytes(64);

        assembly {
            mstore(add(domainInput, 56), shl(192, UPGRADE_DEPOSIT_SOURCE_DOMAIN))
            mstore(add(domainInput, 64), intentHash)
        }

        sourceHash_ = keccak256(domainInput);
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
        vm.serializeUint(key, "gas", uint256(_txn.gas));
        vm.serializeBool(key, "isSystemTransaction", _txn.isSystemTransaction);
        vm.serializeUint(key, "mint", _txn.mint);
        vm.serializeBytes32(key, "sourceHash", _txn.sourceHash);
        vm.serializeAddress(key, "to", _txn.to);
        serializedJson_ = vm.serializeUint(key, "value", _txn.value);
        console.log(serializedJson_);
    }

    /// @notice Reads upgrade transactions from a JSON file.
    /// @param _inputPath The file path for the input JSON.
    /// @return txns_ The array of upgrade transactions.
    function readArtifact(string memory _inputPath) internal view returns (NetworkUpgradeTxn[] memory txns_) {
        string memory json = vm.readFile(_inputPath);
        bytes memory parsedData = vm.parseJson(json);
        console.logBytes(parsedData);
        txns_ = abi.decode(parsedData, (NetworkUpgradeTxns.NetworkUpgradeTxn[]));
    }
}
