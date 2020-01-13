pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";
import {FullStateManager} from "./FullStateManager.sol";
import {ContractAddressGenerator} from "./ContractAddressGenerator.sol";
import {CreatorContract} from "./CreatorContract.sol";

/**
 * @title ExecutionManager
 * @notice The execution manager ensures that the execution of each transaction is sandboxed in a distinct enviornment as defined
 *         by the supplied backend. Only state / contracts from that backend will be accessed.
 */
contract ExecutionManager is FullStateManager, ContractAddressGenerator {
    address ZERO_ADDRESS = 0x0000000000000000000000000000000000000000;

    // Execution storage
    dt.ExecutionContext executionContext;

    // Events
    event CreatedContract(
        address _ovmContractAddress,
        address _codeContractAddress,
        bytes32 _codeContractHash
    );
    event SetStorage(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value
    );

    /**
     * @notice Construct a new ExecutionManager with a specified purity checker & owner.
     * @param _purityCheckerAddress The address for our purity checker, used during contract creation.
     * @param _owner The owner of our contract -- the only address allowed to make calls to our purity checker.
     */
    constructor(address _purityCheckerAddress, address _owner) public {
        // Set the purity checker address
        // TODO

        // Deploy our genesis code contract (the normal way)
        CreatorContract creatorContract = new CreatorContract(address(this));
        // Set our genesis creator contract to be the zero address
        address genesisAddress = ZERO_ADDRESS;
        associateCodeContract(genesisAddress, address(creatorContract));

        // Set our owner
        // TODO
    }

    /**
     * @notice Execute a transaction which consists of running a transaction within the context of a timestamp
     *         and queue origin.
     * @param _transaction The transaction which we will be executing against the state.
     * @param _timestamp The timestamp for the particular rollup block we are running.
     * @param _queueOrigin The queue which this transaction was sent from. Examples include the L1 contract queue, slow-track queue, and sequencer queue.
     * @return The updated storage elements. This will be used by the fraud prover to check the post state root.
     */
    function executeTransaction(dt.Transaction calldata _transaction, uint _timestamp, uint _queueOrigin) external returns(dt.StorageElement[] memory) {
        // Initialize our context
        initializeContext(_timestamp, _queueOrigin);
        // And then make the entrypoint CALL!
        (bool success,) = ovmCALL(_transaction.ovmEntrypoint, _transaction.ovmCalldata);
        require(success);
        // TODO: Track & return storage elements
    }

    /**
     * @notice Execute a call which will return the result of the call instead of the updated storage.
     *         Note: This should only be used with a Web3 `call` operation, otherwise you may accidentally save changes to the state.
     * @param _transaction The transaction which we will be executing against the state.
     * @param _timestamp The timestamp for the particular rollup block we are running.
     * @param _queueOrigin The queue which this transaction was sent from. Examples include the L1 contract queue, slow-track queue, and sequencer queue.
     * @return Result of the call as bytes
     */
    function executeCall(dt.Transaction calldata _transaction, uint _timestamp, uint _queueOrigin) external returns(bytes memory) {
        // Initialize our context
        initializeContext(_timestamp, _queueOrigin);
        // And then make the entrypoint CALL!
        (bool success, bytes memory _callResult) = ovmCALL(_transaction.ovmEntrypoint, _transaction.ovmCalldata);
        require(success);
        return _callResult;
    }

    /**********************
    * OVM Context Opcodes *
    **********************/

    function ovmMsgSender() public view returns(address) {
        // First make sure the ovmMsgSender was set
        require(executionContext.ovmMsgSender != ZERO_ADDRESS, "Error attempting to access non-existent msgSender.");
        // If not, simply return the msgSender
        return executionContext.ovmMsgSender;
    }

    // TODO: Add more context getters like timestamp & queueOrigin.

    /********* Utils *********/

    /**
     * @notice Initialize a new context, setting the timestamp and queue origin as well as zeroing out the
     *         msgSender of the previous context.
     *         NOTE: this zeroing may not technically be needed as the context should always end up as zero at the end of each execution.
     * @param _timestamp The timestamp which should be used for this context.
     * @param _queueOrigin The queue which this context's transaction was sent from.
     */
    function initializeContext(uint _timestamp, uint _queueOrigin) internal {
        // First zero out the context for good measure (Note ZERO_ADDRESS is reserved for the genesis contract & initial msgSender)
        restoreContractContext(ZERO_ADDRESS, ZERO_ADDRESS);
        // And finally set the timestamp & queue origin
        executionContext.timestamp = _timestamp;
        executionContext.queueOrigin = _queueOrigin;
    }

    /**
     * @notice Change the active contract to be something new. This is used when a new contract is called.
     * @param _newActiveContract The new active contract
     * @return The old msgSender and activeContract. This will be used when we restore the old active contract.
     */
    function switchActiveContract(address _newActiveContract) internal returns(address _oldMsgSender, address _oldActiveContract) {
        // Store references to the old context
        _oldActiveContract = executionContext.ovmActiveContract;
        _oldMsgSender = executionContext.ovmMsgSender;
        // Set our new context
        executionContext.ovmActiveContract = _newActiveContract;
        executionContext.ovmMsgSender = _oldActiveContract;
        // Return old context so we can later revert to it
        return (_oldMsgSender, _oldActiveContract);
    }

    /**
     * @notice Restore the contract context to some old values.
     * @param _msgSender The msgSender to be restored.
     * @param _activeContract The activeContract to be restored.
     */
    function restoreContractContext(address _msgSender, address _activeContract) internal {
        // Revert back to the old context
        executionContext.ovmActiveContract = _activeContract;
        executionContext.ovmMsgSender = _msgSender;
    }


    /***************************
    * Contract Creation Opcode *
    ***************************/

    /**
     * @notice CREATE opcode -- deploying a new ovm contract to a CREATE address.
     * @param _ovmInitcode The initcode for our new contract.
     * @return The newly deployed ovm contract address.
     */
    function ovmCREATE(bytes memory _ovmInitcode) public returns(address _newOvmContractAddress) {
        // First we need to generate the CREATE address
        address creator = executionContext.ovmActiveContract;
        uint creatorNonce = getOvmContractNonce(creator);
        _newOvmContractAddress = getAddressFromCREATE(creator, creatorNonce);
        // Next we need to actually create the contract in our state at that address
        createNewContract(_newOvmContractAddress, _ovmInitcode);
        // We also need to increment the contract nonce
        incrementOvmContractNonce(creator);
        // And finally return the address of the newly created ovmContract
        return _newOvmContractAddress;
    }

    /**
     * @notice CREATE2 opcode -- deploying a new ovm contract to a CREATE2 address.
     * @param _salt The CREATE2 salt, used for address generation.
     * @param _ovmInitcode The initcode for our new CREATE2 contract
     * @return The newly deployed ovm contract address.
     */
    function ovmCREATE2(bytes32 _salt, bytes memory _ovmInitcode) public returns(address _newOvmContractAddress) {
        // First we need to generate the CREATE2 address
        address creator = executionContext.ovmActiveContract;
        _newOvmContractAddress = getAddressFromCREATE2(creator, _salt, _ovmInitcode);
        // Next we need to actually create the contract in our state at that address
        createNewContract(_newOvmContractAddress, _ovmInitcode);
        // And finally return the address of the newly created ovmContract
        return _newOvmContractAddress;
    }

    /********* Utils *********/

    /**
     * @notice Create a new contract at some OVM contract address.
     * @param _newOvmContractAddress The desired OVM contract address for this new contract we will deploy.
     * @param _ovmInitcode The initcode for our new contract
     */
    function createNewContract(address _newOvmContractAddress, bytes memory _ovmInitcode) internal {
        // Purity check the initcode
        // TODO
        // Switch the context to be the new contract
        (address oldMsgSender, address oldActiveContract) = switchActiveContract(_newOvmContractAddress);
        // Deploy the _ovmInitcode as a code contract -- Note the init script will run in the newly set context
        address codeContractAddress = deployCodeContract(_ovmInitcode);
        // Associate the code contract with our ovm contract
        associateCodeContract(_newOvmContractAddress, codeContractAddress);
        // Get the code contract address to be emitted by a CreatedContract event
        bytes32 codeContractHash = getCodeContractHash(codeContractAddress);
        // Revert to the previous the context
        restoreContractContext(oldMsgSender, oldActiveContract);
        // Emit CreatedContract event! We've created a new contract!
        emit CreatedContract(_newOvmContractAddress, codeContractAddress, codeContractHash);
    }

    /**
     * @notice Deploys a code contract, and then registers it to the state
     * @param _ovmContractInitcode The bytecode of the contract to be deployed
     * @return the codeContractAddress.
     */
    function deployCodeContract(bytes memory _ovmContractInitcode) internal returns(address codeContractAddress) {
        // Deploy a new contract with this _ovmContractInitCode
        assembly {
            // Set our codeContractAddress to the address returned by our CREATE operation
            codeContractAddress := create(0, add(_ovmContractInitcode, 0x20), mload(_ovmContractInitcode))
            // Make sure that the CREATE was successful (actually deployed something)
            if iszero(extcodesize(codeContractAddress)) {
                revert(0, 0)
            }
        }
        return codeContractAddress;
    }


    /************************
    * Contract CALL Opcodes *
    ************************/

    /**
     * @notice CALL opcode -- simply calls a particular code contract with the desired OVM contract context.
     * @param _targetOvmContractAddress The OVM contract address the we are calling.
     * @param _ovmCalldata The calldata which will be used to call this OVM contract.
     * @return True/False if the contract succeeded or failed, and bytes being the return value.
     */
    function ovmCALL(address _targetOvmContractAddress, bytes memory _ovmCalldata) public returns(bool, bytes memory) {
        // Switch the context to the _targetOvmContractAddress
        (address oldMsgSender, address oldActiveContract) = switchActiveContract(_targetOvmContractAddress);
        // Call the contract
        (bool success, bytes memory returnValue) = getCodeContractAddress(_targetOvmContractAddress).call(_ovmCalldata);
        // Revert back to our old execution context
        restoreContractContext(oldMsgSender, oldActiveContract);
        // Return success and the return value
        return (success, returnValue);
    }
    function ovmSTATICCALL(address _targetOvmContractAddress, bytes memory _ovmCalldata) public { /* TODO */ }
    function ovmDELEGATECALL(address _targetOvmContractAddress, bytes memory _ovmCalldata) public { /* TODO */ }


    /***************************
    * Contract Storage Opcodes *
    ***************************/

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
        // Emit SetStorage event!
        emit SetStorage(executionContext.ovmActiveContract, _slot, _value);
    }

    /***********************
    * Code-related Opcodes *
    ************************/

    function ovmEXTCODESIZE(address _targetOvmContractAddress) public returns (uint) {
        address codeContractAddress = getCodeContractAddress(_targetOvmContractAddress);

        assembly {
            let returnDataMemoryIndex := mload(0x40)
            mstore(0x40, add(returnDataMemoryIndex, 32))
            mstore(returnDataMemoryIndex, extcodesize(codeContractAddress))
            return(returnDataMemoryIndex, 32)
        }
    }

    function ovmEXTCODEHASH(address _targetOvmContractAddress) public returns (bytes32) {
        address codeContractAddress = getCodeContractAddress(_targetOvmContractAddress);

        return getCodeContractHash(codeContractAddress);
    }

    function ovmCODECOPY(address _targetOvmContractAddress, uint _index, uint _length) public returns(bytes memory codeContractBytecode) {
        address codeContractAddress = getCodeContractAddress(_targetOvmContractAddress);
        assembly {
        // allocate output byte array
            codeContractBytecode := mload(0x40)
        // new "memory end" including padding
            mstore(0x40, add(codeContractBytecode, and(add(add(_length, 0x20), 0x1f), not(0x1f))))
        // store length in memory
            mstore(codeContractBytecode, _length)
        // actually retrieve the code, this needs assembly
            extcodecopy(codeContractAddress, add(codeContractBytecode, 0x20), _index, _length)
        }
    }
}
