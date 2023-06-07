// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.10;

/**
 * @title Enum - Collection of enums used in Safe contracts.
 * @author Richard Meissner - @rmeissner
 */
abstract contract Enum {
    enum Operation {
        Call,
        DelegateCall
    }
}

/**
 * @title IGnosisSafe - Gnosis Safe Interface
 */
interface IGnosisSafe {
    event AddedOwner(address owner);
    event ApproveHash(bytes32 indexed approvedHash, address indexed owner);
    event ChangedFallbackHandler(address handler);
    event ChangedGuard(address guard);
    event ChangedThreshold(uint256 threshold);
    event DisabledModule(address module);
    event EnabledModule(address module);
    event ExecutionFailure(bytes32 txHash, uint256 payment);
    event ExecutionFromModuleFailure(address indexed module);
    event ExecutionFromModuleSuccess(address indexed module);
    event ExecutionSuccess(bytes32 txHash, uint256 payment);
    event RemovedOwner(address owner);
    event SafeReceived(address indexed sender, uint256 value);
    event SafeSetup(
        address indexed initiator, address[] owners, uint256 threshold, address initializer, address fallbackHandler
    );
    event SignMsg(bytes32 indexed msgHash);

    function VERSION() external view returns (string memory);
    function addOwnerWithThreshold(address owner, uint256 _threshold) external;
    function approveHash(bytes32 hashToApprove) external;
    function approvedHashes(address, bytes32) external view returns (uint256);
    function changeThreshold(uint256 _threshold) external;
    function checkNSignatures(bytes32 dataHash, bytes memory data, bytes memory signatures, uint256 requiredSignatures)
        external
        view;
    function checkSignatures(bytes32 dataHash, bytes memory data, bytes memory signatures) external view;
    function disableModule(address prevModule, address module) external;
    function domainSeparator() external view returns (bytes32);
    function enableModule(address module) external;
    function encodeTransactionData(
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation,
        uint256 safeTxGas,
        uint256 baseGas,
        uint256 gasPrice,
        address gasToken,
        address refundReceiver,
        uint256 _nonce
    ) external view returns (bytes memory);
    function execTransaction(
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation,
        uint256 safeTxGas,
        uint256 baseGas,
        uint256 gasPrice,
        address gasToken,
        address refundReceiver,
        bytes memory signatures
    ) external payable returns (bool success);
    function execTransactionFromModule(address to, uint256 value, bytes memory data, Enum.Operation operation)
        external
        returns (bool success);
    function execTransactionFromModuleReturnData(address to, uint256 value, bytes memory data, Enum.Operation operation)
        external
        returns (bool success, bytes memory returnData);
    function getChainId() external view returns (uint256);
    function getModulesPaginated(address start, uint256 pageSize)
        external
        view
        returns (address[] memory array, address next);
    function getOwners() external view returns (address[] memory);
    function getStorageAt(uint256 offset, uint256 length) external view returns (bytes memory);
    function getThreshold() external view returns (uint256);
    function getTransactionHash(
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation,
        uint256 safeTxGas,
        uint256 baseGas,
        uint256 gasPrice,
        address gasToken,
        address refundReceiver,
        uint256 _nonce
    ) external view returns (bytes32);
    function isModuleEnabled(address module) external view returns (bool);
    function isOwner(address owner) external view returns (bool);
    function nonce() external view returns (uint256);
    function removeOwner(address prevOwner, address owner, uint256 _threshold) external;
    function requiredTxGas(address to, uint256 value, bytes memory data, Enum.Operation operation) external returns (uint256);
    function setFallbackHandler(address handler) external;
    function setGuard(address guard) external;
    function setup(
        address[] memory _owners,
        uint256 _threshold,
        address to,
        bytes memory data,
        address fallbackHandler,
        address paymentToken,
        uint256 payment,
        address paymentReceiver
    ) external;
    function signedMessages(bytes32) external view returns (uint256);
    function simulateAndRevert(address targetContract, bytes memory calldataPayload) external;
    function swapOwner(address prevOwner, address oldOwner, address newOwner) external;
}
