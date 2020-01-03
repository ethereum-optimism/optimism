pragma solidity ^0.5.0;

/**
 * @title StateManager
 * @notice The StateManager is a simple abstraction which can be extended by either the Stateful client
 *         or the stateless client so that both can share the same Execution Manager.
 */
contract StateManager {
    // Storage
    function getStorage(address _ovmContractAddress, bytes32 _slot) internal view returns(bytes32);
    function setStorage(address _ovmContractAddress, bytes32 _slot, bytes32 _value) internal;

    // Nonces (this is used during contract creation to determine the contract address)
    function getOvmContractNonce(address _ovmContractAddress) internal returns(uint);
    function setOvmContractNonce(address _ovmContractAddress, uint _value) internal;
    function incrementOvmContractNonce(address _ovmContractAddress) internal;

    // Contract code storage / contract address retrieval
    function associateCodeContract(address _ovmContractAddress, address _codeContractAddress) internal;
    function getCodeContractAddress(address _ovmContractAddress) internal returns(address);
    function getCodeContractBytecode(address _codeContractAddress) internal view returns (bytes memory codeContractBytecode);
    function getCodeContractHash(address _codeContractAddress) internal view returns (bytes32 _codeContractHash);
}
