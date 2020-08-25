pragma experimental ABIEncoderV2;

/**
 * @title IStateManager
 * @notice The State Manager interface which the Execution Manager uses.
 */
contract IStateManager {
    /*
     * Contract Variables
     */

    /*
     * Public Functions
     */

    /**********
    * Storage *
    **********/

    function getStorage(
        address _ovmContractAddress,
        bytes32 _slot
    ) public returns (bytes32);

    function getStorageView(
        address _ovmContractAddress,
        bytes32 _slot
    ) public view returns (bytes32);

    function setStorage(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value
    ) public;

    /**********
    * Accounts *
    **********/

    function getOvmContractNonce(
        address _ovmContractAddress
    ) public returns (uint);

    function getOvmContractNonceView(
        address _ovmContractAddress
    ) public view returns (uint);

    function setOvmContractNonce(
        address _ovmContractAddress,
        uint _value
    ) public;

    function incrementOvmContractNonce(
        address _ovmContractAddress
    ) public;
    
    /**********
    * Code *
    **********/

    function associateCodeContract(
        address _ovmContractAddress,
        address _codeContractAddress
    ) public;

    function registerCreatedContract(
        address _ovmContractAddress
    ) public;

    function getCodeContractAddressView(
        address _ovmContractAddress
    ) public view returns (address);

    function getCodeContractAddressFromOvmAddress(
        address _ovmContractAddress
    ) public returns(address);
    
    function getCodeContractBytecode(
        address _codeContractAddress
    ) public view returns (bytes memory codeContractBytecode);

    function getCodeContractHash(
        address _codeContractAddress
    ) public view returns (bytes32 _codeContractHash);

}