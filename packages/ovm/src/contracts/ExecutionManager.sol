pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";
import {FullStateManager} from "./FullStateManager.sol";
import {ContractAddressGenerator} from "./ContractAddressGenerator.sol";
import {CreatorContract} from "./CreatorContract.sol";
import {PurityChecker} from "./PurityChecker.sol";
import {RLPEncode} from "./RLPEncode.sol";

/**
 * @title ExecutionManager
 * @notice The execution manager ensures that the execution of each transaction is sandboxed in a distinct enviornment as defined
 *         by the supplied backend. Only state / contracts from that backend will be accessed.
 */
contract ExecutionManager is FullStateManager {
    // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
    bytes32 constant ovmCallMethodId = keccak256("ovmCALL()") >> 224;
    bytes32 constant ovmCreateMethodId = keccak256("ovmCREATE()") >> 224;
    bytes32 constant executeCallMethodId = keccak256("executeCall()") >> 224;

    // creator contract address
    address constant creatorContractAddress = 0x0000000000000000000000000000000000000000;
    address ZERO_ADDRESS = 0x0000000000000000000000000000000000000000;

    // Execution storage
    dt.ExecutionContext executionContext;
    // Add Contract Address Generation contract
    ContractAddressGenerator contractAddressGenerator;
    // Add Purity Checker contract
    PurityChecker purityChecker;
    RLPEncode rlp;
    // for testing: if true, then do not perform purity checking on init code or deployed bytecode
    bool overridePurityChecker;

    // Events
    event ActiveContract(address _activeContract);
    event CreatedContract(
        address _ovmContractAddress,
        address _codeContractAddress,
        bytes32 _codeContractHash
    );
    event CallingWithEOA();
    event EOACreatedContract(
        address _ovmContractAddress
    );
    event SetStorage(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value
    );

    /**
     * @notice Construct a new ExecutionManager with a specified purity checker & owner.
     * @param _opcodeWhitelistMask A bit mask representing which opcodes are whitelisted or not for our purity checker
     * @param _owner The owner of this contract.
     * @param _blockGasLimit The block gas limit for OVM blocks
     * @param _overridePurityChecker Set to true to disable purity checking (WARNING: Only do this in test environments)
     */
    constructor(uint256 _opcodeWhitelistMask, address _owner, uint _blockGasLimit, bool _overridePurityChecker) public {
        rlp = new RLPEncode();
        // Set override purity checker flag
        overridePurityChecker = _overridePurityChecker;
        // Set the purity checker address
        purityChecker = new PurityChecker(_opcodeWhitelistMask, address(this));
        // Initialize new contract address generator
        contractAddressGenerator = new ContractAddressGenerator();

        // Associate all precompiles
        for (uint160 i = 1; i < 20; i++) {
            associateCodeContract(address(i), address(i));
        }

        executionContext.gasLimit = _blockGasLimit;
        executionContext.chainId = 108;

        // Set our owner
        // TODO
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
     *       [ovmEntrypointAddress (address as bytes32 (left-padded, big-endian))]
     *       [callBytes (bytes (variable length))]
     * returndata: [variable-length bytes returned from call]
     */
    function executeCall() external {
        uint _timestamp;
        uint _queueOrigin;
        uint callSize;
        bytes memory callBytes;
        bytes32 methodId = ovmCallMethodId;
        assembly {
            // Revert if we don't have methodId, timestamp, queueOrigin, and ovmEntrypointAddress.
            if lt(calldatasize, 100) {
                revert(0,0)
            }

            // populate timestamp and queue origin from calldata
            _timestamp := calldataload(4)
            // skip method ID (bytes4) and timestamp (bytes32)
            _queueOrigin := calldataload(0x24)

            callBytes := mload(0x40)
            // set callsize: total param size minus 2 uints (methodId bytes are repurposed)
            callSize := sub(calldatasize, 0x40)
            mstore(0x40, add(callBytes, callSize))

            // leave room for method ID, skip ahead in calldata methodID(4), timestamp(32), queueOrigin(32)
            calldatacopy(add(callBytes, 4), 0x44, sub(callSize, 4))

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

    /********************
    * Execute EOA Calls *
    ********************/

    /**
     * @notice Execute an Externally Owned Account (EOA) call. This will accept all information required
     *         for an OVM transaction as well as a signature from an EOA. First we will calculate the
     *         sender address (EOA address) and then we will perform the call.
     * @param _timestamp The timestamp which should be used for this call's context.
     * @param _queueOrigin The parent-chain queue from which this call originated.
     * @param _nonce The current nonce of the EOA.
     * @param _ovmEntrypoint The contract which this transaction should be executed against.
     * @param _callBytes The calldata for this ovm transaction.
     * @param _v The v value of the ECDSA signature + CHAIN_ID.
     * @param _r The r value of the ECDSA signature.
     * @param _s The s value of the ECDSA signature.
     */
    function executeEOACall(
        uint _timestamp,
        uint _queueOrigin,
        uint _nonce,
        address _ovmEntrypoint,
        bytes memory _callBytes,
        uint8 _v,
        bytes32 _r,
        bytes32 _s
    ) public {
        // Get EOA address
        address eoaAddress = recoverEOAAddress(_nonce, _ovmEntrypoint, _callBytes, _v, _r, _s);
        // Require that the EOA signature isn't zero (invalid signature)
        require(eoaAddress != ZERO_ADDRESS, "Failed to recover signature");
        // Require nonce to be correct
        require(_nonce == getOvmContractNonce(eoaAddress), "Incorrect nonce!");
        // Make the EOA call for the account
        executeUnsignedEOACall(_timestamp, _queueOrigin, _ovmEntrypoint, _callBytes, eoaAddress);
    }

    /**
     * @notice Execute an unsigned EOA call. Note that unsigned EOA calls are unauthenticated.
     *         This means that they should not be allowed for normal execution.
     * @param _timestamp The timestamp which should be used for this call's context.
     * @param _queueOrigin The parent-chain queue from which this call originated.
     * @param _ovmEntrypoint The contract which this transaction should be executed against.
     * @param _callBytes The calldata for this ovm transaction.
     * @param _fromAddress The address which this call should originate from--the msg.sender.
     */
    function executeUnsignedEOACall(
        uint _timestamp,
        uint _queueOrigin,
        address _ovmEntrypoint,
        bytes memory _callBytes,
        address _fromAddress
    ) public {
        uint _nonce = getOvmContractNonce(_fromAddress);
        emit CallingWithEOA();
        // Initialize our context
        initializeContext(_timestamp, _queueOrigin);

        // Set the active contract to be our EOA address
        switchActiveContract(_fromAddress);

        // Set methodId based on whether we're creating a contract
        bytes32 methodId;
        uint256 callSize;
        bool isCreate = _ovmEntrypoint == ZERO_ADDRESS;
        // Check if we're creating -- ovmEntrypoint == ZERO_ADDRESS
        if (isCreate) {
          methodId = ovmCreateMethodId;
          callSize = _callBytes.length + 4;
          // Emit event that we are creating a contract with an EOA
          address _newOvmContractAddress = contractAddressGenerator.getAddressFromCREATE(_fromAddress, _nonce);
          emit EOACreatedContract(_newOvmContractAddress);
        } else {
          methodId = ovmCallMethodId;
          callSize = _callBytes.length + 32 + 4;
        }

        assembly {
          if eq(isCreate, 0) {
            _callBytes := sub(_callBytes, 4)
            mstore8(_callBytes, shr(24, methodId))
            mstore8(add(_callBytes, 1), shr(16, methodId))
            mstore8(add(_callBytes, 2), shr(8, methodId))
            mstore8(add(_callBytes, 3), methodId)
            // And now set the ovmEntrypoint
            mstore(add(_callBytes, 4), _ovmEntrypoint)
          }
          if eq(isCreate, 1) {
            _callBytes := add(_callBytes, 28)
            mstore8(_callBytes, shr(24, methodId))
            mstore8(add(_callBytes, 1), shr(16, methodId))
            mstore8(add(_callBytes, 2), shr(8, methodId))
            mstore8(add(_callBytes, 3), methodId)
          }
        }

        address addr = address(this);
        assembly {
            let result := mload(0x40)
            let success := call(gas, addr, 0, _callBytes, callSize, result, 500000)
            let size := returndatasize

            if eq(success, 0) {
                revert(0, 0)
            }

            return(result, size)
        }
    }

    /**
     * @notice Recover the EOA of an ECDSA-signed Ethereum transaction. Note some values will be set to zero by default.
     *         Additionally, the `to=ZERO_ADDRESS` is reserved for contract creation transactions.
     * @param _nonce The nonce of the transaction.
     * @param _to The entrypoint / recipient of the transaction.
     * @param _callData The calldata which will be applied to the entrypoint contract.
     * @param _v The v value of the ECDSA signature + CHAIN_ID.
     * @param _r The r value of the ECDSA signature.
     * @param _s The s value of the ECDSA signature.
     */
    function recoverEOAAddress(uint _nonce, address _to, bytes memory _callData, uint8 _v, bytes32 _r, bytes32 _s) public 
view returns (address) {
        bytes[] memory message = new bytes[](9);
        message[0] = rlp.encodeUint(_nonce); // Nonce
        message[1] = rlp.encodeUint(0); // Gas price
        message[2] = rlp.encodeUint(executionContext.gasLimit); // Gas limit
        // To -- Special rlp encoding handling if _to is the ZERO_ADDRESS
        if (_to == ZERO_ADDRESS) {
            message[3] = rlp.encodeUint(0);
        } else {
            message[3] = rlp.encodeAddress(_to);
        }
        message[4] = rlp.encodeUint(0); // Value
        message[5] = rlp.encodeBytes(_callData); // Data
        message[6] = rlp.encodeUint(executionContext.chainId); // ChainID
        message[7] = rlp.encodeUint(0); // Zeros for R
        message[8] = rlp.encodeUint(0); // Zeros for S

        bytes memory encodedMessage = rlp.encodeList(message);
        bytes32 hash = keccak256(abi.encodePacked(encodedMessage));
        /*
         * Replay protection is used to prevent signatures on one chain from
         * being used on other chains. To support replay protection ethereum
         * modifies the value of v in the signature to be different for each
         * chainID. This was implemented based on the following EIP:
         * https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md#specification
         */
        return ecrecover(hash, (_v - uint8(executionContext.chainId) * 2) - 8, _r, _s);
    }

    /**********************
    * OVM Context Opcodes *
    **********************/

    /**
     * @notice CALLER opcode (msg.sender) -- this gets the caller of the currently-running contract.
     * Note: Calling this requires a CALL, which changes the CALLER, which is why we use executionContext.
     *
     * This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: 4 bytes: [methodID (bytes4)]
     * returndata: 32-byte CALLER address containing the left-padded, big-endian encoding of the address.
     */
    function ovmCALLER() public view {
        // First make sure the ovmMsgSender was set
        require(executionContext.ovmMsgSender != ZERO_ADDRESS, "Error: attempting to access non-existent msgSender.");

        // This is returned as left-padded, big-endian, so pad it left!
        bytes32 addressBytes = bytes32(bytes20(executionContext.ovmMsgSender)) >> 96;

        assembly {
            let addressMemory := mload(0x40)
            mstore(addressMemory, addressBytes)
            return(addressMemory, 32)
        }
    }

    /**
     * @notice ADDRESS opcode -- Gets the address of the currently-running contract.
     * Note: Calling this requires a CALL, which changes the ADDRESS, which is why we use executionContext.
     *
     * This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: 4 bytes: [methodID (bytes4)]
     * returndata: 32-byte ADDRESS containing the left-padded, big-endian encoding of the address.
     */
    function ovmADDRESS() public view {
        // First make sure the ovmMsgSender was set
        require(executionContext.ovmActiveContract != ZERO_ADDRESS, "Error: attempting to access non-existent ovmActiveContract.");

        // This is returned as left-padded, big-endian, so pad it left!
        bytes32 addressBytes = bytes32(bytes20(executionContext.ovmActiveContract)) >> 96;

        assembly {
            let addressMemory := mload(0x40)
            mstore(addressMemory, addressBytes)
            return(addressMemory, 32)
        }
    }

    /**
     * @notice TIMESTAMP opcode -- this gets the current timestamp. Since the L2 value for this
     * will necessarily be different than L1, this needs to be overridden for the OVM.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: 4 bytes: [methodID (bytes4)]
     * returndata: uint256 representing the current timestamp.
     */
    function ovmTIMESTAMP() public view {
        // First make sure the timestamp was set
        require(executionContext.timestamp != 0, "Error: attempting to access non-existent timestamp.");

        uint t = executionContext.timestamp;

        assembly {
            let timestampMemory := mload(0x40)
            mstore(timestampMemory, t)
            return(timestampMemory, 32)
        }
    }

    /**
     * @notice GASLIMIT opcode -- this gets the gas limit for the current transaction. Since the L2 value for this
     * may be different than L1, this needs to be overridden for the OVM.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: 4 bytes: [methodID (bytes4)]
     * returndata: uint256 representing the current gas limit.
     */
    function ovmGASLIMIT() public view {
        uint g = executionContext.gasLimit;

        assembly {
            let gasLimitMemory := mload(0x40)
            mstore(gasLimitMemory, g)
            return(gasLimitMemory, 32)
        }
    }

    /**
     * @notice Gets the gas limit for fraud proofs. This value exists to make sure that fraud proofs
     * don't require an excessive amount of gas that is not feasible on L1.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: 4 bytes: [methodID (bytes4)]
     * returndata: uint256 representing the fraud proof gas limit.
     */
    function ovmBlockGasLimit() public view {
        uint g = executionContext.gasLimit;

        assembly {
            let gasLimitMemory := mload(0x40)
            mstore(gasLimitMemory, g)
            return(gasLimitMemory, 32)
        }
    }

    /**
     * @notice Gets the queue origin in the current Execution Context.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: 4 bytes: [methodID (bytes4)]
     * returndata: uint256 representing the current queue origin.
     */
    function ovmQueueOrigin() public view {
        uint q = executionContext.queueOrigin;

        assembly {
            let queueOriginMemory := mload(0x40)
            mstore(queueOriginMemory, q)
            return(queueOriginMemory, 32)
        }
    }

    /**
     * @notice This gets whether or not this contract is currently in a static call context.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: 4 bytes: [methodID (bytes4)]
     * returndata: uint256 of 1 if in a static context and 0 if not.
     */
    function isStaticContext() public view {
        uint staticContext = executionContext.inStaticContext ? 1 : 0;

        assembly {
            let contextMemory := mload(0x40)
            mstore(contextMemory, staticContext)
            return(contextMemory, 32)
        }
    }

    /********* Utils *********/

    /**
     * @notice Initialize a new context, setting the timestamp, queue origin, and gasLimit as well as zeroing out the
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
        // Emit an event so we can track the active contract. This is used in order to parse transaction receipts in the fullnode
        emit ActiveContract(_newActiveContract);
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
        require(!executionContext.inStaticContext, "Cannot create new contracts from a STATICCALL.");

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
        address _newOvmContractAddress = contractAddressGenerator.getAddressFromCREATE(creator, creatorNonce);
        // Next we need to actually create the contract in our state at that address
        createNewContract(_newOvmContractAddress, _ovmInitcode);
        // We also need to increment the contract nonce
        incrementOvmContractNonce(creator);

        // Shifting so that it is left-padded, big-endian ('00'x12 + 20 bytes of address)
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
        require(!executionContext.inStaticContext, "Cannot create new contracts from a STATICCALL.");

        bytes memory _ovmInitcode;
        bytes32 _salt;
        assembly {
            // everything other than MethodID and salt is initcode
            let initcodeSize := sub(calldatasize, 0x24)
            _ovmInitcode := mload(0x40)
            // skip methodID, copy first 32 bytes for _salt
            _salt := calldataload(4)
            // need to ABI-encode _ovmInitcode for solidity calls
            mstore(_ovmInitcode, initcodeSize)
            // read calldata, ignoring methodID and salt -- the rest is _ovmInitcode
            calldatacopy(add(_ovmInitcode, 0x20), 0x24, initcodeSize)

            mstore(0x40, add(0x20,add(_ovmInitcode, initcodeSize)))
        }

        // First we need to generate the CREATE2 address
        address creator = executionContext.ovmActiveContract;
        address _newOvmContractAddress = contractAddressGenerator.getAddressFromCREATE2(creator, _salt, _ovmInitcode);
        // Next we need to actually create the contract in our state at that address
        createNewContract(_newOvmContractAddress, _ovmInitcode);

        // Shifting so that it is left-padded, big-endian ('00'x12 + 20 bytes of address)
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
        // Purity check the initcode -- unless the overridePurityChecker flag is set to true
        require(overridePurityChecker || purityChecker.isBytecodePure(_ovmInitcode), "createNewContract: Contract init code is not pure.");
        // Switch the context to be the new contract
        (address oldMsgSender, address oldActiveContract) = switchActiveContract(_newOvmContractAddress);
        // Deploy the _ovmInitcode as a code contract -- Note the init script will run in the newly set context
        address codeContractAddress = deployCodeContract(_ovmInitcode);
        // Get the runtime bytecode
        bytes memory codeContractBytecode = getCodeContractBytecode(codeContractAddress);
        // Purity check the runtime bytecode -- unless the overridePurityChecker flag is set to true
        require(overridePurityChecker || purityChecker.isBytecodePure(codeContractBytecode), "createNewContract: Contract runtime bytecode is not pure.");
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
     *       [targetOvmContractAddress (address as bytes32 (left-padded, big-endian))]
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

            if eq(success, 0) {
                revert(0, 0)
            }

            returnSize := returndatasize
            mstore(0x40, add(returnData, returnSize))
        }

        // Revert back to our old execution context
        restoreContractContext(oldMsgSender, oldActiveContract);

        // Return the return value
        assembly {
            return(returnData, returnSize)
        }
    }

    /**
     * @notice STATICCALL opcode -- calls the code in question without allowing state modification.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: variable-length bytes:
     *       [methodID (bytes4)]
     *       [targetOvmContractAddress (address as bytes32 (big-endian))]
     *       [callBytes (bytes (variable length))]
     * returndata: [variable-length bytes returned from call]
     */
    function ovmSTATICCALL() public {
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

        bool wasStaticContext = executionContext.inStaticContext;
        executionContext.inStaticContext = true;

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
        // This covers the nested STATICCALL case
        executionContext.inStaticContext = wasStaticContext;

        // Return the return value
        assembly {
            return(returnData, returnSize)
        }
    }

    /**
     * @notice DELEGATECALL opcode -- calls the code in question without changing the OVM contract context.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: variable-length bytes:
     *       [methodID (bytes4)]
     *       [targetOvmContractAddress (address as bytes32 (big-endian))]
     *       [callBytes (bytes (variable length))]
     * returndata: [variable-length bytes returned from call]
     */
    function ovmDELEGATECALL() public {
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
        // NOTE: WE DO NOT SWITCH CONTEXTS HERE.
        address codeAddress = getCodeContractAddress(_targetOvmContractAddress);

        // make the call
        assembly {
            let returnData := mload(0x40)

            let success := call(
            gas,
            codeAddress,
            0,
            _callBytes,
            callSize,
            returnData,
            500000
            )

            if eq(success, 0) {
                revert(0, 0)
            }

            return(returnData, returndatasize)
        }
    }


    /***************************
    * Contract Storage Opcodes *
    ***************************/

    /**
     * @notice Load a value from storage. Note each contract has it's own storage.
     * Note: This is a raw function, so there are no listed (ABI-encoded) inputs / outputs.
     * Below format of the bytes expected as input and written as output:
     * calldata: 36 bytes:
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
     * calldata: 68 bytes:
     *       [methodID (bytes4)]
     *       [storageSlot (bytes32)]
     *       [storageValue (bytes32)]
     * returndata: empty.
     */
    function ovmSSTORE() public {
        require(!executionContext.inStaticContext, "Cannot call SSTORE from within a STATICCALL.");

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
     *      [targetOvmContractAddress (address as bytes32 (left-padded, big-endian))]
     * returndata: 32 bytes: the big-endian codesize int.
     */
    function ovmEXTCODESIZE() public view {
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
     *      [targetOvmContractAddress (address as bytes32 (left-padded, big-endian))]
     * returndata: 32 bytes: the hash.
     */
    function ovmEXTCODEHASH() public view {
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
     *       [targetOvmContractAddress (address as bytes32 (left-padded, big-endian))]
     *       [index (uint (32)]
     *       [length (uint (32))]
     * returndata: length (input param) bytes of contract at address, starting at index.
     */
    function ovmEXTCODECOPY() public view {
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
