pragma experimental ABIEncoderV2;

/* Contract Imports */
import { IStateManager } from "./interfaces/IStateManager.sol";
import { StateTransitioner } from "./StateTransitioner.sol";
import { ExecutionManager } from "./ExecutionManager.sol";

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
contract StateManagerGasProxy is IStateManager {
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

    uint private externalStateManagerGasConsumed;

    uint private virtualStateManagerGasConsumed;

    /*
     * Modifiers
     */


    /*
     * Constructor
     */

    /**
     * @param _addressResolver Address of the AddressResolver contract.
     * @param _stateTransitioner Address of the StateTransitioner attached to this contract.
     */
    constructor(
        address _addressResolver,
        address _stateTransitioner
    )
        public
        ContractResolver(_addressResolver)
    {
        stateTransitioner = StateTransitioner(_stateTransitioner);
    }

    /*
     * Gas Virtualization Logic
     */

    function recordExternalGasConsumed(uint _externalGasConsumed) internal {
        externalStateManagerGasConsumed += _externalGasConsumed;
    }

    function recordVirtualGasConsumed(uint _virtualGasConsumed) interal {
        virtualStateManagerGasConsumed += _virtualGasConsumed;
    }

    function forwardCallAndRecordExternalConsumption() internal {
        uint initialGas = gasleft();
        address stateManager = resolveStateManager();
        assembly {
            initialFreeMemStart := mload(0x40)
            callSize := calldatasize()
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
            if eq(success, 0) {
                revert(0,0) // surface revert up to the EM
            }
        }
        // increment the external gas by the amount consumed
        recordExternalGasConsumed(
            initialGas - gasleft()
        );
    }

    function returnProxiedReturnData() internal {
        assembly {
            freememory := mload(0x40)
            returnSize := returndatasize()
            returndatacopy(
                freemeory,
                0,
                returnSize
            )
            return(freememory, returnSize)
        }
    }

    function executeProxyRecordingVirtualizedGas(
        uint _virtualGasToConsume
    ) {
        forwardCallAndRecordExternalConsumption();
        recordVirtualGasConsumed(_virtualGasToConsume);
        returnProxiedReturnData();
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
    ) public view returns (bytes32) {
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
    ) public view returns (uint) {
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
    ) public view returns (address) {
        executeProxyRecordingVirtualizedGas(GET_CODE_CONTRACT_ADDRESS_VIRTUAL_GAS_COST);
    }

    function getCodeContractAddressFromOvmAddress(
        address _ovmContractAddress
    ) public returns(address) {
        executeProxyRecordingVirtualizedGas(GET_CODE_CONTRACT_ADDRESS_VIRTUAL_GAS_COST);
    }
    
    function getCodeContractBytecode(
        address _codeContractAddress
    ) public view returns (bytes memory codeContractBytecode) {
        forwardCallAndRecordExternalConsumption();

        uint returnedCodeSize;
        assembly {
            returnedCodeSize := returndatasize()
        }
        recordVirtualGasConsumed(
            returnedCodeSize * GET_CODE_CONTRACT_BYTECODE_VIRUAL_GAS_COST_PER_BYTE
        );

        returnProxiedReturnData();
    }

    function getCodeContractHash(
        address _codeContractAddress
    ) public view returns (bytes32 _codeContractHash) {
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