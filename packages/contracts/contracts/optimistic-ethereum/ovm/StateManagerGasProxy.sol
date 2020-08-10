pragma solidity ^0.5.0;

/* Contract Imports */
import { IStateManager } from "./interfaces/IStateManager.sol";
import { StateTransitioner } from "./StateTransitioner.sol";
import { ExecutionManager } from "./ExecutionManager.sol";
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { GasConsumer } from "../utils/libraries/GasConsumer.sol";

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
 // TODO: rename.  Gas sanitizer?
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
    uint constant GET_CODE_CONTRACT_BYTECODE_VIRTUAL_GAS_COST_PER_BYTE = 10;

    /*
     * Constant/Upper-bounded Fixed Gas Cost Constants
     */

// todo parameterize

    // Storage
    uint constant GET_STORAGE_GAS_COST_UPPER_BOUND = 200000;
    uint constant SET_STORAGE_GAS_COST_UPPER_BOUND = 200000;
    // Nonces
    uint constant GET_CONTRACT_NONCE_GAS_COST_UPPER_BOUND = 200000;
    uint constant SET_CONTRACT_NONCE_GAS_COST_UPPER_BOUND = 200000;
    uint constant INCREMENT_CONTRACT_NONCE_GAS_COST_UPPER_BOUND = 200000;
    // Code
    uint constant ASSOCIATE_CODE_CONTRACT_GAS_COST_UPPER_BOUND = 200000;
    uint constant REGISTER_CREATED_CONTRACT_GAS_COST_UPPER_BOUND = 200000;
    uint constant GET_CODE_CONTRACT_ADDRESS_GAS_COST_UPPER_BOUND = 200000;
    uint constant GET_CODE_CONTRACT_HASH_GAS_COST_UPPER_BOUND = 200000;
    // Code copy retrieval, linear in code size
    uint constant GET_CODE_CONTRACT_BYTECODE_GAS_COST_UPPER_BOUND = 200000;


    /*
     * Contract Variables
     */

    uint OVMRefund;

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
    function resetOVMRefund() external {
        OVMRefund = 0;
    }

    function getOVMRefund() external returns(uint) {
        return OVMRefund;
    }

    // Internal Logic

    function addToOVMRefund(uint _refund) internal {
        OVMRefund += _refund;
        return;
    }

    /** TODO UPDATE THIS DOCSTR
     * Forwards a call to this proxy along to the actual state manager, and records the consumned external gas.
     * Reverts if the forwarded call reverts, but currently does not forward revert message, as an SM should never revert.
     */
    function performSanitizedProxyAndRecordRefund(
        uint _sanitizedGasCost,
        uint _virtualGasCost
    ) internal {
        uint initialGas = gasleft();

        uint refund = _sanitizedGasCost - _virtualGasCost;
        addToOVMRefund(refund);

        address stateManager = resolveStateManager();
        // #if FLAG_IS_DEBUG
        console.log("calling SM at", stateManager);
        // #endif

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

            // write the returndata to memory
            returnDataStart := mload(0x40)
            mstore(0x40, add(returnDataStart, returnedSize))
            returndatacopy(
                returnDataStart,
                0,
                returnedSize
            )
        }

        // todo safemath negatives
        GasConsumer gasConsumer = GasConsumer(resolveGasConsumer());
        uint gasAlreadyConsumed = initialGas - gasleft();
        uint gasLeftToConsume = _sanitizedGasCost - gasAlreadyConsumed;
        // #if FLAG_IS_DEBUG
        console.log("calling CG at", address(gasConsumer), "with amount of gas left to consume", gasLeftToConsume);
        console.log( "success is", success, "returned size is", returnedSize);
        // #endif
        gasConsumer.consumeGasInternalCall(gasLeftToConsume);

        assembly {
            if eq(success, 0) {
                revert(0,0) // surface revert up to the EM
            }
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

    // function executeProxyRecordingVirtualizedGas(
    //     uint _virtualGasToConsume
    // ) internal {
    //     recordVirtualGasConsumed(_virtualGasToConsume);
    //     proxyCallAndRecordExternalConsumption();
    // }
    
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
        performSanitizedProxyAndRecordRefund(
            GET_STORAGE_GAS_COST_UPPER_BOUND,
            GET_STORAGE_VIRTUAL_GAS_COST
        );
    }

    function getStorageView(
        address _ovmContractAddress,
        bytes32 _slot
    ) public returns (bytes32) {
        // #if FLAG_IS_DEBUG
        console.log("in getStorageView");
        // #endif
        performSanitizedProxyAndRecordRefund(
            SET_STORAGE_GAS_COST_UPPER_BOUND,
            GET_STORAGE_VIRTUAL_GAS_COST
        );
    }

    function setStorage(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value
    ) public {
        performSanitizedProxyAndRecordRefund(
            SET_STORAGE_GAS_COST_UPPER_BOUND,
            SET_STORAGE_VIRTUAL_GAS_COST
        );
    }

    /**********
    * Accounts *
    **********/

    function getOvmContractNonce(
        address _ovmContractAddress
    ) public returns (uint) {
        performSanitizedProxyAndRecordRefund(
            GET_CONTRACT_NONCE_GAS_COST_UPPER_BOUND,
            GET_CONTRACT_NONCE_VIRTUAL_GAS_COST
        );
    }

    function getOvmContractNonceView(
        address _ovmContractAddress
    ) public returns (uint) {
        performSanitizedProxyAndRecordRefund(
            GET_CONTRACT_NONCE_GAS_COST_UPPER_BOUND,
            GET_CONTRACT_NONCE_VIRTUAL_GAS_COST
        );
    }

    function setOvmContractNonce(
        address _ovmContractAddress,
        uint _value
    ) public {
        performSanitizedProxyAndRecordRefund(
            SET_CONTRACT_NONCE_GAS_COST_UPPER_BOUND,
            SET_CONTRACT_NONCE_VIRTUAL_GAS_COST
        );
    }

    function incrementOvmContractNonce(
        address _ovmContractAddress
    ) public {
        performSanitizedProxyAndRecordRefund(
            INCREMENT_CONTRACT_NONCE_GAS_COST_UPPER_BOUND,
            INCREMENT_CONTRACT_NONCE_VIRTUAL_GAS_COST
        );
    }
    
    /**********
    * Code *
    **********/

    function associateCodeContract(
        address _ovmContractAddress,
        address _codeContractAddress
    ) public {
        performSanitizedProxyAndRecordRefund(
            ASSOCIATE_CODE_CONTRACT_GAS_COST_UPPER_BOUND,
            ASSOCIATE_CODE_CONTRACT_VIRTUAL_GAS_COST
        );
    }

    function registerCreatedContract(
        address _ovmContractAddress
    ) public {
        performSanitizedProxyAndRecordRefund(
            REGISTER_CREATED_CONTRACT_GAS_COST_UPPER_BOUND,
            REGISTER_CREATED_CONTRACT_VIRTUAL_GAS_COST
        );
    }

    function getCodeContractAddressView(
        address _ovmContractAddress
    ) public returns (address) {
        performSanitizedProxyAndRecordRefund(
            GET_CODE_CONTRACT_ADDRESS_GAS_COST_UPPER_BOUND,
            GET_CODE_CONTRACT_ADDRESS_VIRTUAL_GAS_COST
        );
    }

    function getCodeContractAddressFromOvmAddress(
        address _ovmContractAddress
    ) public returns(address) {
        performSanitizedProxyAndRecordRefund(
            GET_CODE_CONTRACT_ADDRESS_GAS_COST_UPPER_BOUND,
            GET_CODE_CONTRACT_ADDRESS_VIRTUAL_GAS_COST);
    }
    
    function getCodeContractBytecode(
        address _codeContractAddress
    ) public returns (bytes memory codeContractBytecode) {
        // TODO: make this a multiplier
        performSanitizedProxyAndRecordRefund(
            GET_CODE_CONTRACT_BYTECODE_GAS_COST_UPPER_BOUND,
            GET_CODE_CONTRACT_BYTECODE_VIRTUAL_GAS_COST_PER_BYTE
        );
    }

    function getCodeContractHash(
        address _codeContractAddress
    ) public returns (bytes32 _codeContractHash) {
        performSanitizedProxyAndRecordRefund(
            GET_CODE_CONTRACT_HASH_GAS_COST_UPPER_BOUND,
            GET_CODE_CONTRACT_HASH_VIRTUAL_GAS_COST
        );
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

    function resolveGasConsumer()
        internal
        view returns(address)
    {
        return resolveContract("GasConsumer");
    }
}