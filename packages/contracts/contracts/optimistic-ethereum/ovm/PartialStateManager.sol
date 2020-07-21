pragma experimental ABIEncoderV2;

/* Internal Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { StateManager } from "./StateManager.sol";
import { StateTransitioner } from "./StateTransitioner.sol";
import { ExecutionManager } from "./ExecutionManager.sol";

/**
 * @title PartialStateManager
 * @notice The PartialStateManager is used for the on-chain fraud proof checker.
 *         It is supplied with only the state which is used to execute a single transaction. This
 *         is unlike the FullStateManager which has access to every storage slot.
 */
contract PartialStateManager is ContractResolver {
    address constant ZERO_ADDRESS = 0x0000000000000000000000000000000000000000;

    StateTransitioner stateTransitioner;

    mapping(address=>mapping(bytes32=>bytes32)) ovmContractStorage;
    mapping(address=>uint) ovmContractNonces;
    mapping(address=>address) ovmAddressToCodeContractAddress;

    bool public existsInvalidStateAccessFlag;

    mapping(address=>mapping(bytes32=>bool)) public isVerifiedStorage;
    mapping(address=>bool) public isVerifiedContract;
    mapping(uint=>bytes32) updatedStorageSlotContract;
    mapping(uint=>bytes32) updatedStorageSlotKey;
    mapping(address=>mapping(bytes32=>bool)) storageSlotTouched;
    uint public updatedStorageSlotCounter;
    mapping(uint=>address) updatedContracts;
    mapping(address=>bool) contractTouched;
    uint public updatedContractsCounter;

    modifier onlyStateTransitioner {
        require(msg.sender == address(stateTransitioner));
        _;
    }

    modifier onlyExecutionManager {
        ExecutionManager executionManager = resolveExecutionManager();
        require(msg.sender == address(executionManager));
        _;
    }

    /**
     * @notice Construct a new PartialStateManager
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

    /**
     * @notice Initialize a new transaction execution
     */
    function initNewTransactionExecution() onlyStateTransitioner external {
        existsInvalidStateAccessFlag = false;
        updatedStorageSlotCounter = 0;
        updatedContractsCounter = 0;
    }

    function flagIfNotVerifiedStorage(address _ovmContractAddress, bytes32 _slot) private {
        if (!isVerifiedStorage[_ovmContractAddress][_slot]) {
            existsInvalidStateAccessFlag = true;
        }
    }

    function flagIfNotVerifiedContract(address _ovmContractAddress) private {
        if (!isVerifiedContract[_ovmContractAddress]) {
            existsInvalidStateAccessFlag = true;
        }
    }

    /****************
    * Pre-Execution *
    ****************/

    function insertVerifiedStorage(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value
    ) external onlyStateTransitioner {
        isVerifiedStorage[_ovmContractAddress][_slot] = true;
        ovmContractStorage[_ovmContractAddress][_slot] = _value;
    }

    function insertVerifiedContract(
        address _ovmContractAddress,
        address _codeContractAddress,
        uint _nonce
    ) external onlyStateTransitioner {
        isVerifiedContract[_ovmContractAddress] = true;
        ovmContractNonces[_ovmContractAddress] = _nonce;
        ovmAddressToCodeContractAddress[_ovmContractAddress] = _codeContractAddress;
    }

    /*****************
    * Post-Execution *
    *****************/

    function peekUpdatedStorageSlot() public view returns (
        address storageSlotContract,
        bytes32 storageSlotKey,
        bytes32 storageSlotValue
    ) {
        require(updatedStorageSlotCounter > 0, "No more elements to update.");

        storageSlotContract = address(bytes20(updatedStorageSlotContract[updatedStorageSlotCounter - 1]));
        storageSlotKey = updatedStorageSlotKey[updatedStorageSlotCounter - 1];
        storageSlotValue = ovmContractStorage[storageSlotContract][storageSlotKey];

        return (storageSlotContract, storageSlotKey, storageSlotValue);
    }

    function popUpdatedStorageSlot() public onlyStateTransitioner returns (
        address storageSlotContract,
        bytes32 storageSlotKey,
        bytes32 storageSlotValue
    ) {
        (
            storageSlotContract,
            storageSlotKey,
            storageSlotValue
        ) = peekUpdatedStorageSlot();

        // Decrement the updatedStorageSlotCounter (we go reverse through the updated storage).
        // Note that when this hits zero we'll have updated all storage slots that were changed during
        // transaction execution.
        updatedStorageSlotCounter -= 1;

        return (storageSlotContract, storageSlotKey, storageSlotValue);
    }

    function peekUpdatedContract() public view returns (
        address ovmContractAddress,
        uint contractNonce,
        bytes32 codeHash
    ) {
        require(updatedContractsCounter > 0, "No more elements to update.");

        ovmContractAddress = address(bytes20(updatedContracts[updatedContractsCounter - 1]));
        contractNonce = ovmContractNonces[ovmContractAddress];

        address codeContractAddress = getCodeContractAddressView(ovmContractAddress);
        assembly {
            codeHash := extcodehash(codeContractAddress)
        }

        return (ovmContractAddress, contractNonce, codeHash);
    }

    function popUpdatedContract() public onlyStateTransitioner returns (
        address ovmContractAddress,
        uint contractNonce,
        bytes32 codeHash
    ) {
        (
            ovmContractAddress,
            contractNonce,
            codeHash
        ) = peekUpdatedContract();

        updatedContractsCounter -= 1;

        return (ovmContractAddress, contractNonce, codeHash);
    }

    /**********
    * Storage *
    **********/

    /**
     * @notice Get storage for OVM contract at some slot.
     * @param _ovmContractAddress The contract we're getting storage of.
     * @param _slot The slot we're querying.
     * @return The bytes32 value stored at the particular slot.
     */
    function getStorage(
        address _ovmContractAddress,
        bytes32 _slot
    ) onlyExecutionManager public returns (bytes32) {
        flagIfNotVerifiedStorage(_ovmContractAddress, _slot);

        return ovmContractStorage[_ovmContractAddress][_slot];
    }

    function getStorageView(
        address _ovmContractAddress,
        bytes32 _slot
    ) public view returns (bytes32) {
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
    ) onlyExecutionManager public {
        if (!storageSlotTouched[_ovmContractAddress][_slot]) {
            updatedStorageSlotContract[updatedStorageSlotCounter] = bytes32(bytes20(_ovmContractAddress));
            updatedStorageSlotKey[updatedStorageSlotCounter] = _slot;
            updatedStorageSlotCounter += 1;
            storageSlotTouched[_ovmContractAddress][_slot] = true;
        }

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
    function getOvmContractNonce(
        address _ovmContractAddress
    ) onlyExecutionManager public returns (uint) {
        flagIfNotVerifiedContract(_ovmContractAddress);

        return ovmContractNonces[_ovmContractAddress];
    }

    function getOvmContractNonceView(
        address _ovmContractAddress
    ) public view returns (uint) {
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
    ) onlyExecutionManager public {
        // TODO: Figure out if we actually need to verify contracts here.
        //flagIfNotVerifiedContract(_ovmContractAddress);

        if (!contractTouched[_ovmContractAddress]) {
            updatedContracts[updatedContractsCounter] = _ovmContractAddress;
            updatedContractsCounter += 1;
            contractTouched[_ovmContractAddress] = true;
        }

        // Return the nonce
        ovmContractNonces[_ovmContractAddress] = _value;
    }

    /**
     * @notice Increment the nonce for a particular OVM contract.
     * @param _ovmContractAddress The contract we're incrementing by 1 the nonce of.
     */
    function incrementOvmContractNonce(
        address _ovmContractAddress
    ) onlyExecutionManager public {
        flagIfNotVerifiedContract(_ovmContractAddress);

        if (!contractTouched[_ovmContractAddress]) {
            updatedContracts[updatedContractsCounter] = _ovmContractAddress;
            updatedContractsCounter += 1;
            contractTouched[_ovmContractAddress] = true;
        }

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
    function associateCodeContract(
        address _ovmContractAddress,
        address _codeContractAddress
    ) onlyExecutionManager public {
        ovmAddressToCodeContractAddress[_ovmContractAddress] = _codeContractAddress;
    }

    /**
     * @notice Marks an address as newly created via ovmCREATE. Sets its nonce to zero and automatically
     * marks the contract as verified.
     * @param _ovmContractAddress Address of the contract to mark as verified.
     */
    function associateCreatedContract(
        address _ovmContractAddress
    ) onlyExecutionManager public {
        isVerifiedContract[_ovmContractAddress] = true;
        setOvmContractNonce(_ovmContractAddress, 0);
    }

    /**
     * @notice Lookup the code contract for some OVM contract, allowing CALL opcodes to be performed.
     * @param _ovmContractAddress The address of the OVM contract.
     * @return The associated code contract address.
     */
    function getCodeContractAddressView(
        address _ovmContractAddress
    ) public view returns (address) {
        return ovmAddressToCodeContractAddress[_ovmContractAddress];
    }

    /**
     * @notice Lookup the code contract for some OVM contract, allowing ovmCALL operations to be performed.
     * @param _ovmContractAddress The address of the OVM contract.
     * @return The associated code contract address.
     */
    function getCodeContractAddressFromOvmAddress(
        address _ovmContractAddress
    ) onlyExecutionManager public returns(address) {
        flagIfNotVerifiedContract(_ovmContractAddress);

        return ovmAddressToCodeContractAddress[_ovmContractAddress];
    }

    /**
     * @notice Get the bytecode at some code  address. NOTE: This is code taken from Solidity docs here:
     *         https://solidity.readthedocs.io/en/v0.5.0/assembly.html#example
     * @param _codeContractAddress The address of the code contract.
     * @return The bytecode at this address.
     */
    function getCodeContractBytecode(
        address _codeContractAddress
    ) public view returns (bytes memory codeContractBytecode) {
        // NOTE: We don't need to verify that this is an authenticated contract
        // because this will always be proceeded by a call to
        // getCodeContractAddressFromOvmAddress(address _ovmContractAddress) in the EM which does this check.

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
        // NOTE: We don't need to verify that this is an authenticated contract
        // because this will always be proceeded by a call to
        // getCodeContractAddressFromOvmAddress(address _ovmContractAddress) in the EM which does this check.

        // TODO: Use EXTCODEHASH instead of this really inefficient stuff.
        bytes memory codeContractBytecode = getCodeContractBytecode(_codeContractAddress);
        _codeContractHash = keccak256(codeContractBytecode);
        return _codeContractHash;
    }


    /*
     * Contract Resolution
     */

    function resolveExecutionManager() internal view returns (ExecutionManager) {
        return ExecutionManager(resolveContract("ExecutionManager"));
    }
}