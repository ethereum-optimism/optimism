pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import { FraudVerifier } from "./FraudVerifier.sol";
import { PartialStateManager } from "./PartialStateManager.sol";
import { ExecutionManager } from "./ExecutionManager.sol";
import { IStateTransitioner } from "./interfaces/IStateTransitioner.sol";
import { DataTypes } from "../utils/DataTypes.sol";
import { EthMerkleTrie } from "../utils/EthMerkleTrie.sol";

/**
 * @title StateTransitioner
 * @notice Manages the execution of a transaction suspected to be fraudulent.
 */
contract StateTransitioner is IStateTransitioner {
    /*
     * Data Structures
     */

    enum TransitionPhases {
        PreExecution,
        PostExecution,
        Complete
    }


    /*
     * Contract Constants
     */

    bytes32 constant BYTES32_NULL = bytes32('');
    uint256 constant UINT256_NULL = uint256(0);


    /*
     * Contract Variables
     */

    TransitionPhases public currentTransitionPhase;
    uint256 public transitionIndex;
    bytes32 public preStateRoot;
    bytes32 public stateRoot;

    FraudVerifier public fraudVerifier;
    PartialStateManager public stateManager;
    ExecutionManager executionManager;
    EthMerkleTrie public ethMerkleTrie;

    DataTypes.OVMTransactionData transactionData;


    /*
     * Modifiers
     */

    modifier onlyDuringPhase(TransitionPhases _phase) {
        require(
            currentTransitionPhase == _phase,
            "Must be called during the correct phase."
        );
        _;
    }


    /*
     * Constructor
     */

    /**
     * @param _transitionIndex Position of the transaction suspected to be
     * fraudulent within the canonical transaction chain.
     * @param _preStateRoot Root of the state trie before the transaction was
     * executed.
     * @param _executionManagerAddress Address of the ExecutionManager to be
     * used during transaction execution.
     */
    constructor(
        uint256 _transitionIndex,
        bytes32 _preStateRoot,
        DataTypes.OVMTransactionData memory _transactionData,
        address _executionManagerAddress
    ) public {
        transitionIndex = _transitionIndex;
        preStateRoot = _preStateRoot;
        stateRoot = _preStateRoot;
        transactionData = _transactionData;
        currentTransitionPhase = TransitionPhases.PreExecution;

        fraudVerifier = FraudVerifier(msg.sender);
        executionManager = ExecutionManager(_executionManagerAddress);
        stateManager = new PartialStateManager(address(this), address(executionManager));
        ethMerkleTrie = new EthMerkleTrie();
    }


    /*
     * Public Functions
     */

    /*****************************
     * Pre-Transaction Execution *
     *****************************/

    /**
     * @notice Allows a user to prove the state for a given contract. Currently
     * only requires that the user prove the nonce. Only callable before the
     * transaction suspected to be fraudulent has been executed.
     * @param _ovmContractAddress Address of the contract on the OVM.
     * @param _codeContractAddress Address of the above contract on the EVM.
     * @param _nonce Claimed current nonce of the contract.
     * @param _stateTrieWitness Merkle trie inclusion proof for the contract
     * within the state trie.
     */
    function proveContractInclusion(
        address _ovmContractAddress,
        address _codeContractAddress,
        uint256 _nonce,
        bytes memory _stateTrieWitness
    ) public onlyDuringPhase(TransitionPhases.PreExecution) {
        bytes32 codeHash;
        assembly {
            codeHash := extcodehash(_codeContractAddress)
        }

        require (
            ethMerkleTrie.proveAccountState(
                _ovmContractAddress,
                DataTypes.AccountState({
                    nonce: _nonce,
                    balance: uint256(0),
                    storageRoot: bytes32(''),
                    codeHash: codeHash
                }),
                DataTypes.ProofMatrix({
                    checkNonce: true,
                    checkBalance: false,
                    checkStorageRoot: false,
                    checkCodeHash: true
                }),
                _stateTrieWitness,
                stateRoot
            ),
            "Invalid account state provided."
        );

        stateManager.insertVerifiedContract(
            _ovmContractAddress,
            _codeContractAddress,
            _nonce
        );
    }

    /**
     * @notice Allows a user to prove the value of a given storage slot for
     * some contract. Only callable before the transaction suspected to be
     * fraudulent has been executed.
     * @param _ovmContractAddress Address of the contract on the OVM.
     * @param _slot Key for the storage slot to prove.
     * @param _value Value for the storage slot to prove.
     * @param _stateTrieWitness Merkle trie inclusion proof for the contract
     * within the state trie.
     * @param _storageTrieWitness Merkle trie inclusion proof for the specific
     * storage slot being proven within the account's storage trie.
     */
    function proveStorageSlotInclusion(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness
    ) public onlyDuringPhase(TransitionPhases.PreExecution) {
        require(
            stateManager.isVerifiedContract(_ovmContractAddress),
            "Contract must be verified before proving storage!"
        );

        require (
            ethMerkleTrie.proveAccountStorageSlotValue(
                _ovmContractAddress,
                _slot,
                _value,
                _stateTrieWitness,
                _storageTrieWitness,
                stateRoot
            ),
            "Invalid account state provided."
        );

        stateManager.insertVerifiedStorage(
            _ovmContractAddress,
            _slot,
            _value
        );
    }

    /*************************
     * Transaction Execution *
     *************************/

    /**
    * @notice Executes the transaction suspected to be fraudulent via the
    * ExecutionManager. Will revert if the transaction attempts to access
    * state that has not been proven during the pre-execution phase.
     */
    function applyTransaction() public {
        // Initialize our execution context.
        stateManager.initNewTransactionExecution();
        executionManager.setStateManager(address(stateManager));

        // Execute the transaction via the execution manager.
        executionManager.executeTransaction(
            transactionData.timestamp,
            transactionData.queueOrigin,
            transactionData.ovmEntrypoint,
            transactionData.callBytes,
            transactionData.fromAddress,
            transactionData.l1MsgSenderAddress,
            transactionData.allowRevert
        );

        require(
            stateManager.existsInvalidStateAccessFlag() == false,
            "Detected an invalid state access."
        );

        currentTransitionPhase = TransitionPhases.PostExecution;
    }

    /******************************
     * Post-Transaction Execution *
     ******************************/

    /**
     * @notice Updates the root of the state trie by making a modification to
     * a contract's storage slot. Contract storage to be modified depends on a
     * stack of slots modified during execution.
     * @param _stateTrieWitness Merkle trie inclusion proof for the contract
     * within the current state trie.
     * @param _storageTrieWitness Merkle trie inclusion proof for the slot
     * being modified within the contract's storage trie.
     */
    function proveUpdatedStorageSlot(
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness
    ) public onlyDuringPhase(TransitionPhases.PostExecution) {
        (
            address storageSlotContract,
            bytes32 storageSlotKey,
            bytes32 storageSlotValue
        ) = stateManager.popUpdatedStorageSlot();

        stateRoot = ethMerkleTrie.updateAccountStorageSlotValue(
            storageSlotContract,
            storageSlotKey,
            storageSlotValue,
            _stateTrieWitness,
            _storageTrieWitness,
            stateRoot
        );
    }

    /**
     * @notice Updates the root of the state trie by making a modification to
     * a contract's state. Contract to be modified depends on a stack of
     * contract states modified during execution.
     * @param _stateTrieWitness Merkle trie inclusion proof for the contract
     * within the current state trie.
     */
    function proveUpdatedContract(
        bytes memory _stateTrieWitness
    ) public onlyDuringPhase(TransitionPhases.PostExecution) {
        (
            address ovmContractAddress,
            uint contractNonce,
            bytes32 codeHash
        ) = stateManager.popUpdatedContract();

        stateRoot = ethMerkleTrie.updateAccountState(
            ovmContractAddress,
            DataTypes.AccountState({
                nonce: contractNonce,
                balance: UINT256_NULL,
                storageRoot: BYTES32_NULL,
                codeHash: codeHash
            }),
            DataTypes.ProofMatrix({
                checkNonce: true,
                checkBalance: false,
                checkStorageRoot: false,
                checkCodeHash: codeHash != 0x0
            }),
            _stateTrieWitness,
            stateRoot
        );
    }

    /**
     * @notice Finalizes the state transition process once all state trie or
     * storage trie modifications have been reflected in the state root.
     */
    function completeTransition()
        public
        onlyDuringPhase(TransitionPhases.PostExecution)
    {
        require(
            stateManager.updatedStorageSlotCounter() == 0,
            "There's still updated storage to account for!"
        );
        require(
            stateManager.updatedStorageSlotCounter() == 0,
            "There's still updated contracts to account for!"
        );

        currentTransitionPhase = TransitionPhases.Complete;
    }

    /**
     * @notice Utility; checks whether the process is complete.
     * @return `true` if the process is complete, `false` otherwise.
     */
    function isComplete() public view returns (bool) {
        return (currentTransitionPhase == TransitionPhases.Complete);
    }
}
