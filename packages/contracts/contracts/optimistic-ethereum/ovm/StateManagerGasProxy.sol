pragma solidity ^0.5.0;

/* Contract Imports */
import { IStateManager } from "./interfaces/IStateManager.sol";
import { StateTransitioner } from "./StateTransitioner.sol";
import { ExecutionManager } from "./ExecutionManager.sol";
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";

/* Testing Imports */
import { console } from "@nomiclabs/buidler/console.sol";

/**
 * @title StateManagerGasProxy
 * @notice The StateManagerGasProxy is used to virtualize all calls to the state manager.
 *         It serves as a proxy between an EM and SM implementation, recording all consumed SM gas ("external gas consumed"),
 *         as well as a "virtual gas" which should be charged on L2.  The EM will subtract the external gas, and add the virtual gas, at the end of execution.
 *
 *         This allows for OVM gas metering to be independent of the actual consumption of the SM, so that different SM implementations use the same gas.
 */

 // TODO: cannot inerit IStateManager here due to visibility changes. How to resolve?
contract StateManagerGasProxy is ContractResolver {
    /*
     * Virtual (i.e. Charged by OVM) Gas Cost Constants
     */

    // Storage
    uint constant GET_STORAGE_VIRTUAL_GAS_COST = 10000;
    uint constant SET_STORAGE_VIRTUAL_GAS_COST = 30000;
    // Nonces
    uint constant GET_CONTRACT_NONCE_VIRTUAL_GAS_COST = 10000;
    uint constant SET_CONTRACT_NONCE_VIRTUAL_GAS_COST = 30000;
    uint constant INCREMENT_CONTRACT_NONCE_VIRTUAL_GAS_COST = 35000;
    // Code
    uint constant ASSOCIATE_CODE_CONTRACT_VIRTUAL_GAS_COST = 1000;
    uint constant REGISTER_CREATED_CONTRACT_VIRTUAL_GAS_COST = 1000;
    uint constant GET_CODE_CONTRACT_ADDRESS_VIRTUAL_GAS_COST = 1000;
    uint constant GET_CODE_CONTRACT_HASH_VIRTUAL_GAS_COST = 1000;
    // Code copy retrieval, linear in code size
    uint constant GET_CODE_CONTRACT_BYTECODE_VIRUAL_GAS_COST_PER_BYTE = 10;

    /*
     * Contract Variables
     */

    uint externalStateManagerGasConsumed;

    uint virtualStateManagerGasConsumed;

    /*
     * Modifiers
     */


    /*
     * Constructor
     */

    /**
     * @param _addressResolver Address of the AddressResolver contract.
     */
    constructor(
        address _addressResolver
    )
        public
        ContractResolver(_addressResolver)
    {}

    /*
     * Gas Virtualization and Storage
     */

    // External Initialization and Retrieval Logic
    function inializeGasConsumedValues() external {
        externalStateManagerGasConsumed = 0;
        virtualStateManagerGasConsumed = 0;
    }

    function getStateManagerExternalGasConsumed() external returns(uint) {
        return externalStateManagerGasConsumed;
    }

    function getStateManagerVirtualGasConsumed() external returns(uint) {
        return virtualStateManagerGasConsumed;
    }

    // Internal Logic

    function recordExternalGasConsumed(uint _externalGasConsumed) internal {
        externalStateManagerGasConsumed += _externalGasConsumed;
        return;
    }

    function recordVirtualGasConsumed(uint _virtualGasConsumed) internal {
        virtualStateManagerGasConsumed += _virtualGasConsumed;
        return;
    }

    /**
     * Forwards a call to this proxy along to the actual state manager, and records the consumned external gas.
     * Reverts if the forwarded call reverts, but currently does not forward revert message, as an SM should never revert.
     */
    function proxyCallAndRecordExternalConsumption() internal {
        uint initialGas = gasleft();
        address stateManager = resolveStateManager();
        bool success;
        uint returnedSize;
        uint returnDataStart;
        assembly {
            let initialFreeMemStart := mload(0x40)
            let callSize := calldatasize()
            mstore(0x40, add(initialFreeMemStart, callSize))
            calldatacopy(
                initialFreeMemStart,
                0,
                callSize
            )
            success := call(
                gas(), // all remaining gas, leaving enough for this to execute
                stateManager,
                0,
                initialFreeMemStart,
                callSize,
                0, // we will RETURNDATACOPY the return data later, no need to use now
                0
            )
            returnedSize := returndatasize()
            if eq(success, 0) {
                revert(0,0) // surface revert up to the EM
            }

            // write the returndata to memory
            returnDataStart := mload(0x40)
            mstore(0x40, add(returnDataStart, returnedSize))
            returndatacopy(
                returnDataStart,
                0,
                returnedSize
            )
        }

        // #if FLAG_IS_DEBUG
        console.log("In call forwarder. success is", success, ", returnedSize is", returnedSize);
        // #endif

        // increment the external gas by the amount consumed
        recordExternalGasConsumed(
            initialGas - gasleft()
        );

        // #if FLAG_IS_DEBUG
        console.log("recorded external gas consumed");
        // #endif

        assembly {
            return(returnDataStart, returnedSize)
        }
    }

//     /**
//     * Returns the result of a forwarded SM call to back to the execution manager.
//     * Uses RETURNDATACOPY, so that virtualization logic can be implemented in between here and the forwarded call.
//     */
//     function returnProxiedReturnData() internal {
//                 uint returnedDataSize;
// assembly {
//                 returnedDataSize := returndatasize()

// }

//         // #if FLAG_IS_DEBUG
//         console.log("returning data size of", returnedDataSize);
//         // #endif

//         assembly {
//             let freememory := mload(0x40)
//             let returnSize := returndatasize()
//             returndatacopy(
//                 freememory,
//                 0,
//                 returnSize
//             )
//             return(freememory, returnSize)
//         }
//     }

    function executeProxyRecordingVirtualizedGas(
        uint _virtualGasToConsume
    ) internal {
        recordVirtualGasConsumed(_virtualGasToConsume);
        proxyCallAndRecordExternalConsumption();
    }
    
    /*
     * Public Functions
     */

    /**********
    * Storage *
    **********/

    function getStorage(
        address _ovmContractAddress,
        bytes32 _slot
    ) public returns (bytes32) {
        executeProxyRecordingVirtualizedGas(GET_STORAGE_VIRTUAL_GAS_COST);
    }

    function getStorageView(
        address _ovmContractAddress,
        bytes32 _slot
    ) public returns (bytes32) {
        // #if FLAG_IS_DEBUG
        console.log("in getStorageView");
        // #endif
        executeProxyRecordingVirtualizedGas(GET_STORAGE_VIRTUAL_GAS_COST);
    }

    function setStorage(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value
    ) public {
        executeProxyRecordingVirtualizedGas(SET_STORAGE_VIRTUAL_GAS_COST);
    }

    /**********
    * Accounts *
    **********/

    function getOvmContractNonce(
        address _ovmContractAddress
    ) public returns (uint) {
        executeProxyRecordingVirtualizedGas(GET_CONTRACT_NONCE_VIRTUAL_GAS_COST);
    }

    function getOvmContractNonceView(
        address _ovmContractAddress
    ) public returns (uint) {
        executeProxyRecordingVirtualizedGas(GET_CONTRACT_NONCE_VIRTUAL_GAS_COST);
    }

    function setOvmContractNonce(
        address _ovmContractAddress,
        uint _value
    ) public {
        executeProxyRecordingVirtualizedGas(SET_CONTRACT_NONCE_VIRTUAL_GAS_COST);
    }

    function incrementOvmContractNonce(
        address _ovmContractAddress
    ) public {
        executeProxyRecordingVirtualizedGas(INCREMENT_CONTRACT_NONCE_VIRTUAL_GAS_COST);
    }
    
    /**********
    * Code *
    **********/

    function associateCodeContract(
        address _ovmContractAddress,
        address _codeContractAddress
    ) public {
        executeProxyRecordingVirtualizedGas(ASSOCIATE_CODE_CONTRACT_VIRTUAL_GAS_COST);
    }

    function registerCreatedContract(
        address _ovmContractAddress
    ) public {
        executeProxyRecordingVirtualizedGas(REGISTER_CREATED_CONTRACT_VIRTUAL_GAS_COST);
    }

    function getCodeContractAddressView(
        address _ovmContractAddress
    ) public returns (address) {
        executeProxyRecordingVirtualizedGas(GET_CODE_CONTRACT_ADDRESS_VIRTUAL_GAS_COST);
    }

    function getCodeContractAddressFromOvmAddress(
        address _ovmContractAddress
    ) public returns(address) {
        executeProxyRecordingVirtualizedGas(GET_CODE_CONTRACT_ADDRESS_VIRTUAL_GAS_COST);
    }
    
    function getCodeContractBytecode(
        address _codeContractAddress
    ) public returns (bytes memory codeContractBytecode) {
        // TODO: make this a multiplier
        executeProxyRecordingVirtualizedGas(GET_CODE_CONTRACT_BYTECODE_VIRUAL_GAS_COST_PER_BYTE);
    }

    function getCodeContractHash(
        address _codeContractAddress
    ) public returns (bytes32 _codeContractHash) {
        executeProxyRecordingVirtualizedGas(GET_CODE_CONTRACT_HASH_VIRTUAL_GAS_COST);
    }

    /*
     * Contract Resolution
     */

    function resolveStateManager()
        internal
        view returns (address)
    {
        return resolveContract("StateManager");
    }
}