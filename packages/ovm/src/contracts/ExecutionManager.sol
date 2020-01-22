pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";
import {FullStateManager} from "./FullStateManager.sol";
import {ContractAddressGenerator} from "./ContractAddressGenerator.sol";
import {CreatorContract} from "./CreatorContract.sol";
import {PurityChecker} from "./PurityChecker.sol";

/**
 * @title ExecutionManager
 * @notice The execution manager ensures that the execution of each transaction is sandboxed in a distinct enviornment as defined
 *         by the supplied backend. Only state / contracts from that backend will be accessed.
 */
contract ExecutionManager is FullStateManager {
    address ZERO_ADDRESS = 0x0000000000000000000000000000000000000000;

    // Execution storage
    dt.ExecutionContext executionContext;
    // Add Contract Address Generation library
    ContractAddressGenerator cag;
    // Add Purity Checker library
    PurityChecker purityChecker;

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
    constructor(address _purityCheckerAddress, address _contractAddressGeneration, address _owner) public {
        // Set the purity checker address
        purityChecker = PurityChecker(_purityCheckerAddress);
        // Set the contract address generation address
        cag = ContractAddressGenerator(_contractAddressGeneration);
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
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: variable-length bytes:
     *       [methodID (bytes4)]
     *       [timestamp (uint)]
     *       [queueOrigin (uint)]
     *       [ovmEntrypointAddress (address as bytes32 (big-endian))]
     *       [callBytes (bytes (variable length))]
     * returndata: [variable-length bytes returned from call] - The updated storage elements.
     *      This will be used by the fraud prover to check the post state root.
     */
    function executeTransaction() external {
        address addr = address(this);
        bytes4 methodId = bytes4(keccak256("executeCall()") >> 224);

        bytes memory execCallBytes;
        assembly {
            execCallBytes := mload(0x40)
            calldatacopy(execCallBytes, 0, calldatasize)

            mstore8(execCallBytes, shr(24, methodId))
            mstore8(add(execCallBytes, 1), shr(16, methodId))
            mstore8(add(execCallBytes, 2), shr(8, methodId))
            mstore8(add(execCallBytes, 3), methodId)

            // overwrite call's data
            let result := mload(0x40)
            let success := call(gas, addr, 0, execCallBytes, calldatasize, result, 500000)

            if eq(success,0) {
                revert(0,0)
            }
        }
        // TODO: Track & return storage elements
    }

    /**
     * @notice Execute a call which will return the result of the call instead of the updated storage.
     *         Note: This should only be used with a Web3 `call` operation, otherwise you may accidentally save changes to the state.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: variable-length bytes:
     *       [methodID (bytes4)]
     *       [timestamp (uint)]
     *       [queueOrigin (uint)]
     *       [ovmEntrypointAddress (address as bytes32 (big-endian))]
     *       [callBytes (bytes (variable length))]
     * returndata: [variable-length bytes returned from call]
     */
    function executeCall() external {
        bytes4 methodId = bytes4(keccak256("ovmCALL()") >> 224);

        uint _timestamp;
        uint _queueOrigin;
        uint callSize;
        bytes memory callBytes;

        assembly {
            // populate timestamp and queue origin from calldata
            _timestamp := calldataload(4)
            // skip method ID (bytes4) and timestamp (bytes32)
            _queueOrigin := calldataload(0x24)

            // set callsize: total param size minus 2 uints plus 4 byte method ID - 4 bytes (new method ID)
            callSize := sub(calldatasize, 0x40)
            callBytes := mload(0x40)
            mstore(0x40, add(callBytes, callSize))

            // leave room for method ID, skip ahead in calldata methodID(4), timestamp(32), queueOrigin(32)
            calldatacopy(add(callBytes, 4), 0x44, callSize)
            mstore8(callBytes, shr(24, methodId))
            mstore8(add(callBytes, 1), shr(16, methodId))
            mstore8(add(callBytes, 2), shr(8, methodId))
            mstore8(add(callBytes, 3), methodId)
        }

        // Initialize our context
        initializeContext(_timestamp, _queueOrigin);

        address addr = address(this);
        assembly {
            let result := mload(0x40)
            let success := call(gas, addr, 0, callBytes, callSize, result, 500000)
            let size := returndatasize

            if eq(success, 0) {
                revert(0, 0)
            }

            return(result, size)
        }
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
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: variable-length bytes:
     *       [methodID (bytes4)]
     *       [ovmInitcode (bytes (variable length))]
     * returndata: [newOvmContractAddress (as bytes32)]
     */
    function ovmCREATE() public {
        bytes memory _ovmInitcode;
        assembly {
            _ovmInitcode := mload(0x40)
            // ignore methodID
            let initcodeSize := sub(calldatasize, 4)
            // need to ABI-encode _ovmInitcode for solidity calls
            mstore(_ovmInitcode, initcodeSize)
            // read calldata, ignoring methodID -- the rest is _ovmInitcode
            calldatacopy(add(_ovmInitcode, 0x20), 4, initcodeSize)
            // update free mem pointer
            mstore(0x40, add(add(_ovmInitcode, 0x20), initcodeSize))
        }

        // First we need to generate the CREATE address
        address creator = executionContext.ovmActiveContract;
        uint creatorNonce = getOvmContractNonce(creator);
        address _newOvmContractAddress = cag.getAddressFromCREATE(creator, creatorNonce);
        // Next we need to actually create the contract in our state at that address
        createNewContract(_newOvmContractAddress, _ovmInitcode);
        // We also need to increment the contract nonce
        incrementOvmContractNonce(creator);

        // Shifting so that it is big-endian ('00'x12 + 20 bytes of address)
        bytes32 newOvmContractAddressBytes32 = bytes32(bytes20(_newOvmContractAddress)) >> 96;

        // And finally return the address of the newly created ovmContract
        assembly {
            let returnData := mload(0x40)
            mstore(returnData, newOvmContractAddressBytes32)
            return(returnData, 0x20)
        }
    }

    /**
     * @notice CREATE2 opcode -- deploying a new ovm contract to a CREATE2 address.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: variable-length bytes:
     *       [methodID (bytes4)]
     *       [salt (bytes32)]
     *       [ovmInitcode (bytes (variable length))]
     * returndata: [newOvmContractAddress (as bytes32)]
     */
    function ovmCREATE2() public {
        bytes memory _ovmInitcode;
        bytes32 _salt;
        assembly {
            _ovmInitcode := mload(0x40)
            // skip methodID, copy first 32 bytes for _salt
            _salt := calldataload(4)

            // Copy initcode
            let initcodeSize := sub(calldatasize, 0x24)
            calldatacopy(_ovmInitcode, 0x24, initcodeSize)

            mstore(0x40, add(_ovmInitcode, initcodeSize))
        }

        // First we need to generate the CREATE2 address
        address creator = executionContext.ovmActiveContract;
        address _newOvmContractAddress = cag.getAddressFromCREATE2(creator, _salt, _ovmInitcode);
        // Next we need to actually create the contract in our state at that address
        createNewContract(_newOvmContractAddress, _ovmInitcode);

        // Shifting so that it is big-endian ('00'x12 + 20 bytes of address)
        bytes32 newOvmContractAddressBytes32 = bytes32(bytes20(_newOvmContractAddress)) >> 96;

        // And finally return the address of the newly created ovmContract
        assembly {
            let returnData := mload(0x40)
            mstore(returnData, newOvmContractAddressBytes32)
            return(returnData, 0x20)
        }
    }

    /********* Utils *********/

    /**
     * @notice Create a new contract at some OVM contract address.
     * @param _newOvmContractAddress The desired OVM contract address for this new contract we will deploy.
     * @param _ovmInitcode The initcode for our new contract
     */
    function createNewContract(address _newOvmContractAddress, bytes memory _ovmInitcode) internal {
        // Purity check the initcode
        require(purityChecker.isBytecodePure(_ovmInitcode), "createNewContract: Contract init code is not pure.");
        // Switch the context to be the new contract
        (address oldMsgSender, address oldActiveContract) = switchActiveContract(_newOvmContractAddress);
        // Deploy the _ovmInitcode as a code contract -- Note the init script will run in the newly set context
        address codeContractAddress = deployCodeContract(_ovmInitcode);
        // Get the runtime bytecode
        bytes memory codeContractBytecode = getCodeContractBytecode(codeContractAddress);
        // Purity check the runtime bytecode
        require(purityChecker.isBytecodePure(codeContractBytecode), "createNewContract: Contract runtime bytecode is not pure.");
        // Associate the code contract with our ovm contract
        associateCodeContract(_newOvmContractAddress, codeContractAddress);
        // Get the code contract address to be emitted by a CreatedContract event
        bytes32 codeContractHash = keccak256(codeContractBytecode);
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
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: variable-length bytes:
     *       [methodID (bytes4)]
     *       [targetOvmContractAddress (address as bytes32 (big-endian))]
     *       [callBytes (bytes (variable length))]
     * returndata: [variable-length bytes returned from call]
     */
    function ovmCALL() public {
        uint callSize;
        bytes memory _callBytes;
        bytes32 _targetOvmContractAddressBytes;
        // parse calldata
        assembly {
            // skip 4 bytes for methodID and first 12 bytes of address
            _targetOvmContractAddressBytes := calldataload(16)

            // size is calldata - methodID - address (as bytes32)
            callSize := sub(calldatasize, 0x24)

            // set callBytes
            _callBytes := mload(0x40)
            mstore(0x40, add(_callBytes, callSize))
            calldatacopy(_callBytes, 0x24, callSize)
        }
        address _targetOvmContractAddress = address(bytes20(_targetOvmContractAddressBytes));

        // switch the context to the _targetOvmContractAddress
        (address oldMsgSender, address oldActiveContract) = switchActiveContract(_targetOvmContractAddress);
        address codeAddress = getCodeContractAddress(_targetOvmContractAddress);

        bytes memory returnData;
        uint returnSize;
        // make the call
        assembly {
            returnData := mload(0x40)

            let success := call(
              gas,
              codeAddress,
              0,
              _callBytes,
              callSize,
              returnData,
              500000
            )

            returnSize := returndatasize

            if eq(success, 0) {
                revert(0, 0)
            }
        }

        // Revert back to our old execution context
        restoreContractContext(oldMsgSender, oldActiveContract);

        // Return the return value
        assembly {
            return(returnData, returnSize)
        }
    }

    function ovmSTATICCALL(address _targetOvmContractAddress, bytes memory _ovmCalldata) public { /* TODO */ }
    function ovmDELEGATECALL(address _targetOvmContractAddress, bytes memory _ovmCalldata) public { /* TODO */ }


    /***************************
    * Contract Storage Opcodes *
    ***************************/

    /**
     * @notice Load a value from storage. Note each contract has it's own storage.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: variable-length bytes:
     *       [methodID (bytes4)]
     *       [storageSlot (bytes32)]
     * returndata: [storageValue (bytes32)]
     */
    function ovmSLOAD() public view {
        bytes32 _storageSlot;
        assembly {
            // skip methodID (4 bytes)
            _storageSlot := calldataload(4)
        }

        bytes32 slotValue = getStorage(executionContext.ovmActiveContract, _storageSlot);

        assembly {
            let ret := mload(0x40)
            mstore(ret, slotValue)
            return(ret, 0x20)
        }
    }

    /**
     * @notice Store a value. Note each contract has it's own storage.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: variable-length bytes:
     *       [methodID (bytes4)]
     *       [storageSlot (bytes32)]
     *       [storageValue (bytes32)]
     * returndata: empty.
     */
    function ovmSSTORE() public {
        bytes32 _storageSlot;
        bytes32 _storageValue;

        assembly {
            // skip methodID (4 bytes)
            _storageSlot := calldataload(4)
            _storageValue := calldataload(0x24)
        }

        setStorage(executionContext.ovmActiveContract, _storageSlot, _storageValue);
        // Emit SetStorage event!
        emit SetStorage(executionContext.ovmActiveContract, _storageSlot, _storageValue);
    }

    /***********************
    * Code-related Opcodes *
    ************************/

    /**
     * @notice Executes the extcodesize operation for the contract address provided.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: 36 bytes:
     *      [methodID (bytes4)]
     *      [targetOvmContractAddress (address as bytes32 (big-endian))]
     * returndata: 32 bytes: the big-endian codesize int.
     */
    function ovmEXTCODESIZE() public {
        bytes32 _targetAddressBytes;
        assembly {
        // read calldata, ignoring methodID and first 12 bytes of address
            _targetAddressBytes := calldataload(16)
        }

        address _targetOvmContractAddress = address(bytes20(_targetAddressBytes));
        address codeContractAddress = getCodeContractAddress(_targetOvmContractAddress);

        assembly {
            let sizeBytes := mload(0x40)
            mstore(sizeBytes, extcodesize(codeContractAddress))
            return(sizeBytes, 32)
        }
    }

    /**
     * @notice Executes the extcodehash operation for the contract address provided.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: 36 bytes:
     *      [methodID (bytes4)]
     *      [targetOvmContractAddress (address as bytes32 (big-endian))]
     * returndata: 32 bytes: the hash.
     */
    function ovmEXTCODEHASH() public {
        bytes32 _targetAddressBytes;
        assembly {
            // read calldata, ignoring methodID and first 12 bytes of address
            _targetAddressBytes := calldataload(16)
        }

        address _targetOvmContractAddress = address(bytes20(_targetAddressBytes));
        address codeContractAddress = getCodeContractAddress(_targetOvmContractAddress);

        bytes32 hash = getCodeContractHash(codeContractAddress);

        assembly {
            let hashBytes := mload(0x40)
            mstore(hashBytes, hash)
            return(hashBytes, 32)
        }
    }

    /**
     * @notice Executes the extcodecopy operation for the contract address, index, and length provided.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: 100 bytes:
     *       [methodID (bytes4)]
     *       [targetOvmContractAddress (address as bytes32 (big-endian))]
     *       [index (uint (32)]
     *       [length (uint (32))]
     * returndata: length (input param) bytes of contract at address, starting at index.
     */
    function ovmEXTCODECOPY() public {
        bytes32 _targetAddressBytes;
        uint _index;
        uint _length;
        assembly {
            // read calldata, ignoring methodID
            _targetAddressBytes := calldataload(16)
            // skip 4 + 32
            _index := calldataload(0x24)
            // skip 4 + 32 + 32
            _length := calldataload(0x44)
        }

        address _targetOvmContractAddress = address(bytes20(_targetAddressBytes));
        address codeContractAddress = getCodeContractAddress(_targetOvmContractAddress);

        assembly {
            let codeContractBytecode := mload(0x40)
            // store code in memory
            extcodecopy(codeContractAddress, codeContractBytecode, _index, _length)
            // write code to returndata
            return(codeContractBytecode, _length)
        }
    }
}
