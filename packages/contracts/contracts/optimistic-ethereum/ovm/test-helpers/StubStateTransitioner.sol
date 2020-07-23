pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import { FraudVerifier } from "../FraudVerifier.sol";
import { ExecutionManager } from "../ExecutionManager.sol";
import { ContractResolver } from "../../utils/resolvers/ContractResolver.sol";
import { DataTypes } from "../../utils/libraries/DataTypes.sol";
import { IStateTransitioner } from "../interfaces/IStateTransitioner.sol";

/**
 * @title StubStateTransitioner
 */
contract StubStateTransitioner is IStateTransitioner, ContractResolver {
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


    /*
     * Constructor
     */

    constructor(
        address _addressResolver,
        uint256 _transitionIndex,
        bytes32 _preStateRoot,
        bytes32 _ovmTransactionHash
    ) public ContractResolver(_addressResolver) {
        transitionIndex = _transitionIndex;
        preStateRoot = _preStateRoot;
        stateRoot = _preStateRoot;
        ovmTransactionHash = _ovmTransactionHash;
        currentTransitionPhase = TransitionPhases.PreExecution;

        fraudVerifier = FraudVerifier(msg.sender);
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
