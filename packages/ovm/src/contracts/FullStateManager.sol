pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {StateManager} from "./StateManager.sol";

/**
 * @title FullStateManager
 * @notice The FullStateManager is used for off-chain tx evaluation. It holds a complete mapping
 *         of all chain storage.
 */
contract FullStateManager is StateManager {
    mapping(address=>mapping(bytes32=>bytes32)) contractStorage;

    // Storage
    function getStorage(address contractAddress, bytes32 slot) internal view returns(bytes32) {
        return contractStorage[contractAddress][slot];
    }
    function setStorage(address contractAddress, bytes32 slot, bytes32 value) internal {
        contractStorage[contractAddress][slot] = value;
    }

    // Nonces (this is used during contract creation to determine the contract address)
    function getNonce(address contractAddress) internal returns(bytes32) { /* TODO */ }
    function setNonce(address contractAddress, bytes32 value) internal { /* TODO */ }
    function incrementNonce(address contractAddress) internal { /* TODO */ }

    // Contract code storage / contract address retrieval
    function getCode(address contractAddress) internal returns(bytes memory) { /* TODO */ }
    function getCodeHash(address contractAddress) internal returns(bytes32) { /* TODO */ }
    function getCodeAddress(address contractAddress) internal returns(address) { /* TODO */ }
    function setCode(address contractAddress, bytes memory code) internal { /* TODO */ }
}
