pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";
import {FullStateManager} from "./FullStateManager.sol";

/**
 * @title ExecutionManager
 * @notice The execution manager ensures that the execution of each transaction is sandboxed in a distinct enviornment as defined
           by the supplied backend. Only state / contracts from that backend will be accessed.
 */
contract ExecutionManager is FullStateManager {
    dt.ExecutionContext executionContext;

    /**
     * @notice Construct a new ExecutionManager with a specified purity checker & owner.
     * @param _purityCheckerAddress The address for our purity checker, used during contract creation.
     * @param _owner The owner of our contract -- the only address allowed to make calls to our purity checker.
     */
    constructor(address _purityCheckerAddress, address _owner) public {
        // Initialize our execution manager with:
        // 1) the desired purity checker
        // 2) an owner who may call execute transaction
    }

    /**
     * @notice Execute a transition which consists of running a transaction within the context of a timestamp
     *         and queue origin.
     * @param _transaction The transaction which we will be executing against the state.
     * @param _timestamp The timestamp for the particular rollup block we are running.
     * @param _queueOrigin The queue which this transaction was sent from. Examples include the L1 contract queue, slow-track queue, and sequencer queue.
     * @return The updated storage elements. This will be used by the fraud prover to check the post state root.
     */
    function executeTransition(dt.Transaction calldata _transaction, uint _timestamp, uint _queueOrigin) external returns(dt.StorageElement[] memory) {
        // Pull out the contract which will be sent this tx's calldata.
        // With the contract's address, pull it's code address
        // Call/create the contract
    }

    /******************
    * Opcode wrappers *
    ******************/

    // Contract creation
    function ovmCREATE(address contractAddress, bytes memory ovm_bytecode) public { /* TODO */ }
    function ovmCREATE2(address contractAddress, bytes32 salt, bytes memory ovm_bytecode) public { /* TODO */ }

    // Contract calls
    function ovmCALL(address contractAddress, bytes memory ovm_calldata) public { /* TODO */ }
    function ovmSTATICCALL(address contractAddress, bytes memory ovm_calldata) public { /* TODO */ }
    function ovmDELEGATECALL(address contractAddress, bytes memory ovm_calldata) public { /* TODO */ }

    // Contract storage

    /**
     * @notice Load a value from storage. Note each contract has it's own storage.
     * @param _slot The slot (aka key) for the storage slot value that you are trying to load.
     * @return The value of that storage slot.
     */
    function ovmSLOAD(bytes32 _slot) public view returns(bytes32) {
        return getStorage(executionContext.ovmActiveContract, _slot);
    }

    /**
     * @notice Store a value. Note each contract has it's own storage.
     * @param _slot The slot (aka key) for the storage slot value that you are trying to store.
     * @param _value The desired value of the storage slot.
     */
    function ovmSSTORE(bytes32 _slot, bytes32 _value) public {
        setStorage(executionContext.ovmActiveContract, _slot, _value);
    }
}
