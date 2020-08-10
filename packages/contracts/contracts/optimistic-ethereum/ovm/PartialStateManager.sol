pragma experimental ABIEncoderV2;

/* Contract Imports */
import { StateManager } from "./StateManager.sol";
import { StateTransitioner } from "./StateTransitioner.sol";
import { ExecutionManager } from "./ExecutionManager.sol";

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";

/* Testing Imports */
import { console } from "@nomiclabs/buidler/console.sol";

/**
 * @title PartialStateManager
 * @notice The PartialStateManager is used for the on-chain fraud proof checker.
 *         It is supplied with only the state which is used to execute a single transaction. This
 *         is unlike the FullStateManager which has access to every storage slot.
 */
contract PartialStateManager is ContractResolver {
    /*
     * Contract Constants
     */

    address constant ZERO_ADDRESS = 0x0000000000000000000000000000000000000000;


    /*
     * Contract Variables
     */

    StateTransitioner private stateTransitioner;

    mapping(address => mapping(bytes32 => bytes32)) private ovmContractStorage;
    mapping(address => uint) private ovmContractNonces;
    mapping(address => address) private ovmAddressToCodeContractAddress;

    bool public existsInvalidStateAccessFlag;

    mapping(address => mapping(bytes32 => bool)) public isVerifiedStorage;
    mapping(address => bool) public isVerifiedContract;
    mapping(uint => bytes32) private updatedStorageSlotContract;
    mapping(uint => bytes32) private updatedStorageSlotKey;
    mapping(address => mapping(bytes32 => bool)) private storageSlotTouched;
    uint public updatedStorageSlotCounter;
    mapping(uint => address) private updatedContracts;
    mapping(address => bool) private contractTouched;
    uint public updatedContractsCounter;


    /*
     * Modifiers
     */

    modifier onlyStateTransitioner {
        require(msg.sender == address(stateTransitioner));
        _;
    }

    modifier onlyExecutionManager {
        ExecutionManager executionManager = resolveExecutionManager();
        require(msg.sender == address(executionManager));
        _;
    }


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
     * Public Functions
     */

    /**
     * Initialize a new transaction execution
     */
    function initNewTransactionExecution()
        public
        onlyStateTransitioner
    {
        // #if FLAG_IS_DEBUG
        console.log("Initializing new transaction execution.");
        // #endif

        existsInvalidStateAccessFlag = false;
        updatedStorageSlotCounter = 0;
        updatedContractsCounter = 0;
    }

    /****************
    * Pre-Execution *
    ****************/

    /**
     * Inserts a verified storage slot.
     * @param _ovmContractAddress Address to insert a slot for.
     * @param _slot ID of the slot to insert.
     * @param _value Value for the provided slot.
     */
    function insertVerifiedStorage(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value
    )
        public
        onlyStateTransitioner
    {
        // #if FLAG_IS_DEBUG
        console.log("Inserting verified storage slot.");
        console.log("Contract address: %s", _ovmContractAddress);
        console.log("Slot ID:");
        console.logBytes32(_slot);
        console.log("Slot value:");
        console.logBytes32(_value);
        // #endif

        isVerifiedStorage[_ovmContractAddress][_slot] = true;
        ovmContractStorage[_ovmContractAddress][_slot] = _value;
    }

    /**
     * Inserts a verified contract address.
     * @param _ovmContractAddress Address of the contract on the OVM.
     * @param _codeContractAddress Address of the contract on the EVM.
     * @param _nonce Nonce for the provided contract.
     */
    function insertVerifiedContract(
        address _ovmContractAddress,
        address _codeContractAddress,
        uint _nonce
    )
        public
        onlyStateTransitioner
    {
        // #if FLAG_IS_DEBUG
        console.log("Inserting verified contract.");
        console.log("OVM contract address: %s", _ovmContractAddress);
        console.log("Code contract address: %s", _codeContractAddress);
        console.log("Contract nonce: %s", _nonce);
        // #endif

        isVerifiedContract[_ovmContractAddress] = true;
        ovmContractNonces[_ovmContractAddress] = _nonce;
        ovmAddressToCodeContractAddress[_ovmContractAddress] = _codeContractAddress;
    }

    /*****************
    * Post-Execution *
    *****************/

    /**
     * Peeks the next storage slot to update.
     * @return Information about the next storage slot to update.
     */
    function peekUpdatedStorageSlot()
        public
        view
        returns (
            address storageSlotContract,
            bytes32 storageSlotKey,
            bytes32 storageSlotValue
        )
    {
        require(updatedStorageSlotCounter > 0, "No more elements to update.");

        storageSlotContract = address(bytes20(updatedStorageSlotContract[updatedStorageSlotCounter - 1]));
        storageSlotKey = updatedStorageSlotKey[updatedStorageSlotCounter - 1];
        storageSlotValue = ovmContractStorage[storageSlotContract][storageSlotKey];

        return (storageSlotContract, storageSlotKey, storageSlotValue);
    }

    /**
     * Pops the next storage slot to update.
     * @return Information about the next storage slot to update.
     */
    function popUpdatedStorageSlot()
        public
        onlyStateTransitioner
        returns (
            address storageSlotContract,
            bytes32 storageSlotKey,
            bytes32 storageSlotValue
        )
    {
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

    /**
     * Peeks the next account state to update.
     * @return Information about the next account state to update.
     */
    function peekUpdatedContract()
        public
        view
        returns (
            address ovmContractAddress,
            uint contractNonce,
            bytes32 codeHash
        )
    {
        require(updatedContractsCounter > 0, "No more elements to update.");

        ovmContractAddress = address(bytes20(updatedContracts[updatedContractsCounter - 1]));
        contractNonce = ovmContractNonces[ovmContractAddress];

        address codeContractAddress = getCodeContractAddressView(ovmContractAddress);
        assembly {
            codeHash := extcodehash(codeContractAddress)
        }

        return (ovmContractAddress, contractNonce, codeHash);
    }

    /**
     * Pops the next account state to update.
     * @return Information about the next account state to update.
     */
    function popUpdatedContract()
        public
        onlyStateTransitioner
        returns (
            address ovmContractAddress,
            uint contractNonce,
            bytes32 codeHash
        )
    {
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
     * Get storage for OVM contract at some slot.
     * @param _ovmContractAddress The contract we're getting storage of.
     * @param _slot The slot we're querying.
     * @return The bytes32 value stored at the particular slot.
     */
    function getStorage(
        address _ovmContractAddress,
        bytes32 _slot
    )
        public
        onlyExecutionManager
        returns (bytes32)
    {
        flagIfNotVerifiedStorage(_ovmContractAddress, _slot);

        return ovmContractStorage[_ovmContractAddress][_slot];
    }

    /**
     * Get a storage slot without changing state.
     * @param _ovmContractAddress The contract we're getting storage of.
     * @param _slot The slot we're querying.
     * @return The bytes32 value stored at the particular slot.
     */
    function getStorageView(
        address _ovmContractAddress,
        bytes32 _slot
    )
        public
        view
        returns (bytes32)
    {
        return ovmContractStorage[_ovmContractAddress][_slot];
    }

    /**
     * Set storage for OVM contract at some slot.
     * @param _ovmContractAddress The contract we're setting storage of.
     * @param _slot The slot we're setting.
     * @param _value The value we will set the storage to.
     */
    function setStorage(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value
    )
        public
        onlyExecutionManager
    {
        ExecutionManager executionManager = resolveExecutionManager();

        if (
            !storageSlotTouched[_ovmContractAddress][_slot]
            && _ovmContractAddress != executionManager.METADATA_STORAGE_ADDRESS()
        ) {
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
     * Get the nonce for a particular OVM contract.
     * @param _ovmContractAddress The contract we're getting the nonce of.
     * @return The contract nonce used for contract creation.
     */
    function getOvmContractNonce(
        address _ovmContractAddress
    )
        public
        onlyExecutionManager
        returns (uint)
    {
        flagIfNotVerifiedContract(_ovmContractAddress);

        return ovmContractNonces[_ovmContractAddress];
    }

    /**
     * Get a contract nonce without touching state.
     * @param _ovmContractAddress The contract we're getting the nonce of.
     * @return The contract nonce used for contract creation.
     */
    function getOvmContractNonceView(
        address _ovmContractAddress
    )
        public
        view
        returns (uint)
    {
        return ovmContractNonces[_ovmContractAddress];
    }

    /**
     * Set the nonce for a particular OVM contract
     * @param _ovmContractAddress The contract we're setting the nonce of.
     * @param _value The new nonce.
     */
    function setOvmContractNonce(
        address _ovmContractAddress,
        uint _value
    )
        public
        onlyExecutionManager
    {
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
     * Increment the nonce for a particular OVM contract.
     * @param _ovmContractAddress The contract we're incrementing by 1 the nonce of.
     */
    function incrementOvmContractNonce(
        address _ovmContractAddress
    )
        public
        onlyExecutionManager
    {
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

    /**
     * Attaches some code contract to the desired OVM contract. This allows the Execution Manager
     * to later on get the code contract address to perform calls for this OVM contract.
     * @param _ovmContractAddress The address of the OVM contract we'd like to associate with some code.
     * @param _codeContractAddress The address of the code contract that's been deployed.
     */
    function associateCodeContract(
        address _ovmContractAddress,
        address _codeContractAddress
    )
        public
        onlyExecutionManager
    {
        ovmAddressToCodeContractAddress[_ovmContractAddress] = _codeContractAddress;
    }

    /**
     * Marks an address as newly created via ovmCREATE. Sets its nonce to zero and automatically
     * marks the contract as verified.
     * @param _ovmContractAddress Address of the contract to mark as verified.
     */
    function registerCreatedContract(
        address _ovmContractAddress
    )
        public
        onlyExecutionManager
    {
        isVerifiedContract[_ovmContractAddress] = true;
        setOvmContractNonce(_ovmContractAddress, 0);
    }

    /**
     * Lookup the code contract for some OVM contract, allowing CALL opcodes to be performed.
     * @param _ovmContractAddress The address of the OVM contract.
     * @return The associated code contract address.
     */
    function getCodeContractAddressView(
        address _ovmContractAddress
    )
        public
        view
        returns (address)
    {
        return ovmAddressToCodeContractAddress[_ovmContractAddress];
    }

    /**
     * @notice Lookup the code contract for some OVM contract, allowing ovmCALL operations to be performed.
     * @param _ovmContractAddress The address of the OVM contract.
     * @return The associated code contract address.
     */
    function getCodeContractAddressFromOvmAddress(
        address _ovmContractAddress
    )
        public
        onlyExecutionManager
        returns(address)
    {
        flagIfNotVerifiedContract(_ovmContractAddress);

        return ovmAddressToCodeContractAddress[_ovmContractAddress];
    }

    /**
     * Get the bytecode at some code  address. NOTE: This is code taken from Solidity docs here:
     * https://solidity.readthedocs.io/en/v0.5.0/assembly.html#example
     * @param _codeContractAddress The address of the code contract.
     * @return The bytecode at this address.
     */
    function getCodeContractBytecode(
        address _codeContractAddress
    )
        public
        view
        returns (bytes memory codeContractBytecode)
    {
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
     * Get the hash of the deployed bytecode of some code contract.
     * @param _codeContractAddress The address of the code contract.
     * @return The hash of the bytecode at this address.
     */
    function getCodeContractHash(
        address _codeContractAddress
    )
        public
        view
        returns (bytes32 _codeContractHash)
    {
        // NOTE: We don't need to verify that this is an authenticated contract
        // because this will always be proceeded by a call to
        // getCodeContractAddressFromOvmAddress(address _ovmContractAddress) in the EM which does this check.

        // TODO: Use EXTCODEHASH instead of this really inefficient stuff.
        bytes memory codeContractBytecode = getCodeContractBytecode(_codeContractAddress);
        _codeContractHash = keccak256(codeContractBytecode);
        return _codeContractHash;
    }


    /*
     * Private Functions
     */

    /**
     * Flags a storage slot if not verified.
     * @param _ovmContractAddress OVM contract address to flag a slot for.
     * @param _slot Slot ID to flag.
     */
    function flagIfNotVerifiedStorage(
        address _ovmContractAddress,
        bytes32 _slot
    )
        private
    {
        if (!isVerifiedStorage[_ovmContractAddress][_slot]) {
            // #if FLAG_IS_DEBUG
            console.log("Flagging as unverified because of a storage slot access.");
            console.log("Contract address: %s", _ovmContractAddress);
            console.log("Slot ID:");
            console.logBytes32(_slot);
            // #endif

            existsInvalidStateAccessFlag = true;
        }
    }

    /**
     * Flags a contract if not verified.
     * @param _ovmContractAddress OVM contract address to flag.
     */
    function flagIfNotVerifiedContract(
        address _ovmContractAddress
    )
        private
    {
        if (!isVerifiedContract[_ovmContractAddress]) {
            // #if FLAG_IS_DEBUG
            console.log("Flagging as unverified because of a contract access.");
            console.log("Contract address: %s", _ovmContractAddress);
            // #endif

            existsInvalidStateAccessFlag = true;
        }
    }


    /*
     * Contract Resolution
     */

    function resolveExecutionManager()
        internal
        view returns (ExecutionManager)
    {
        return ExecutionManager(resolveContract("ExecutionManager"));
    }
}