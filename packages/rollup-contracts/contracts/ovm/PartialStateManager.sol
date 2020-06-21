pragma experimental ABIEncoderV2;

/* Internal Imports */
import {StateManager} from "../StateManager.sol";
import {SafetyChecker} from "../SafetyChecker.sol";
import {FraudVerifier} from "./FraudVerifier.sol";
import {ExecutionManager} from "../ExecutionManager.sol";

/**
 * @title PartialStateManager
 * @notice The PartialStateManager is used for the on-chain fraud proof checker.
 *         It is supplied with only the state which is used to execute a single transaction. This
 *         is unlike the FullStateManager which has access to every storage slot.
 */
contract PartialStateManager {
    address constant ZERO_ADDRESS = 0x0000000000000000000000000000000000000000;

    SafetyChecker safetyChecker;
    FraudVerifier fraudVerifier;
    ExecutionManager executionManager;


    mapping(address=>mapping(bytes32=>bytes32)) ovmContractStorage;
    mapping(address=>uint) ovmContractNonces;
    mapping(address=>address) ovmCodeContracts;

    bool public existsInvalidStateAccess;
    bool public isTransactionSuccessfullyExecuted;

    mapping(address=>mapping(bytes32=>bool)) isVerifiedStorage;
    mapping(address=>bool) isVerifiedContract;
    mapping(uint=>bytes32) updatedStorage;
    uint updatedStorageCounter;
    mapping(uint=>address) updatedContracts;
    uint updatedContractsCounter;
    bytes32 stateRoot;

    modifier onlyFraudVerifier {
        require(msg.sender == address(fraudVerifier));
        _;
    }

    modifier onlyExecutionManager {
        require(msg.sender == address(executionManager));
        _;
    }

    modifier preExecution {
        require(!isTransactionSuccessfullyExecuted, "Function only callable before tx execution.");
        _;
    }

    modifier postExecution {
        require(isTransactionSuccessfullyExecuted, "Function only callable after tx execution.");
        _;
    }

    /**
     * @notice Construct a new FullStateManager with a specified safety checker.
     */
    constructor(bytes32 _stateRoot, address _safetyCheckerAddress, address _fraudVerifierAddress) public {
        stateRoot = _stateRoot;
        safetyChecker = SafetyChecker(_safetyCheckerAddress);
        fraudVerifier = FraudVerifier(_fraudVerifierAddress);
    }

    /**
     * @notice This is a seperate function because it allows us to first deploy the state manager,
     * then deploy the execution manager (passing in the state manager address), and then set the execution manager
     * address in the state manager. This is a bit ugly & probably should be thought through a bit more.
     */
    function setExecutionManager(address _executionManagerAddress) onlyFraudVerifier public {
        executionManager = ExecutionManager(_executionManagerAddress);
    }

    /**
     * @notice Initialize a new transaction execution
     */
    function initNewTransactionExecution() onlyFraudVerifier external {
        existsInvalidStateAccess = false;
        updatedStorageCounter = 0;
        updatedContractsCounter = 0;
    }

    function setIsTransitionSuccessfullyExecuted(bool _isTransactionSuccessfullyExecuted) onlyFraudVerifier external {
        isTransactionSuccessfullyExecuted = _isTransactionSuccessfullyExecuted;
    }

    function ensureVerifiedStorage(address _ovmContractAddress, bytes32 _slot) private {
        if (!isVerifiedStorage[_ovmContractAddress][_slot]) {
            existsInvalidStateAccess = true;
        }
    }

    function ensureVerifiedContract(address _ovmContractAddress) private {
        if (!isVerifiedContract[_ovmContractAddress]) {
            existsInvalidStateAccess = true;
        }
    }

    /****************
    * Pre-Execution *
    ****************/

    function verifyStorage(address _ovmContractAddress, bytes32 _slot, bytes32 _value) external preExecution {
        // TODO: Verify inclusion proof against root

        isVerifiedStorage[_ovmContractAddress][_slot] = true;
        ovmContractStorage[_ovmContractAddress][_slot] = _value;
    }

    function verifyContract(address _ovmContractAddress, bytes32 _codeHash, uint _nonce) external preExecution {
        // TODO: Verify inclusion proof against root
        // TODO: Get the actual code contract addr from the registry
        address _codeContractAddress = 0x0000000000000000000000000000000000000000;

        isVerifiedContract[_ovmContractAddress] = true;
        ovmContractNonces[_ovmContractAddress] = _nonce;
        associateCodeContract(_ovmContractAddress, _codeContractAddress);
    }

    /*****************
    * Post-Execution *
    *****************/

    // TODO:
    function updateStorageNode(address _ovmContractAddress, bytes32 _slot, bytes32 _value) external postExecution {
        // TODO: Finish filling out this function...
        // Get the next storage we need to update using the updatedStorageCounter
        address contractAddressToUpdate = address(bytes20(updatedStorage[updatedStorageCounter - 1]));
        bytes32 storageKeyToUpdate = updatedStorage[updatedStorageCounter];
        bytes32 storageValueToUpdate = getStorage(contractAddressToUpdate, storageKeyToUpdate);

        // Update the storage / store the new root

        // Then decrement the updatedStorageCounter (we go reverse through the updated storage).
        // Note that when this hits zero we'll have updated all storage slots that were changed during
        // transaction execution.
        updatedStorageCounter -= 2;
    }
    // function updateContractNode(...) external postExecution {}

    /**********
    * Storage *
    **********/

    /**
     * @notice Get storage for OVM contract at some slot.
     * @param _ovmContractAddress The contract we're getting storage of.
     * @param _slot The slot we're querying.
     * @return The bytes32 value stored at the particular slot.
     */
    function getStorage(address _ovmContractAddress, bytes32 _slot) onlyExecutionManager public returns(bytes32) {
        ensureVerifiedStorage(_ovmContractAddress, _slot);

        return ovmContractStorage[_ovmContractAddress][_slot];
    }

    /**
     * @notice Set storage for OVM contract at some slot.
     * @param _ovmContractAddress The contract we're setting storage of.
     * @param _slot The slot we're setting.
     * @param _value The value we will set the storage to.
     */
    function setStorage(address _ovmContractAddress, bytes32 _slot, bytes32 _value) onlyExecutionManager public {
        ensureVerifiedStorage(_ovmContractAddress, _slot);

        // Add this storage slot to the list of updated storage
        updatedStorage[updatedStorageCounter] = bytes32(bytes20(_ovmContractAddress));
        updatedStorage[updatedStorageCounter+1] = _slot;
        updatedStorageCounter += 2;

        // Set the new storage value
        ovmContractStorage[_ovmContractAddress][_slot] = _value;
    }


    /*********
    * Nonces *
    *********/

    /**
     * @notice Get the nonce for a particular OVM contract
     * @param _ovmContractAddress The contract we're getting the nonce of.
     * @return The contract nonce used for contract creation.
     */
    function getOvmContractNonce(address _ovmContractAddress) onlyExecutionManager public returns(uint) {
        ensureVerifiedContract(_ovmContractAddress);

        return ovmContractNonces[_ovmContractAddress];
    }

    /**
     * @notice Set the nonce for a particular OVM contract
     * @param _ovmContractAddress The contract we're setting the nonce of.
     * @param _value The new nonce.
     */
    function setOvmContractNonce(address _ovmContractAddress, uint _value) onlyExecutionManager public {
        ensureVerifiedContract(_ovmContractAddress);

        // Add this contract to the list of updated contracts
        updatedContracts[updatedContractsCounter] = _ovmContractAddress;
        updatedContractsCounter += 1;

        // Return the nonce
        ovmContractNonces[_ovmContractAddress] = _value;
    }

    /**
     * @notice Increment the nonce for a particular OVM contract.
     * @param _ovmContractAddress The contract we're incrementing by 1 the nonce of.
     */
    function incrementOvmContractNonce(address _ovmContractAddress) onlyExecutionManager public {
        ensureVerifiedContract(_ovmContractAddress);

        // Add this contract to the list of updated contracts
        updatedContracts[updatedContractsCounter] = _ovmContractAddress;
        updatedContractsCounter += 1;

        // Increment the nonce
        ovmContractNonces[_ovmContractAddress] += 1;
    }


    /*****************
    * Contract Codes *
    *****************/
    // This is used when CALLing a contract

    /**
     * @notice Attaches some code contract to the desired OVM contract. This allows the Execution Manager
     *         to later on get the code contract address to perform calls for this OVM contract.
     * @param _ovmContractAddress The address of the OVM contract we'd like to associate with some code.
     * @param _codeContractAddress The address of the code contract that's been deployed.
     */
    function associateCodeContract(address _ovmContractAddress, address _codeContractAddress) onlyExecutionManager public {
        ovmCodeContracts[_ovmContractAddress] = _codeContractAddress;
    }

    /**
     * @notice Lookup the code contract for some OVM contract, allowing CALL opcodes to be performed.
     * @param _ovmContractAddress The address of the OVM contract.
     * @return The associated code contract address.
     */
    function getCodeContractAddress(address _ovmContractAddress) onlyExecutionManager public returns(address) {
        ensureVerifiedContract(_ovmContractAddress);

        return ovmCodeContracts[_ovmContractAddress];
    }

    /**
     * @notice Get the bytecode at some contract address. NOTE: This is code taken from Solidity docs here:
     *         https://solidity.readthedocs.io/en/v0.5.0/assembly.html#example
     * @param _codeContractAddress The address of the code contract.
     * @return The bytecode at this address.
     */
    function getCodeContractBytecode(address _codeContractAddress) public view returns (bytes memory codeContractBytecode) {
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
    function getCodeContractHash(address _codeContractAddress) public view returns (bytes32 _codeContractHash) {
        // TODO: Look up cached hash values eventually to avoid having to load all of this bytecode
        bytes memory codeContractBytecode = getCodeContractBytecode(_codeContractAddress);
        _codeContractHash = keccak256(codeContractBytecode);
        return _codeContractHash;
    }

    /**
     * @notice Deploys a code contract, and then registers it to the state
     * @param _ovmContractInitcode The bytecode of the contract to be deployed
     * @return the codeContractAddress.
     */
    function deployContract(
        address _newOvmContractAddress,
        bytes memory _ovmContractInitcode
    ) onlyExecutionManager public returns(address codeContractAddress) {
        ensureVerifiedContract(_newOvmContractAddress);

        // Safety check the initcode
        if (!safetyChecker.isBytecodeSafe(_ovmContractInitcode)) {
            // Contract initcode is not safe.
            return ZERO_ADDRESS;
        }

        // Deploy a new contract with this _ovmContractInitCode
        assembly {
            // Set our codeContractAddress to the address returned by our CREATE operation
            codeContractAddress := create(0, add(_ovmContractInitcode, 0x20), mload(_ovmContractInitcode))
            // Make sure that the CREATE was successful (actually deployed something)
            if iszero(extcodesize(codeContractAddress)) {
                revert(0, 0)
            }
        }

        // Safety check the runtime bytecode
        bytes memory codeContractBytecode = getCodeContractBytecode(codeContractAddress);
        if (!safetyChecker.isBytecodeSafe(codeContractBytecode)) {
            // Contract runtime bytecode is not safe.
            return ZERO_ADDRESS;
        }

        // Associate the code contract with the ovm contract
        associateCodeContract(_newOvmContractAddress, codeContractAddress);

        // Add this contract to the list of updated contracts
        updatedContracts[updatedContractsCounter] = _newOvmContractAddress;
        updatedContractsCounter += 1;

        return codeContractAddress;
    }
}
