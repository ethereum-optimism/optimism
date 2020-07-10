pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import { StateManager } from "./StateManager.sol";
import { SafetyChecker } from "./SafetyChecker.sol";

/**
 * @title FullStateManager
 * @notice The FullStateManager is used for off-chain tx evaluation. It holds a complete mapping
 *         of all chain storage.
 */
contract FullStateManager is StateManager {
    /*
     * Contract Variables
     */

    address ZERO_ADDRESS = 0x0000000000000000000000000000000000000000;

    mapping(address=>mapping(bytes32=>bytes32)) ovmContractStorage;
    mapping(address=>uint) ovmContractNonces;
    mapping(address=>address) ovmAddressToCodeContractAddress;
    mapping(address=>address) codeContractAddressToOvmAddress;


    /*
     * Public Functions
     */


    /***********
     * Storage *
     ***********/

    /**
     * @notice Get storage for OVM contract at some slot.
     * @param _ovmContractAddress The contract we're getting storage of.
     * @param _slot The slot we're querying.
     * @return The bytes32 value stored at the particular slot.
     */
    function getStorage(
        address _ovmContractAddress,
        bytes32 _slot
    ) public returns (bytes32) {
        return ovmContractStorage[_ovmContractAddress][_slot];
    }

    /**
     * @notice Set storage for OVM contract at some slot.
     * @param _ovmContractAddress The contract we're setting storage of.
     * @param _slot The slot we're setting.
     * @param _value The value we will set the storage to.
     */
    function setStorage(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value
    ) public {
        ovmContractStorage[_ovmContractAddress][_slot] = _value;
    }


    /**********
     * Nonces *
     **********/

    /**
     * @notice Get the nonce for a particular OVM contract
     * @param _ovmContractAddress The contract we're getting the nonce of.
     * @return The contract nonce used for contract creation.
     */
    function getOvmContractNonce(
        address _ovmContractAddress
    ) public returns (uint) {
        return ovmContractNonces[_ovmContractAddress];
    }

    /**
     * @notice Set the nonce for a particular OVM contract
     * @param _ovmContractAddress The contract we're setting the nonce of.
     * @param _value The new nonce.
     */
    function setOvmContractNonce(
        address _ovmContractAddress,
        uint _value
    ) public {
        ovmContractNonces[_ovmContractAddress] = _value;
    }

    /**
     * @notice Increment the nonce for a particular OVM contract.
     * @param _ovmContractAddress The contract we're incrementing by 1 the nonce of.
     */
    function incrementOvmContractNonce(
        address _ovmContractAddress
    ) public {
        ovmContractNonces[_ovmContractAddress] += 1;
    }


    /******************
     * Contract Codes *
     ******************/

    /**
     * @notice Attaches some code contract to the desired OVM contract. This allows the Execution Manager
     *         to later on get the code contract address to perform calls for this OVM contract.
     * @param _ovmContractAddress The address of the OVM contract we'd like to associate with some code.
     * @param _codeContractAddress The address of the code contract that's been deployed.
     */
    function associateCodeContract(
        address _ovmContractAddress,
        address _codeContractAddress
    ) public {
        ovmAddressToCodeContractAddress[_ovmContractAddress] = _codeContractAddress;
        codeContractAddressToOvmAddress[_codeContractAddress] = _ovmContractAddress;
    }

    /**
     * @notice Lookup the code contract for some OVM contract, allowing CALL opcodes to be performed.
     * @param _ovmContractAddress The address of the OVM contract.
     * @return The associated code contract address.
     */
    function getCodeContractAddressFromOvmAddress(
        address _ovmContractAddress
    ) public view returns (address) {
        return ovmAddressToCodeContractAddress[_ovmContractAddress];
    }

    /**
     * @notice Lookup the OVM contract for some code contract
     * @param _codeContractAddress The address of the code contract.
     * @return The associated OVM contract address.
     */
    function getOvmAddressFromCodeContractAddress(
        address _codeContractAddress
    ) public view returns (address) {
        return codeContractAddressToOvmAddress[_codeContractAddress];
    }

    /**
     * @notice Get the bytecode at some contract address. NOTE: This is code taken from Solidity docs here:
     *         https://solidity.readthedocs.io/en/v0.5.0/assembly.html#example
     * @param _codeContractAddress The address of the code contract.
     * @return The bytecode at this address.
     */
    function getCodeContractBytecode(
        address _codeContractAddress
    ) public view returns (bytes memory codeContractBytecode) {
        assembly {
            // retrieve the size of the code
            let size := extcodesize(_codeContractAddress)
            // allocate output byte array - this could also be done without assembly
            // by using codeContractBytecode = new bytes(size)
            codeContractBytecode := mload(0x40)
            // new "memory end" including padding
            mstore(0x40, add(codeContractBytecode, and(add(add(size, 0x20), 0x1f), not(0x1f))))
            // store length in memory
            mstore(codeContractBytecode, size)
            // actually retrieve the code, this needs assembly
            extcodecopy(_codeContractAddress, add(codeContractBytecode, 0x20), 0, size)
        }
    }

    /**
     * @notice Get the hash of the deployed bytecode of some code contract.
     * @param _codeContractAddress The address of the code contract.
     * @return The hash of the bytecode at this address.
     */
    function getCodeContractHash(
        address _codeContractAddress
    ) public view returns (bytes32 _codeContractHash) {
        // TODO: Look up cached hash values eventually to avoid having to load all of this bytecode
        bytes memory codeContractBytecode = getCodeContractBytecode(_codeContractAddress);
        _codeContractHash = keccak256(codeContractBytecode);
        return _codeContractHash;
    }
}
