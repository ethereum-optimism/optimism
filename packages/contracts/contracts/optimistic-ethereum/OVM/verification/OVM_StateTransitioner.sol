// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_EthUtils } from "../../libraries/utils/Lib_EthUtils.sol";
import { Lib_SecureMerkleTrie } from "../../libraries/trie/Lib_SecureMerkleTrie.sol";

/* Interface Imports */
import { iOVM_StateTransitioner } from "../../iOVM/verification/iOVM_StateTransitioner.sol";
import { iOVM_BondManager } from "../../iOVM/verification/iOVM_BondManager.sol";
import { iOVM_ExecutionManager } from "../../iOVM/execution/iOVM_ExecutionManager.sol";
import { iOVM_StateManager } from "../../iOVM/execution/iOVM_StateManager.sol";
import { iOVM_StateManagerFactory } from "../../iOVM/execution/iOVM_StateManagerFactory.sol";

/* Contract Imports */
import { OVM_FraudContributor } from "./OVM_FraudContributor.sol";

/**
 * @title OVM_StateTransitioner
 */
contract OVM_StateTransitioner is Lib_AddressResolver, OVM_FraudContributor, iOVM_StateTransitioner {

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
     * @param _libAddressManager Address of the Address Manager.
     * @param _stateTransitionIndex Index of the state transition being verified.
     * @param _preStateRoot State root before the transition was executed.
     * @param _transactionHash Hash of the executed transaction.
     */
    constructor(
        address _libAddressManager,
        uint256 _stateTransitionIndex,
        bytes32 _preStateRoot,
        bytes32 _transactionHash
    )
        Lib_AddressResolver(_libAddressManager)
    {
        stateTransitionIndex = _stateTransitionIndex;
        preStateRoot = _preStateRoot;
        postStateRoot = _preStateRoot;
        transactionHash = _transactionHash;

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
        address _ethContractAddress,
        Lib_OVMCodec.EVMAccount memory _account,
        bytes memory _stateTrieWitness
    )
        override
        public
        onlyDuringPhase(TransitionPhase.PRE_EXECUTION)
        contributesToFraudProof(preStateRoot)
    {
        // Exit quickly to avoid unnecessary work.
        require(
            ovmStateManager.hasAccount(_ovmContractAddress) == false,
            "Account state has already been proven"
        );

        require(
            _account.codeHash == Lib_EthUtils.getCodeHash(_ethContractAddress),
            "Invalid code hash provided."
        );

        require(
            Lib_SecureMerkleTrie.verifyInclusionProof(
                abi.encodePacked(_ovmContractAddress),
                Lib_OVMCodec.encodeEVMAccount(_account),
                _stateTrieWitness,
                preStateRoot
            ),
            "Account state is not correct or invalid inclusion proof provided."
        );

        ovmStateManager.putAccount(
            _ovmContractAddress,
            Lib_OVMCodec.Account({
                nonce: _account.nonce,
                balance: _account.balance,
                storageRoot: _account.storageRoot,
                codeHash: _account.codeHash,
                ethAddress: _ethContractAddress,
                isFresh: false
            })
        );
    }

    /**
     * Allows a user to prove that an account does *not* exist in the state.
     * @param _ovmContractAddress Address of the contract on the OVM.
     * @param _stateTrieWitness Proof of the (empty) account state.
     */
    function proveEmptyContractState(
        address _ovmContractAddress,
        bytes memory _stateTrieWitness
    )
        override
        public
        onlyDuringPhase(TransitionPhase.PRE_EXECUTION)
        contributesToFraudProof(preStateRoot)
    {
        // Exit quickly to avoid unnecessary work.
        require(
            ovmStateManager.hasEmptyAccount(_ovmContractAddress) == false,
            "Account state has already been proven."
        );

        require(
            Lib_SecureMerkleTrie.verifyExclusionProof(
                abi.encodePacked(_ovmContractAddress),
                _stateTrieWitness,
                preStateRoot
            ),
            "Account is not empty or invalid inclusion proof provided."
        );

        ovmStateManager.putEmptyAccount(_ovmContractAddress);
    }

    /**
     * Allows a user to prove the initial state of a contract storage slot.
     * @param _ovmContractAddress Address of the contract on the OVM.
     * @param _key Claimed account slot key.
     * @param _value Claimed account slot value.
     * @param _storageTrieWitness Proof of the storage slot.
     */
    function proveStorageSlot(
        address _ovmContractAddress,
        bytes32 _key,
        bytes32 _value,
        bytes memory _storageTrieWitness
    )
        override
        public
        onlyDuringPhase(TransitionPhase.PRE_EXECUTION)
        contributesToFraudProof(preStateRoot)
    {
        // Exit quickly to avoid unnecessary work.
        require(
            ovmStateManager.hasContractStorage(_ovmContractAddress, _key) == false,
            "Storage slot has already been proven."
        );

        require(
            ovmStateManager.hasAccount(_ovmContractAddress) == true,
            "Contract must be verified before proving a storage slot."
        );

        (
            bool exists,
            bytes memory value
        ) = Lib_SecureMerkleTrie.get(
            abi.encodePacked(_key),
            _storageTrieWitness,
            ovmStateManager.getAccountStorageRoot(_ovmContractAddress)
        );

        if (exists == true) {
            require(
                keccak256(value) == keccak256(abi.encodePacked(_value)),
                "Provided storage slot value is invalid."
            );
        } else {
            require(
                _value == bytes32(0),
                "Provided storage slot value is invalid."
            );
        }

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
        onlyDuringPhase(TransitionPhase.PRE_EXECUTION)
        contributesToFraudProof(preStateRoot)
    {
        require(
            Lib_OVMCodec.hashTransaction(_transaction) == transactionHash,
            "Invalid transaction provided."
        );

        iOVM_ExecutionManager ovmExecutionManager = iOVM_ExecutionManager(resolve("OVM_ExecutionManager"));

        // We call `setExecutionManager` right before `run` (and not earlier) just in case the
        // OVM_ExecutionManager address was updated between the time when this contract was created
        // and when `applyTransaction` was called.
        ovmStateManager.setExecutionManager(address(ovmExecutionManager));

        // `run` always succeeds *unless* the user hasn't provided enough gas to `applyTransaction`
        // or an INVALID_STATE_ACCESS flag was triggered. Either way, we won't get beyond this line
        // if that's the case.
        ovmExecutionManager.run(_transaction, address(ovmStateManager));

        phase = TransitionPhase.POST_EXECUTION;
    }


    /************************************
     * Public Functions: Post-Execution *
     ************************************/

    /**
     * Allows a user to commit the final state of a contract.
     * @param _ovmContractAddress Address of the contract on the OVM.
     * @param _stateTrieWitness Proof of the account state.
     */
    function commitContractState(
        address _ovmContractAddress,
        bytes memory _stateTrieWitness
    )
        override
        public
        onlyDuringPhase(TransitionPhase.POST_EXECUTION)
        contributesToFraudProof(preStateRoot)
    {
        require(
            ovmStateManager.commitAccount(_ovmContractAddress) == true,
            "Account was not changed or has already been committed."
        );

        Lib_OVMCodec.Account memory account = ovmStateManager.getAccount(_ovmContractAddress);

        postStateRoot = Lib_SecureMerkleTrie.update(
            abi.encodePacked(_ovmContractAddress),
            Lib_OVMCodec.encodeEVMAccount(
                Lib_OVMCodec.toEVMAccount(account)
            ),
            _stateTrieWitness,
            postStateRoot
        );
    }

    /**
     * Allows a user to commit the final state of a contract storage slot.
     * @param _ovmContractAddress Address of the contract on the OVM.
     * @param _key Claimed account slot key.
     * @param _stateTrieWitness Proof of the account state.
     * @param _storageTrieWitness Proof of the storage slot.
     */
    function commitStorageSlot(
        address _ovmContractAddress,
        bytes32 _key,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness
    )
        override
        public
        onlyDuringPhase(TransitionPhase.POST_EXECUTION)
        contributesToFraudProof(preStateRoot)
    {
        require(
            ovmStateManager.commitContractStorage(_ovmContractAddress, _key) == true,
            "Storage slot was not changed or has already been committed."
        );

        Lib_OVMCodec.Account memory account = ovmStateManager.getAccount(_ovmContractAddress);
        bytes32 value = ovmStateManager.getContractStorage(_ovmContractAddress, _key);

        account.storageRoot = Lib_SecureMerkleTrie.update(
            abi.encodePacked(_key),
            abi.encodePacked(value),
            _storageTrieWitness,
            account.storageRoot
        );

        postStateRoot = Lib_SecureMerkleTrie.update(
            abi.encodePacked(_ovmContractAddress),
            Lib_OVMCodec.encodeEVMAccount(
                Lib_OVMCodec.toEVMAccount(account)
            ),
            _stateTrieWitness,
            postStateRoot
        );

        ovmStateManager.putAccount(_ovmContractAddress, account);
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
