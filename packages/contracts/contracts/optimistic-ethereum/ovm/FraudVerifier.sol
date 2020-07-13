pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import { DataTypes } from "../utils/DataTypes.sol";
import { RLPWriter } from "../utils/RLPWriter.sol";
import { StateCommitmentChain } from "../chain/StateCommitmentChain.sol";
import { CanonicalTransactionChain } from "../chain/CanonicalTransactionChain.sol";
import { ExecutionManager } from "./ExecutionManager.sol";
import { StateTransitioner } from "./StateTransitioner.sol";
import { IStateTransitioner } from "./interfaces/IStateTransitioner.sol";
import { StubStateTransitioner } from "./test-helpers/StubStateTransitioner.sol";

/**
 * @title FraudVerifier
 * @notice Manages fraud proof verification and modifies the state commitment
 * chain in the case that a transaction is shown to be invalid.
 */
contract FraudVerifier {
    /*
     * Contract Variables
     */

    ExecutionManager executionManager;
    StateCommitmentChain stateCommitmentChain;
    CanonicalTransactionChain canonicalTransactionChain;
    mapping (uint256 => IStateTransitioner) public stateTransitioners;

    bool isTest;

    /*
     * Constructor
     */

    /**
     * @param _executionManagerAddress Address of the ExecutionManager to use
     * during the execution of transactions suspected to be fraudulent.
     * @param _stateCommitmentChainAddress Address of the StateCommitmentChain
     * to read state roots from and write to in the case that a fraud proof is
     * successfully verified.
     */
    constructor(
        address _executionManagerAddress,
        address _stateCommitmentChainAddress,
        address _canonicalTransactionChainAddress,
        bool _isTest
    ) public {
        executionManager = ExecutionManager(_executionManagerAddress);
        stateCommitmentChain = StateCommitmentChain(_stateCommitmentChainAddress);
        canonicalTransactionChain = CanonicalTransactionChain(_canonicalTransactionChainAddress);

        isTest = _isTest;
    }


    /*
     * Public Functions
     */

    /**
     * @notice Initializes the fraud proof verification process. Creates a new
     * StateTransitioner instance if none already exists for the given state
     * transition index.
     * @param _preStateTransitionIndex Index of the state transition suspected
     * to be fraudulent.
     * @param _preStateRoot Root of the state trie before the state transition
     * was executed.
     * @param _preStateRootProof Inclusion proof for the given pre-state root.
     * Since state roots are submitted in batches and merklized, we cannot
     * simply read the state roots from the StateCommitmentChain.
     * @param _transactionData Data for the transaction suspected to be
     * fraudulent.
     * @param _transactionProof Inclusion proof for the given transaction data.
     * Since transactions are submitted in batches and merklized, we cannot
     * simply read the state roots from the CanonicalTransactionChain.
     */
    function initializeFraudVerification(
        uint256 _preStateTransitionIndex,
        bytes32 _preStateRoot,
        DataTypes.StateElementInclusionProof memory _preStateRootProof,
        DataTypes.OVMTransactionData memory _transactionData,
        DataTypes.TxElementInclusionProof memory _transactionProof
    ) public {
        // For user convenience; no point in carrying out extra work here if a
        // StateTransitioner instance already exists for the given state
        // transition index. Return early to save the user some gas.
        if (hasStateTransitioner(_preStateTransitionIndex, _preStateRoot)) {
            return;
        }

        require(
            verifyStateRoot(
                _preStateRoot,
                _preStateTransitionIndex,
                _preStateRootProof
            ),
            "Provided pre-state root inclusion proof is invalid."
        );

        require(
            verifyTransaction(
                _transactionData,
                _preStateTransitionIndex,
                _transactionProof
            ),
            "Provided transaction data is invalid."
        );

        // Note that a StateTransitioner may be overwritten when a state root
        // *before* its pre-state root is shown to be fraudulent. This would
        // invalidate the old StateTransitioner, creating the need to
        // initialize a new one with the correct pre-state root. A case like
        // this is handled by the hasStateTransitioner check above, which would
        // fail when the existing StateTransitioner's pre-state root does not
        // match the provided one.
        if (isTest) {
            stateTransitioners[_preStateTransitionIndex] = new StubStateTransitioner(
                _preStateTransitionIndex,
                _preStateRoot,
                _transactionData,
                address(executionManager)
            );
        } else {
            stateTransitioners[_preStateTransitionIndex] = new StateTransitioner(
                _preStateTransitionIndex,
                _preStateRoot,
                _transactionData,
                address(executionManager)
            );
        }
    }

    /**
     * @notice Finalizes the fraud verification process. Checks that the state
     * transitioner has executed the transition to completion and that the
     * resulting state root differs from the one previous published.
     * @param _preStateTransitionIndex Index of the state transition suspected
     * to be fraudulent.
     * @param _postStateRoot Published root of the state trie after the state
     * transition was executed. If the transition was indeed fraudulent, then
     * this root will differ from the one computed by the StateTransitioner.
     * @param _postStateRootProof Inclusion proof for the given pre-state root.
     * Since state roots are submitted in batches and merklized, we cannot
     * simply read the state roots from the StateCommitmentChain.
     */
    function finalizeFraudVerification(
        uint256 _preStateTransitionIndex,
        bytes32 _preStateRoot,
        DataTypes.StateElementInclusionProof memory _preStateRootProof,
        bytes32 _postStateRoot,
        DataTypes.StateElementInclusionProof memory _postStateRootProof
    ) public {
        IStateTransitioner stateTransitioner = stateTransitioners[_preStateTransitionIndex];

        // Fraud cannot be verified until the StateTransitioner has fully
        // executed the given state transition. Otherwise, the
        // StateTransitioner will always report an invalid root.
        require(
            stateTransitioner.isComplete(),
            "State transition process has not been completed."
        );

        // We want the StateTransitioner to be reusable in the case that yet
        // another invalid state root is published for the post-state. This
        // saves users the gas cost of executing the entire state transition
        // more than once. However, if a state root *before* the pre-state root
        // was found to be fraudulent, then the StateTransitioner is no longer
        // valid (since its execution is based on an outdated pre-state root).
        // We therefore need to check that the StateTransitioner was based on
        // the given pre-state root and that the pre-state root is still part
        // of the StateCommitmentChain.
        require(
            _preStateRoot == stateTransitioner.preStateRoot(),
            "Provided pre-state root does not match StateTransitioner."
        );
        require(
            verifyStateRoot(
                _preStateRoot,
                _preStateTransitionIndex,
                _preStateRootProof
            ),
            "Provided pre-state root inclusion proof is invalid."
        );

        require(
            verifyStateRoot(
                _postStateRoot,
                _preStateTransitionIndex + 1,
                _postStateRootProof
            ),
            "Provided post-state root inclusion proof is invalid."
        );

        // State transitions are fraudlent when the state root published to the
        // StateCommitmentChain differs from the one computed by the
        // StateTransitioner.
        require(
            _postStateRoot != stateTransitioner.stateRoot(),
            "State transition has not been proven fraudulent."
        );

        // If we're here, then the state transition was found to be fraudulent.
        // We therefore need to remove all state roots from the
        // StateCommitmentChain after (and including) the fraudulent root.
        // However, since state roots are submitted in batches, we'll actually
        // need to remove all *batches* after (and including) the one in which
        // the fraudulent root was published.
        stateCommitmentChain.deleteAfterInclusive(
            _postStateRootProof.batchIndex,
            _postStateRootProof.batchHeader
        );
    }

    /**
     * @notice Utility; checks whether a StateTransitioner exists for a given
     * state transition index. Can be used by clients to preemtively avoid
     * attempts to initialize the same StateTransitioner multiple times.
     * @param _stateTransitionIndex Index of the state transition suspected to
     * be fraudulent.
     * @param _preStateRoot Pre-state root used to initialize the transitioner.
     * @return `true` if a StateTransitioner exists, `false` otherwise.
     */
    function hasStateTransitioner(
        uint256 _stateTransitionIndex,
        bytes32 _preStateRoot
    ) public view returns (bool) {
        IStateTransitioner stateTransitioner = stateTransitioners[_stateTransitionIndex];

        return (
            (address(stateTransitioner) != address(0x0)) &&
            (stateTransitioner.preStateRoot() == _preStateRoot)
        );
    }


    /*
     * Internal Functions
     */

    /**
     * @notice Utility; verifies that a given state root is valid. Mostly just
     * a convenience wrapper around the current verification method within
     * StateCommitmentChain.
     * @param _stateRoot State trie root to prove is included in the commitment
     * chain.
     * @param _stateRootIndex Global index of the state root within the list of
     * all state roots.
     * @param _stateRootProof Inclusion proof for the given state root and
     * index pair.
     * @return `true` if the root exists within the StateCommitmentChain,
     * `false` otherwise.
     */
    function verifyStateRoot(
        bytes32 _stateRoot,
        uint256 _stateRootIndex,
        DataTypes.StateElementInclusionProof memory _stateRootProof
    ) internal view returns (bool) {
        return stateCommitmentChain.verifyElement(
            abi.encodePacked(_stateRoot),
            _stateRootIndex,
            _stateRootProof
        );
    }

    /**
     * @notice Utility; verifies that a given transaction is valid. Mostly just
     * a convenience wrapper around the current verification method within
     * CanonicalTransactionChain.
     * @param _transaction OVM transaction data to verify.
     * @param _transactionIndex Global index of the transaction within the list
     * of all transactions
     * @param _transactionProof Inclusion proof for the given transaction and
     * index pair.
     * @return `true` if the transaction exists within the
     * CanonicalTransactionChain, `false` otherwise.
     */
    function verifyTransaction(
        DataTypes.OVMTransactionData memory _transaction,
        uint256 _transactionIndex,
        DataTypes.TxElementInclusionProof memory _transactionProof
    ) internal view returns (bool) {
        return canonicalTransactionChain.verifyElement(
            encodeTransactionData(_transaction),
            _transactionIndex,
            _transactionProof
        );
    }

    /**
     * @notice Utility; RLP encodes an OVMTransactionData struct.
     * @dev Likely to be changed (if not moved to another contract). Currently
     * remaining here as to avoid modifying CanonicalTransactionChain. Unclear
     * whether or not this is the correct transaction structure, but it should
     * work for the meantime.
     * @param _transactionData Transaction data to encode.
     * @return RLP encoded transaction data.
     */
    function encodeTransactionData(
        DataTypes.OVMTransactionData memory _transactionData
    ) internal pure returns (bytes memory) {
        bytes[] memory raw = new bytes[](7);

        raw[0] = RLPWriter.encodeUint(_transactionData.timestamp);
        raw[1] = RLPWriter.encodeUint(_transactionData.queueOrigin);
        raw[2] = RLPWriter.encodeAddress(_transactionData.ovmEntrypoint);
        raw[3] = RLPWriter.encodeBytes(_transactionData.callBytes);
        raw[4] = RLPWriter.encodeAddress(_transactionData.fromAddress);
        raw[5] = RLPWriter.encodeAddress(_transactionData.l1MsgSenderAddress);
        raw[6] = RLPWriter.encodeBool(_transactionData.allowRevert);

        return RLPWriter.encodeList(raw);
    }
}
