pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import { FraudVerifier } from "../FraudVerifier.sol";
import { ExecutionManager } from "../ExecutionManager.sol";
import { DataTypes } from "../../utils/DataTypes.sol";
import { IStateTransitioner } from "../interfaces/IStateTransitioner.sol";

/**
 * @title StubStateTransitioner
 */
contract StubStateTransitioner is IStateTransitioner {
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
    bytes32 public ovmTransactionHash;

    FraudVerifier public fraudVerifier;
    ExecutionManager executionManager;


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
        bytes32 _ovmTransactionHash,
        address _executionManagerAddress
    ) public {
        transitionIndex = _transitionIndex;
        preStateRoot = _preStateRoot;
        stateRoot = _preStateRoot;
        ovmTransactionHash = _ovmTransactionHash;
        currentTransitionPhase = TransitionPhases.PreExecution;

        fraudVerifier = FraudVerifier(msg.sender);
        executionManager = ExecutionManager(_executionManagerAddress);
    }

    function proveContractInclusion(
        address _ovmContractAddress,
        address _codeContractAddress,
        uint256 _nonce,
        bytes memory _stateTrieWitness
    ) public {
        return;
    }

    function proveStorageSlotInclusion(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness
    ) public {
        return;
    }

    function applyTransaction(
        DataTypes.OVMTransactionData memory _transactionData
    ) public {
        return;
    }

    function proveUpdatedStorageSlot(
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness
    ) public {
        return;
    }

    function proveUpdatedContract(
        bytes memory _stateTrieWitness
    ) public {
        return;
    }

    function completeTransition() public {
        currentTransitionPhase = TransitionPhases.Complete;
    }

    function isComplete() public view returns (bool) {
        return (currentTransitionPhase == TransitionPhases.Complete);
    }

    function setStateRoot(bytes32 _stateRoot) public {
        stateRoot = _stateRoot;
    }
}
