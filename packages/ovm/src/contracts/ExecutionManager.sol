pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";

/**
 * @title ExecutionManager
 * @notice The execution manager ensures that the execution of each transaction is sandboxed in a distinct enviornment as defined
           by the supplied backend. Only state / contracts from that backend will be accessed.
 */
contract ExecutionManager {
    constructor(address _stateManager, address _purityChecker, address _owner) public {
        // Initialize our execution manager with:
        // 1) the desired state manager
        // 2) the desired purity checker
        // 3) an owner who may call execute transaction
    }

    function executeTransaction(dt.Transaction calldata transaction) external returns(dt.StorageElement[] memory) {
        // Pull out the contract which will be sent this tx's calldata.
        // Optional: Decompress the contract address using our cache.
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
    function ovmSLOAD(address contractAddress, bytes32 slot) public { /* TODO */ }
    function ovmSSTORE(address contractAddress, bytes32 slot, bytes32 value) public { /* TODO */ }
}
