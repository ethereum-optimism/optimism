// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Proxy Imports */
import { Proxy_Resolver } from "../../proxy/Proxy_Resolver.sol";

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_EthUtils } from "../../libraries/utils/Lib_EthUtils.sol";
import { Lib_EthMerkleTrie } from "../../libraries/trie/Lib_EthMerkleTrie.sol";

/* Interface Imports */
import { iOVM_ExecutionManager } from "../../iOVM/execution/iOVM_ExecutionManager.sol";
import { iOVM_StateManager } from "../../iOVM/execution/iOVM_StateManager.sol";
import { iOVM_StateManagerFactory } from "../../iOVM/execution/iOVM_StateManagerFactory.sol";
import { iOVM_StateTransitioner } from "../../iOVM/execution/iOVM_StateTransitioner.sol";

/**
 * @title OVM_StateTransitioner
 */
contract OVM_StateTransitioner is iOVM_StateTransitioner, Proxy_Resolver {

    /*******************
     * Data Structures *
     *******************/

    enum TransitionPhase {
        PRE_EXECUTION,
        POST_EXECUTION,
        COMPLETE
    }


    /*******************************************
     * Contract Variables: Contract References *
     *******************************************/

    iOVM_ExecutionManager internal ovmExecutionManager;
    iOVM_StateManager internal ovmStateManager;


    /*******************************************
     * Contract Variables: Internal Accounting *
     *******************************************/

    bytes32 internal preStateRoot;
    bytes32 internal postStateRoot;
    TransitionPhase internal phase;
    uint256 internal stateTransitionIndex;
    bytes32 internal transactionHash;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _proxyManager Address of the Proxy_Manager.
     * @param _stateTransitionIndex Index of the state transition being verified.
     * @param _preStateRoot State root before the transition was executed.
     * @param _transactionHash Hash of the executed transaction.
     */
    constructor(
        address _proxyManager,
        uint256 _stateTransitionIndex,
        bytes32 _preStateRoot,
        bytes32 _transactionHash
    )
        Proxy_Resolver(_proxyManager)
    {
        stateTransitionIndex = _stateTransitionIndex;
        preStateRoot = _preStateRoot;
        postStateRoot = _preStateRoot;
        transactionHash = _transactionHash;

        ovmExecutionManager = iOVM_ExecutionManager(resolve("OVM_ExecutionManager"));
        ovmStateManager = iOVM_StateManagerFactory(resolve("OVM_StateManagerFactory")).create(address(this));
    }


    /**********************
     * Function Modifiers *
     **********************/

    /**
     * Checks that a function is only run during a specific phase.
     * @param _phase Phase the function must run within.
     */
    modifier onlyDuringPhase(
        TransitionPhase _phase
    ) {
        require(
            phase == _phase,
            "Function must be called during the correct phase."
        );
        _;
    }

    
    /**********************************
     * Public Functions: State Access *
     **********************************/

    /**
     * Retrieves the state root before execution.
     * @return _preStateRoot State root before execution.
     */
    function getPreStateRoot()
        override
        public
        view
        returns (
            bytes32 _preStateRoot
        )
    {
        return preStateRoot;
    }

    /**
     * Retrieves the state root after execution.
     * @return _postStateRoot State root after execution.
     */
    function getPostStateRoot()
        override
        public
        view
        returns (
            bytes32 _postStateRoot
        )
    {
        return postStateRoot;
    }

    /**
     * Checks whether the transitioner is complete.
     * @return _complete Whether or not the transition process is finished.
     */
    function isComplete()
        override
        public
        view
        returns (
            bool _complete
        )
    {
        return phase == TransitionPhase.COMPLETE;
    }
    

    /***********************************
     * Public Functions: Pre-Execution *
     ***********************************/

    /**
     * Allows a user to prove the initial state of a contract.
     * @param _ovmContractAddress Address of the contract on the OVM.
     * @param _account Claimed account state.
     * @param _stateTrieWitness Proof of the account state.
     */
    function proveContractState(
        address _ovmContractAddress,
        Lib_OVMCodec.Account memory _account,
        bytes memory _stateTrieWitness
    )
        override
        public
        onlyDuringPhase(TransitionPhase.PRE_EXECUTION)
    {
        require(
            _account.codeHash == Lib_EthUtils.getCodeHash(_account.ethAddress),
            "Invalid code hash provided."
        );

        require(
            Lib_EthMerkleTrie.proveAccountState(
                _ovmContractAddress,
                _account,
                _stateTrieWitness,
                preStateRoot
            ),
            "Invalid account state provided."
        );

        ovmStateManager.putAccount(
            _ovmContractAddress,
            _account
        );
    }

    /**
     * Allows a user to prove the initial state of a contract storage slot.
     * @param _ovmContractAddress Address of the contract on the OVM.
     * @param _key Claimed account slot key.
     * @param _value Claimed account slot value.
     * @param _stateTrieWitness Proof of the account state.
     * @param _storageTrieWitness Proof of the storage slot.
     */
    function proveStorageSlot(
        address _ovmContractAddress,
        bytes32 _key,
        bytes32 _value,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness
    )
        override
        public
        onlyDuringPhase(TransitionPhase.PRE_EXECUTION)
    {
        require(
            ovmStateManager.hasAccount(_ovmContractAddress) == true,
            "Contract must be verified before proving a storage slot."
        );

        require(
            Lib_EthMerkleTrie.proveAccountStorageSlotValue(
                _ovmContractAddress,
                _key,
                _value,
                _stateTrieWitness,
                _storageTrieWitness,
                preStateRoot
            ),
            "Invalid account state provided."
        );

        ovmStateManager.putContractStorage(
            _ovmContractAddress,
            _key,
            _value
        );
    }


    /*******************************
     * Public Functions: Execution *
     *******************************/

    /**
     * Executes the state transition.
     * @param _transaction OVM transaction to execute.
     */
    function applyTransaction(
        Lib_OVMCodec.Transaction memory _transaction
    )
        override
        public
    {
        require(
            Lib_OVMCodec.hashTransaction(_transaction) == transactionHash,
            "Invalid transaction provided."
        );

        // TODO: Set state manager for EM here.

        ovmStateManager.setExecutionManager(resolveTarget("OVM_ExecutionManager"));
        ovmExecutionManager.run(_transaction, address(ovmStateManager));

        phase = TransitionPhase.POST_EXECUTION;
    }


    /************************************
     * Public Functions: Post-Execution *
     ************************************/

    /**
     * Allows a user to commit the final state of a contract.
     * @param _ovmContractAddress Address of the contract on the OVM.
     * @param _account Claimed account state.
     * @param _stateTrieWitness Proof of the account state.
     */
    function commitContractState(
        address _ovmContractAddress,
        Lib_OVMCodec.Account memory _account,
        bytes memory _stateTrieWitness
    )
        override
        public
        onlyDuringPhase(TransitionPhase.POST_EXECUTION)
    {
        require(
            ovmStateManager.commitAccount(_ovmContractAddress) == true,
            "Cannot commit an account that has not been changed."
        );

        postStateRoot = Lib_EthMerkleTrie.updateAccountState(
            _ovmContractAddress,
            _account,
            _stateTrieWitness,
            postStateRoot
        );
    }

    /**
     * Allows a user to commit the final state of a contract storage slot.
     * @param _ovmContractAddress Address of the contract on the OVM.
     * @param _key Claimed account slot key.
     * @param _value Claimed account slot value.
     * @param _stateTrieWitness Proof of the account state.
     * @param _storageTrieWitness Proof of the storage slot.
     */
    function commitStorageSlot(
        address _ovmContractAddress,
        bytes32 _key,
        bytes32 _value,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness
    )
        override
        public
        onlyDuringPhase(TransitionPhase.POST_EXECUTION)
    {
        require(
            ovmStateManager.commitContractStorage(_ovmContractAddress, _key) == true,
            "Cannot commit a storage slot that has not been changed."
        );

        postStateRoot = Lib_EthMerkleTrie.updateAccountStorageSlotValue(
            _ovmContractAddress,
            _key,
            _value,
            _stateTrieWitness,
            _storageTrieWitness,
            postStateRoot
        );
    }


    /**********************************
     * Public Functions: Finalization *
     **********************************/

    /**
     * Finalizes the transition process.
     */
    function completeTransition()
        override
        public
        onlyDuringPhase(TransitionPhase.POST_EXECUTION)
    {
        require(
            ovmStateManager.getTotalUncommittedAccounts() == 0,
            "All accounts must be committed before completing a transition."
        );

        require(
            ovmStateManager.getTotalUncommittedContractStorage() == 0,
            "All storage must be committed before completing a transition."
        );

        phase = TransitionPhase.COMPLETE;
    }
}
