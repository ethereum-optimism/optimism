pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import { FraudVerifier } from "./FraudVerifier.sol";
import { PartialStateManager } from "./PartialStateManager.sol";
import { ExecutionManager } from "./ExecutionManager.sol";
import { DataTypes } from "../utils/DataTypes.sol";
import { EthMerkleTrie } from "../utils/EthMerkleTrie.sol";

/**
 * @title StateTransitioner
 * @notice A contract which transitions a state from root one to another after a tx execution.
 */
contract StateTransitioner {
    /*
     * Data Structures
     */

    enum TransitionPhases {
        PreExecution,
        PostExecution,
        Complete
    }

    struct OVMTransactionData {
        uint256 timestamp;
        uint256 queueOrigin;
        address ovmEntrypoint;
        bytes callBytes;
        address fromAddress;
        address l1MsgSenderAddress;
        bool allowRevert;
    }


    /*
     * Contract Variables
     */

    TransitionPhases public currentTransitionPhase;

    uint public transitionIndex;
    bytes32 public stateRoot;
    bool public isTransactionSuccessfullyExecuted;

    FraudVerifier public fraudVerifier;
    PartialStateManager public stateManager;
    ExecutionManager executionManager;
    EthMerkleTrie public ethMerkleTrie;

    OVMTransactionData transactionData;


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

    constructor(
        uint _transitionIndex,
        bytes32 _preStateRoot,
        address _executionManagerAddress
    ) public {
        // The FraudVerifier always initializes a StateTransitioner in order to evaluate fraud.
        fraudVerifier = FraudVerifier(msg.sender);

        // Store the transitionIndex & state root which will be used during the proof.
        transitionIndex = _transitionIndex;
        stateRoot = _preStateRoot;

        // And of course set the ExecutionManager pointer.
        executionManager = ExecutionManager(_executionManagerAddress);

        // Finally we'll initialize a new state manager!
        stateManager = new PartialStateManager(address(this), address(executionManager));

        // Create a Merkle trie instance.
        ethMerkleTrie = new EthMerkleTrie();

        // And set our TransitionPhases to the PreExecution phase.
        currentTransitionPhase = TransitionPhases.PreExecution;
    }


    /*
     * Public Functions
     */

    /*****************************
     * Pre-Transaction Execution *
     *****************************/

    function proveContractInclusion(
        address _ovmContractAddress,
        address _codeContractAddress,
        uint _nonce,
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

        stateManager.insertVerifiedContract(_ovmContractAddress, _codeContractAddress, _nonce);
    }

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

        stateManager.insertVerifiedStorage(_ovmContractAddress, _slot, _value);
    }

    /*************************
     * Transaction Execution *
     *************************/

    function applyTransaction() public returns (bool) {
        // Initialize our execution context.
        stateManager.initNewTransactionExecution();
        executionManager.setStateManager(address(stateManager));

        // Execute the transaction via the execution manager.
        OVMTransactionData memory txData = getTransactionData();
        executionManager.executeTransaction(
            txData.timestamp,
            txData.queueOrigin,
            txData.ovmEntrypoint,
            txData.callBytes,
            txData.fromAddress,
            txData.l1MsgSenderAddress,
            txData.allowRevert
        );

        require(
            stateManager.existsInvalidStateAccessFlag() == false,
            "Detected an invalid state access."
        );

        currentTransitionPhase = TransitionPhases.PostExecution;

        return true;
    }

    /******************************
     * Post-Transaction Execution *
     ******************************/

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

    function proveUpdatedContract(
        bytes memory _stateTrieWitness
    ) public onlyDuringPhase(TransitionPhases.PostExecution) {
        (
            address ovmContractAddress,
            uint contractNonce
        ) = stateManager.popUpdatedContract();

        stateRoot = ethMerkleTrie.updateAccountNonce(
            ovmContractAddress,
            contractNonce,
            _stateTrieWitness,
            stateRoot
        );
    }

    function completeTransition() public onlyDuringPhase(TransitionPhases.PostExecution) {
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

    /***********************
     * Temporary Utilities *
     ***********************/

    function setTransactionData(
        OVMTransactionData memory _transactionData
    ) public {
        transactionData = _transactionData;
    }


    /*
     * Internal Functions
     */

    function getTransactionData() internal view returns (OVMTransactionData memory) {
        return transactionData;
    }
}
