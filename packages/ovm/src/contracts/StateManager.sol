pragma solidity ^0.5.0;

/**
 * @title StateManager
 * @notice The StateManager is a simple abstraction which can be extended by either the Stateful client
 *         or the stateless client so that both can share the same Execution Manager.
 */
contract StateManager {
    // Block Metadata
    function getTimestamp() internal returns(uint);
    function getQueueOrigin() internal returns(uint);

    // Storage
    function getStorage(address contractAddress, bytes32 slot) internal returns(bytes32);
    function setStorage(address contractAddress, bytes32 slot, bytes32 value) internal;

    // Nonces (this is used during contract creation to determine the contract address)
    function getNonce(address contractAddress) internal returns(bytes32);
    function setNonce(address contractAddress, bytes32 value) internal;
    function incrementNonce(address contractAddress) internal;

    // Contract code storage / contract address retrieval
    function getCode(address contractAddress) internal returns(bytes memory);
    function getCodeHash(address contractAddress) internal returns(bytes32);
    function getCodeAddress(address contractAddress) internal returns(address);
    function setCode(address contractAddress, bytes memory code) internal;
}
