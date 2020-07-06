pragma solidity ^0.5.0;

import { FraudVerifier } from "./FraudVerifier.sol";
import { PartialStateManager } from "./PartialStateManager.sol";
import { ExecutionManager } from "./ExecutionManager.sol";
import { EthMerkleTrie } from "../utils/EthMerkleTrie.sol";

/**
 * @title StateTransitioner
 * @notice A contract which transitions a state from root one to another after a tx execution.
 */
contract StateTransitioner {
    enum TransitionPhases {
        PreExecution,
        PostExecution,
        Complete
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


    /*
     * Modifiers
     */

    modifier preExecutionPhase {
        require(
            currentTransitionPhase == TransitionPhases.PreExecution,
            "Must be called during correct phase!"
        );
        _;
    }

    modifier postExecutionPhase {
        require(
            currentTransitionPhase == TransitionPhases.PostExecution,
            "Must be called during correct phase!"
        );
        _;
    }

    modifier completePhase {
        require(
            currentTransitionPhase == TransitionPhases.Complete,
            "Must be called during correct phase!"
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

        // And set our TransitionPhases to the PreExecution phase.
        currentTransitionPhase = TransitionPhases.PreExecution;
    }


    /*
     * External Functions
     */

    /*****************************
     * Pre-Transaction Execution *
     *****************************/

    function proveContractInclusion(
        address _ovmContractAddress,
        address _codeContractAddress,
        uint _nonce,
        bytes memory _stateTrieWitness
    ) public preExecutionPhase {
        bytes32 codeHash;
        assembly {
            codeHash := extcodehash(_codeContractAddress)
        }

        require(EthMerkleTrie.proveAccountState(
            _ovmContractAddress,
            _nonce,
            uint256(0),
            bytes32(''),
            codeHash,
            true,
            false,
            false,
            true,
        )

        stateManager.insertVerifiedContract(_ovmContractAddress, _codeContractAddress, _nonce);
    }

    function proveStorageSlotInclusion(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value
    ) public preExecutionPhase {
        require(
            stateManager.isVerifiedContract(_ovmContractAddress),
            "Contract must be verified before proving storage!"
        );
        // TODO: Verify an inclusion proof of the storage slot!

        stateManager.insertVerifiedStorage(_ovmContractAddress, _slot, _value);
    }

    /*************************
     * Transaction Execution *
     *************************/

    function applyTransaction() public returns(bool) {
        stateManager.initNewTransactionExecution();
        executionManager.setStateManager(address(stateManager));
        // TODO: Get the transaction from the _transitionIndex. For now this'll just be dummy data
        executionManager.executeTransaction(
            0,
            0,
            0x1212121212121212121212121212121212121212,
            "0x12",
            0x1212121212121212121212121212121212121212,
            0x1212121212121212121212121212121212121212,
            false
        );
        require(stateManager.existsInvalidStateAccessFlag() == false, "Detected invalid state access!");
        currentTransitionPhase = TransitionPhases.PostExecution;

        // This will allow people to start updating the state root!
        return true;
    }

    /******************************
     * Post-Transaction Execution *
     ******************************/

    function proveUpdatedStorageSlot() public postExecutionPhase {
        (
            address storageSlotContract,
            bytes32 storageSlotKey,
            bytes32 storageSlotValue
        ) = stateManager.popUpdatedStorageSlot();
        // TODO: Prove inclusion / make this update to the root
    }

    function proveUpdatedContract() public postExecutionPhase {
        (
            address ovmContractAddress,
            uint contractNonce
        ) = stateManager.popUpdatedContract();
        // TODO: Prove inclusion / make this update to the root
    }

    function completeTransition() public postExecutionPhase {
        require(
            stateManager.updatedStorageSlotCounter() == 0,
            "There's still updated storage to account for!"
        );
        require(
            stateManager.updatedStorageSlotCounter() == 0,
            "There's still updated contracts to account for!"
        );
        // Tada! We did it reddit!

        currentTransitionPhase = TransitionPhases.Complete;
    }
}
