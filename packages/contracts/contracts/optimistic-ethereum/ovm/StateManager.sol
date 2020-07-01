pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title StateManager
 * @notice The StateManager is a simple abstraction which can be extended by
 *         either the Stateful client or the stateless client so that both can
 *         share the same Execution Manager.
 */
contract StateManager {
    // Storage
    function getStorage(address _ovmContractAddress, bytes32 _slot) external view returns(bytes32);
    function setStorage(address _ovmContractAddress, bytes32 _slot, bytes32 _value) external;

    // Nonces (this is used during contract creation to determine the contract address)
    function getOvmContractNonce(address _ovmContractAddress) external view returns(uint);
    function setOvmContractNonce(address _ovmContractAddress, uint _value) external;
    function incrementOvmContractNonce(address _ovmContractAddress) external;

    // Contract code storage / contract address retrieval
    function associateCodeContract(address _ovmContractAddress, address _codeContractAddress) public;
    function getCodeContractAddress(address _ovmContractAddress) external view returns(address);
    function getCodeContractBytecode(
        address _codeContractAddress
    ) public view returns (bytes memory codeContractBytecode);
    function getCodeContractHash(address _codeContractAddress) external view returns (bytes32 _codeContractHash);
}
