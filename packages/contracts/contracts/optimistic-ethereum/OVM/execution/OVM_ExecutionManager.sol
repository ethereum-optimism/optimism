// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_EthUtils } from "../../libraries/utils/Lib_EthUtils.sol";

/* Interface Imports */
import { iOVM_DataTypes } from "../../iOVM/codec/iOVM_DataTypes.sol";
import { iOVM_ExecutionManager } from "../../iOVM/execution/iOVM_ExecutionManager.sol";
import { iOVM_StateManager } from "../../iOVM/execution/iOVM_StateManager.sol";
import { iOVM_SafetyChecker } from "../../iOVM/execution/iOVM_SafetyChecker.sol";

/**
 * @title OVM_ExecutionManager
 */
contract OVM_ExecutionManager is iOVM_ExecutionManager {

    /********************************
     * External Contract References *
     ********************************/

    iOVM_SafetyChecker public ovmSafetyChecker;
    iOVM_StateManager public ovmStateManager;


    /*******************************
     * Execution Context Variables *
     *******************************/

    GlobalContext internal globalContext;
    TransactionContext internal transactionContext;
    MessageContext internal messageContext;

    TransactionRecord internal transactionRecord;
    MessageRecord internal messageRecord;


    /**************************
     * Gas Metering Constants *
     **************************/

    uint256 constant NUISANCE_GAS_SLOAD = 20000;
    uint256 constant NUISANCE_GAS_SSTORE = 20000;
    uint256 constant NUISANCE_GAS_PER_CONTRACT_BYTE = 100;
    uint256 constant MIN_GAS_FOR_INVALID_STATE_ACCESS = 30000;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _ovmSafetyChecker Address of the iOVM_SafetyChecker implementation.
     */
    constructor(
        address _ovmSafetyChecker
    ) {
        ovmSafetyChecker = iOVM_SafetyChecker(_ovmSafetyChecker);
    }

    
    /**********************
     * Function Modifiers *
     **********************/

    /**
     * Applies a net gas cost refund to a transaction to account for the difference in execution
     * between L1 and L2.
     * @param _cost Gas cost for the function after the refund.
     */
    modifier netGasCost(
        uint256 _cost
    ) {
        uint256 preExecutionGas = gasleft();
        _;
        uint256 postExecutionGas = gasleft();

        // We want to refund everything *except* the specified cost.
        transactionRecord.ovmGasRefund += (
            (preExecutionGas - postExecutionGas) - _cost
        );
    }


    /************************************
     * Transaction Execution Entrypoint *
     ************************************/

    /**
     * Starts the execution of a transaction via the OVM_ExecutionManager.
     * @param _transaction Transaction data to be executed.
     * @param _ovmStateManager iOVM_StateManager implementation providing account state.
     */
    function run(
        iOVM_DataTypes.OVMTransactionData memory _transaction,
        address _ovmStateManager
    )
        override
        public
    {
        
    }


    /******************************
     * Opcodes: Execution Context *
     ******************************/

    /**
     * @notice Overrides CALLER.
     * @return _CALLER Address of the CALLER within the current message context.
     */
    function ovmCALLER()
        override
        public
        returns (
            address _CALLER
        )
    {
        return messageContext.ovmCALLER;
    }

    /**
     * @notice Overrides ADDRESS.
     * @return _ADDRESS Active ADDRESS within the current message context.
     */
    function ovmADDRESS()
        override
        public
        returns (
            address _ADDRESS
        )
    {
        return messageContext.ovmADDRESS;
    }

    /**
     * @notice Overrides ORIGIN.
     * @return _ORIGIN Address of the ORIGIN within the transaction context.
     */
    function ovmORIGIN()
        override
        public
        returns (
            address _ORIGIN
        )
    {
        return transactionContext.ovmORIGIN;
    }

    /**
     * @notice Overrides TIMESTAMP.
     * @return _TIMESTAMP Value of the TIMESTAMP within the transaction context.
     */
    function ovmTIMESTAMP()
        override
        public
        returns (
            uint256 _TIMESTAMP
        )
    {
        return transactionContext.ovmTIMESTAMP;
    }

    /**
     * @notice Overrides GASLIMIT.
     * @return _GASLIMIT Value of the block's GASLIMIT within the transaction context.
     */
    function ovmGASLIMIT()
        override
        public
        returns (
            uint256 _GASLIMIT
        )
    {
        return transactionContext.ovmGASLIMIT;
    }

    /**
     * @notice Overrides CHAINID.
     * @return _CHAINID Value of the chain's CHAINID within the global context.
     */
    function ovmCHAINID()
        override
        public
        returns (
            uint256 _CHAINID
        )
    {
        return globalContext.ovmCHAINID;
    }


    /********************
     * Opcodes: Halting *
     ********************/

    /**
     * @notice Overrides REVERT.
     * @param _data Bytes data to pass along with the REVERT.
     */
    function ovmREVERT(
        bytes memory _data
    )
        override
        public
    {
        _revertWithFlag(RevertFlag.INTENTIONAL_REVERT, _data);
    }


    /******************************
     * Opcodes: Contract Creation *
     ******************************/

    /**
     * @notice Overrides CREATE.
     * @param _bytecode Code to be used to CREATE a new contract.
     * @return _contract Address of the created contract.
     */
    function ovmCREATE(
        bytes memory _bytecode
    )
        override
        public
        netGasCost(40000 + _bytecode.length * 100)
        returns (
            address _contract
        )
    {
        // Creator is always the current ADDRESS.
        address creator = ovmADDRESS();

        // Generate the correct CREATE address.
        address contractAddress = Lib_EthUtils.getAddressForCREATE(
            creator,
            _getAccount(creator).nonce
        );

        _createContract(
            contractAddress,
            _bytecode
        );

        return contractAddress;
    }

    /**
     * @notice Overrides CREATE2.
     * @param _bytecode Code to be used to CREATE2 a new contract.
     * @param _salt Value used to determine the contract's address.
     * @return _contract Address of the created contract.
     */
    function ovmCREATE2(
        bytes memory _bytecode,
        bytes32 _salt
    )
        override
        public
        netGasCost(40000 + _bytecode.length * 100)
        returns (
            address _contract
        )
    {
        // Creator is always the current ADDRESS.
        address creator = ovmADDRESS();

        // Generate the correct CREATE2 address.
        address contractAddress = Lib_EthUtils.getAddressForCREATE2(
            creator,
            _bytecode,
            _salt
        );

        _createContract(
            contractAddress,
            _bytecode
        );

        return contractAddress;
    }


    /*********************************
     * Opcodes: Contract Interaction *
     *********************************/

    /**
     * @notice Overrides CALL.
     * @param _gasLimit Amount of gas to be passed into this call.
     * @param _address Address of the contract to call.
     * @param _calldata Data to send along with the call.
     * @return _success Whether or not the call returned (rather than reverted).
     * @return _returndata Data returned by the call.
     */
    function ovmCALL(
        uint256 _gasLimit,
        address _address,
        bytes memory _calldata
    )
        override
        public
        netGasCost(100000)
        returns (
            bool _success,
            bytes memory _returndata
        )
    {
        // CALL updates the CALLER and ADDRESS.
        MessageContext memory nextMessageContext = messageContext;
        nextMessageContext.ovmCALLER = nextMessageContext.ovmADDRESS;
        nextMessageContext.ovmADDRESS = _address;

        return _callContract(
            nextMessageContext,
            _gasLimit,
            _address,
            _calldata
        );
    }

    /**
     * @notice Overrides STATICCALL.
     * @param _gasLimit Amount of gas to be passed into this call.
     * @param _address Address of the contract to call.
     * @param _calldata Data to send along with the call.
     * @return _success Whether or not the call returned (rather than reverted).
     * @return _returndata Data returned by the call.
     */
    function ovmSTATICCALL(
        uint256 _gasLimit,
        address _address,
        bytes memory _calldata
    )
        override
        public
        netGasCost(80000)
        returns (
            bool _success,
            bytes memory _returndata
        )
    {
        // STATICCALL updates the CALLER, updates the ADDRESS, and runs in a static context.
        MessageContext memory nextMessageContext = messageContext;
        nextMessageContext.ovmCALLER = nextMessageContext.ovmADDRESS;
        nextMessageContext.ovmADDRESS = _address;
        nextMessageContext.isStatic = true;

        return _callContract(
            nextMessageContext,
            _gasLimit,
            _address,
            _calldata
        );
    }

    /**
     * @notice Overrides DELEGATECALL.
     * @param _gasLimit Amount of gas to be passed into this call.
     * @param _address Address of the contract to call.
     * @param _calldata Data to send along with the call.
     * @return _success Whether or not the call returned (rather than reverted).
     * @return _returndata Data returned by the call.
     */
    function ovmDELEGATECALL(
        uint256 _gasLimit,
        address _address,
        bytes memory _calldata
    )
        override
        public
        netGasCost(40000)
        returns (
            bool _success,
            bytes memory _returndata
        )
    {
        // DELEGATECALL does not change anything about the message context.
        MessageContext memory nextMessageContext = messageContext;

        return _callContract(
            nextMessageContext,
            _gasLimit,
            _address,
            _calldata
        );
    }


    /************************************
     * Opcodes: Contract Storage Access *
     ************************************/

    /**
     * @notice Overrides SLOAD.
     * @param _key 32 byte key of the storage slot to load.
     * @return _value 32 byte value of the requested storage slot.
     */
    function ovmSLOAD(
        bytes32 _key
    )
        override
        public
        netGasCost(40000)
        returns (
            bytes32 _value
        )
    {
        // We always SLOAD from the storage of ADDRESS.
        address contractAddress = ovmADDRESS();

        return _getContractStorage(
            contractAddress,
            _key
        );
    }

    /**
     * @notice Overrides SSTORE.
     * @param _key 32 byte key of the storage slot to set.
     * @param _value 32 byte value for the storage slot.
     */
    function ovmSSTORE(
        bytes32 _key,
        bytes32 _value
    )
        override
        public
        netGasCost(60000)
    {
        // We always SSTORE to the storage of ADDRESS.
        address contractAddress = ovmADDRESS();

        _putContractStorage(
            contractAddress,
            _key,
            _value
        );
    }


    /*********************************
     * Opcodes: Contract Code Access *
     *********************************/

    /**
     * @notice Overrides EXTCODECOPY.
     * @param _contract Address of the contract to copy code from.
     * @param _offset Offset in bytes from the start of contract code to copy beyond.
     * @param _length Total number of bytes to copy from the contract's code.
     * @return _code Bytes of code copied from the requested contract.
     */
    function ovmEXTCODECOPY(
        address _contract,
        uint256 _offset,
        uint256 _length
    )
        override
        public
        returns (
            bytes memory _code
        )
    {
        // `ovmEXTCODECOPY` is the only overridden opcode capable of producing exactly one byte of
        // return data. By blocking reads of one byte, we're able to use the condition that an
        // OVM_ExecutionManager function return value having a length of exactly one byte indicates
        // an error without an explicit revert. If users were able to read a single byte, they
        // could forcibly trigger behavior that should only be available to this contract.
        uint256 length = _length == 1 ? 2 : _length;

        return Lib_EthUtils.getCode(
            _getAccount(_contract).ethAddress,
            _offset,
            _length
        );
    }

    /**
     * @notice Overrides EXTCODESIZE.
     * @param _contract Address of the contract to query the size of.
     * @return _EXTCODESIZE Size of the requested contract in bytes.
     */
    function ovmEXTCODESIZE(
        address _contract
    )
        override
        public
        returns (
            uint256 _EXTCODESIZE
        )
    {
        return Lib_EthUtils.getCodeSize(
            _getAccount(_contract).ethAddress
        );
    }

    /**
     * @notice Overrides EXTCODEHASH.
     * @param _contract Address of the contract to query the hash of.
     * @return _EXTCODEHASH Size of the requested contract in bytes.
     */
    function ovmEXTCODEHASH(
        address _contract
    )
        override
        public
        returns (
            bytes32 _EXTCODEHASH
        )
    {
        return Lib_EthUtils.getCodeHash(
            _getAccount(_contract).ethAddress
        );
    }


    /**************************************
     * Public Functions: Execution Safety *
     **************************************/


    /**
     * Performs the logic to create a contract and revert under various potential conditions.
     * @dev This function is implemented as `public` because we need to be able to revert a
     *      contract creation without losing information about exactly *why* the contract reverted.
     *      In particular, we want to be sure that contracts cannot trigger an INVALID_STATE_ACCESS
     *      flag and then revert to reset the flag. We're able to do this by making an external
     *      call from `ovmCREATE` and `ovmCREATE2` to `safeCREATE`, which can capture and relay
     *      information before reverting.
     * @param _address Address of the contract to associate with the one being created.
     * @param _bytecode Code to be used to create the new contract.
     */
    function safeCREATE(
        address _address,
        bytes memory _bytecode
    )
        override
        public
    {
        // Since this function is public, anyone can attempt to directly call it. We need to make
        // sure that the OVM_ExecutionManager itself is the only party that can actually try to
        // call this function.
        if (msg.sender != address(this)) {
            return;
        }

        // Check the creation bytecode against the OVM_SafetyChecker.
        if (ovmSafetyChecker.isBytecodeSafe(_bytecode) == false) {
            _revertWithFlag(RevertFlag.UNSAFE_BYTECODE);
        }

        // We always need to initialize the contract with the default account values.
        _initPendingAccount(_address);

        // Actually deploy the contract and retrieve its address. This step is hiding a lot of
        // complexity because we need to ensure that contract creation *never* reverts by itself.
        // We cover this partially by storing a revert flag and returning (instead of reverting)
        // when we know that we're inside a contract's creation code.
        address ethAddress = Lib_EthUtils.createContract(_bytecode);

        // Here we pull out the revert flag that would've been set during creation code. Now that
        // we're out of creation code again, we can just revert normally while passing the flag
        // through the revert data.
        if (messageRecord.revertFlag != RevertFlag.DID_NOT_REVERT) {
            _revertWithFlag(messageRecord.revertFlag);
        }

        // Again simply checking that the deployed code is safe too. Contracts can generate
        // arbitrary deployment code, so there's no easy way to analyze this beforehand.
        bytes memory deployedCode = Lib_EthUtils.getCode(ethAddress);
        if (ovmSafetyChecker.isBytecodeSafe(deployedCode) == false) {
            _revertWithFlag(RevertFlag.UNSAFE_BYTECODE);
        }

        // Contract creation didn't need to be reverted and the bytecode is safe. We finish up by
        // associating the desired address with the newly created contract's code hash and address.
        _commitPendingAccount(
            _address,
            ethAddress,
            keccak256(deployedCode)
        );
    }


    /********************************************
     * Internal Functions: Contract Interaction *
     ********************************************/

    /**
     * Creates a new contract and associates it with some contract address.
     * @param _contractAddress Address to associate the created contract with.
     * @param _bytecode Bytecode to be used to create the contract.
     */
    function _createContract(
        address _contractAddress,
        bytes memory _bytecode
    )
        internal
    {
        // We always update the nonce of the creating account, even if the creation fails.
        _incrementAccountNonce(ovmADDRESS());

        // We're stepping into a CREATE or CREATE2, so we need to update ADDRESS to point
        // to the contract's associated address.
        MessageContext memory nextMessageContext = messageContext;
        nextMessageContext.ovmADDRESS = _contractAddress;

        // Run `safeCREATE` in a new EVM message so that our changes can be reflected even if
        // `safeCREATE` reverts.
        _handleExternalInteraction(
            nextMessageContext,
            gasleft(),
            address(this),
            abi.encodeWithSignature(
                "safeCREATE(address,bytes)",
                _contractAddress,
                _bytecode
            )
        );

        // Need to make sure that this flag is reset so that it isn't propagated to creations in
        // some parent EVM message.
        messageRecord.revertFlag = RevertFlag.DID_NOT_REVERT;
    }

    /**
     * Calls the deployed contract associated with a given address.
     * @param _nextMessageContext Message context to be used for the call.
     * @param _gasLimit Amount of gas to be passed into this call.
     * @param _contract Address used to resolve the deployed contract.
     * @param _calldata Data to send along with the call.
     * @return _success Whether or not the call returned (rather than reverted).
     * @return _returndata Data returned by the call.
     */
    function _callContract(
        MessageContext memory _nextMessageContext,
        uint256 _gasLimit,
        address _contract,
        bytes memory _calldata
    )
        internal
        returns (
            bool _success,
            bytes memory _returndata
        )
    {
        return _handleExternalInteraction(
            _nextMessageContext,
            _gasLimit,
            _getAccount(_contract).ethAddress,
            _calldata
        );
    }

    /**
     * Handles the logic of making an external call and parsing revert information.
     * @param _nextMessageContext Message context to be used for the call.
     * @param _gasLimit Amount of gas to be passed into this call.
     * @param _target Address of the contract to call.
     * @param _data Data to send along with the call.
     * @return _success Whether or not the call returned (rather than reverted).
     * @return _returndata Data returned by the call.
     */
    function _handleExternalInteraction(
        MessageContext memory _nextMessageContext,
        uint256 _gasLimit,
        address _target,
        bytes memory _data
    )
        internal
        returns (
            bool _success,
            bytes memory _returndata
        )
    {
        // We need to switch over to our next message context for the duration of this call.
        MessageContext memory prevMessageContext = messageContext;
        _switchMessageContext(prevMessageContext, _nextMessageContext);

        // Nuisance gas is a system used to bound the ability for an attacker to make fraud proofs
        // expensive by touching a lot of different accounts or storage slots. Since most contracts
        // only use a few storage slots during any given transaction, this shouldn't be a limiting
        // factor.
        uint256 prevNuisanceGasLeft = messageRecord.nuisanceGasLeft;
        uint256 nuisanceGasLimit = _getNuisanceGasLimit(_gasLimit);
        messageRecord.nuisanceGasLeft = nuisanceGasLimit;

        // Make the call and make sure to pass in the gas limit. Another instance of hidden
        // complexity. `_target` is guaranteed to be a safe contract, meaning its return/revert
        // behavior can be controlled. In particular, we enforce that flags are passed through
        // revert data as to retrieve execution metadata that would normally be reverted out of
        // existence.
        (bool success, bytes memory returndata) = _target.call{gas: _gasLimit}(_data);

        // Reverts at this point are completely OK, but we need to make a few updates based on the
        // information passed through the revert.
        if (success == false) {
            (
                RevertFlag flag,
                uint256 nuisanceGasLeft,
                uint256 ovmGasRefund,
            ) = _decodeRevertData(returndata);

            // INVALID_STATE_ACCESS is the only flag that triggers an immediate abort of the
            // parent EVM message. This behavior is necessary because INVALID_STATE_ACCESS must
            // halt any further transaction execution that could impact the execution result.
            if (flag == RevertFlag.INVALID_STATE_ACCESS) {
                _revertWithFlag(flag);
            }

            // INTENTIONAL_REVERT and UNSAFE_BYTECODE aren't dependent on the input state, so we
            // can just handle them like standard reverts. Our only change here is to record the
            // gas refund reported by the call (enforced by safety checking).
            if (
                flag == RevertFlag.INTENTIONAL_REVERT
                || flag == RevertFlag.UNSAFE_BYTECODE
            ) {
                transactionRecord.ovmGasRefund = ovmGasRefund;
            }

            // Reverts mean we need to use up whatever "nuisance gas" was used by the call.
            // EXCEEDS_NUISANCE_GAS explicitly reduces the remaining nuisance gas for this message
            // to zero. OUT_OF_GAS is a "pseudo" flag given that messages return no data when they
            // run out of gas, so we have to treat this like EXCEEDS_NUISANCE_GAS. All other flags
            // will simply pass up the remaining nuisance gas.
            messageRecord.nuisanceGasLeft = prevNuisanceGasLeft - (nuisanceGasLimit - nuisanceGasLeft);
        }

        // Switch back to the original message context now that we're out of the call.
        _switchMessageContext(_nextMessageContext, prevMessageContext);

        return (
            success,
            returndata
        );
    }

    
    /******************************************
     * Internal Functions: State Manipulation *
     ******************************************/

    /**
     * Retrieves an account from the OVM_StateManager.
     * @param _address Address of the account to retrieve.
     * @return _account Retrieved account object.
     */
    function _getAccount(
        address _address
    )
        internal
        returns (
            iOVM_DataTypes.OVMAccount memory _account
        )
    {
        // We need to make sure that the transaction isn't trying to access an account that hasn't
        // been provided to the OVM_StateManager. We'll immediately abort if this is the case.
        _checkInvalidStateAccess(
            ovmStateManager.hasAccount(_address)
        );

        // Check whether the account has been loaded before and mark it as loaded if not. We need
        // this because "nuisance gas" only applies to the first time that an account is loaded.
        (
            bool _wasAccountAlreadyLoaded
        ) = ovmStateManager.testAndSetAccountLoaded(_address);

        // Actually retrieve the account.
        iOVM_DataTypes.OVMAccount memory account = ovmStateManager.getAccount(_address);

        // If we hadn't already loaded the account, then we'll need to charge "nuisance gas" based
        // on the size of the contract code.
        if (_wasAccountAlreadyLoaded == false) {
            _useNuisanceGas(
                Lib_EthUtils.getCodeSize(account.ethAddress) * NUISANCE_GAS_PER_CONTRACT_BYTE
            );
        }

        return account;
    }

    /**
     * Increments the nonce of an account.
     * @param _address Address of the account to bump.
     */
    function _incrementAccountNonce(
        address _address
    )
        internal
    {
        ovmStateManager.incrementAccountNonce(_address);
    }

    /**
     * Creates the default account object for the given address.
     * @param _address Address of the account create.
     */
    function _initPendingAccount(
        address _address
    )
        internal
    {
        ovmStateManager.initPendingAccount(_address);
    }

    /**
     * Stores additional relevant data for a new account, thereby "committing" it to the state.
     * This function is only called during `ovmCREATE` and `ovmCREATE2` after a successful contract
     * creation.
     * @param _address Address of the account to commit.
     * @param _ethAddress Address of the associated deployed contract.
     * @param _codeHash Hash of the code stored at the address.
     */
    function _commitPendingAccount(
        address _address,
        address _ethAddress,
        bytes32 _codeHash
    )
        internal
    {
        // Check whether the account has been changed before and mark it as changed if not. We need
        // this because "nuisance gas" only applies to the first time that an account is changed.
        (
            bool _wasAccountAlreadyChanged
        ) = ovmStateManager.testAndSetAccountChanged(_address);

        // If we hadn't already changed the account, then we'll need to charge "nuisance gas" based
        // on the size of the contract code.
        if (_wasAccountAlreadyChanged == false) {
            _useNuisanceGas(
                Lib_EthUtils.getCodeSize(_ethAddress) * NUISANCE_GAS_PER_CONTRACT_BYTE
            );
        }

        // Actually commit the contract.
        ovmStateManager.commitPendingAccount(
            _address,
            _ethAddress,
            _codeHash
        );
    }

    /**
     * Retrieves the value of a storage slot.
     * @param _contract Address of the contract to query.
     * @param _key 32 byte key of the storage slot.
     * @return _value 32 byte storage slot value.
     */
    function _getContractStorage(
        address _contract,
        bytes32 _key
    )
        internal
        returns (
            bytes32 _value
        )
    {
        // We need to make sure that the transaction isn't trying to access storage that hasn't
        // been provided to the OVM_StateManager. We'll immediately abort if this is the case.
        _checkInvalidStateAccess(
            ovmStateManager.hasContractStorage(_contract, _key)
        );

        // Check whether the slot has been loaded before and mark it as loaded if not. We need
        // this because "nuisance gas" only applies to the first time that a slot is loaded.
        (
            bool _wasContractStorageAlreadyLoaded
        ) = ovmStateManager.testAndSetContractStorageLoaded(_contract, _key);

        // If we hadn't already loaded the account, then we'll need to charge some fixed amount of
        // "nuisance gas".
        if (_wasContractStorageAlreadyLoaded == false) {
            _useNuisanceGas(NUISANCE_GAS_SLOAD);
        }

        // Actually retrieve the storage slot.
        return ovmStateManager.getContractStorage(_contract, _key);
    }
  
    /**
     * Sets the value of a storage slot.
     * @param _contract Address of the contract to modify.
     * @param _key 32 byte key of the storage slot.
     * @param _value 32 byte storage slot value.
     */
    function _putContractStorage(
        address _contract,
        bytes32 _key,
        bytes32 _value
    )
        internal
    {
        // Check whether the slot has been changed before and mark it as changed if not. We need
        // this because "nuisance gas" only applies to the first time that a slot is changed.
        (
            bool _wasContractStorageAlreadyChanged
        ) = ovmStateManager.testAndSetContractStorageChanged(_contract, _key);

        // If we hadn't already changed the account, then we'll need to charge some fixed amount of
        // "nuisance gas".
        if (_wasContractStorageAlreadyChanged == false) {
            _useNuisanceGas(NUISANCE_GAS_SSTORE);
        }

        // Actually modify the storage slot.
        ovmStateManager.putContractStorage(_contract, _key, _value);
    }


    /************************************
     * Internal Functions: Revert Logic *
     ************************************/

    /**
     * Simple encoding for revert data.
     * @param _flag Flag to revert with.
     * @param _data Additional user-provided revert data.
     * @return _revertdata Encoded revert data.
     */
    function _encodeRevertData(
        RevertFlag _flag,
        bytes memory _data
    )
        internal
        returns (
            bytes memory _revertdata
        )
    {
        // Running out of gas will return no data, so simulating it shouldn't either.
        if (_flag == RevertFlag.OUT_OF_GAS) {
            return bytes('');
        }

        // Just ABI encode the rest of the parameters.
        return abi.encode(
            _flag,
            messageRecord.nuisanceGasLeft,
            transactionRecord.ovmGasRefund,
            _data
        );
    }

    /**
     * Simple decoding for revert data.
     * @param _revertdata Revert data to decode.
     * @return _flag Flag used to revert.
     * @return _nuisanceGasLeft Amount of nuisance gas unused by the message.
     * @return _ovmGasRefund Amount of gas refunded during the message.
     * @return _data Additional user-provided revert data.
     */
    function _decodeRevertData(
        bytes memory _revertdata
    )
        internal
        returns (
            RevertFlag _flag,
            uint256 _nuisanceGasLeft,
            uint256 _ovmGasRefund,
            bytes memory _data
        )
    {
        // A length of zero means the call ran out of gas, just return empty data.
        if (_revertdata.length == 0) {
            return (
                RevertFlag.OUT_OF_GAS,
                0,
                0,
                bytes('')
            );
        }

        // ABI decode the incoming data.
        return abi.decode(_revertdata, (RevertFlag, uint256, uint256, bytes));
    }

    /**
     * Causes a message to revert or abort.
     * @param _flag Flag to revert with.
     * @param _data Additional user-provided data.
     */
    function _revertWithFlag(
        RevertFlag _flag,
        bytes memory _data
    )
        internal
    {
        // We don't want to revert when we're inside a CREATE or CREATE2, because those opcodes
        // fail silently (we can't pass any data upwards). Instead, we set a flag and return a
        // *single* byte, something the OVM_ExecutionManager will not return in any other case.
        // We're thereby allowed to communicate failure without allowing contracts to trick us into
        // thinking there was a failure.
        if (_inCreationContext()) {
            messageRecord.revertFlag = _flag;

            assembly {
                return(0, 1)
            }
        }

        // If we're not inside a CREATE or CREATE2, we can simply encode the necessary data and
        // revert normally.
        bytes memory revertdata = _encodeRevertData(
            _flag,
            _data
        );

        assembly {
            revert(add(revertdata, 0x20), mload(revertdata))
        } 
    }

    /**
     * Causes a message to revert or abort.
     * @param _flag Flag to revert with.
     */
    function _revertWithFlag(
        RevertFlag _flag
    )
        internal
    {
        _revertWithFlag(_flag, bytes(''));
    }

    /**
     * Checks for an attempt to access some inaccessible state.
     * @param _condition Result of some function that checks for bad access.
     */
    function _checkInvalidStateAccess(
        bool _condition
    )
        internal
    {
        // Another case of hidden complexity. If we didn't enforce this requirement, then a
        // contract could pass in just enough gas to cause this to fail on L1 but not on L2.
        // A contract could use this behavior to prevent the OVM_ExecutionManager from detecting
        // an invalid state access. Reverting with OUT_OF_GAS allows us to also charge for the
        // full message nuisance gas as to generally disincentivize this attack.
        if (gasleft() < MIN_GAS_FOR_INVALID_STATE_ACCESS) {
            _revertWithFlag(RevertFlag.OUT_OF_GAS);
        }

        // We have enough gas to comfortably run this revert, so do it.
        if (_condition == false) {
            _revertWithFlag(RevertFlag.INVALID_STATE_ACCESS);
        }
    }


    /******************************************
     * Internal Functions: Nuisance Gas Logic *
     ******************************************/

    /**
     * Computes the nuisance gas limit from the gas limit.
     * @dev This function is currently using a naive implementation whereby the nuisance gas limit
     *      is set to exactly equal the lesser of the gas limit or remaining gas. It's likely that
     *      this implementation is perfectly fine, but we may change this formula later.
     * @param _gasLimit Gas limit to compute from.
     * @return _nuisanceGasLimit Computed nuisance gas limit.
     */
    function _getNuisanceGasLimit(
        uint256 _gasLimit
    )
        internal
        view
        returns (
            uint256 _nuisanceGasLimit
        )
    {
        return _gasLimit < gasleft() ? _gasLimit : gasleft();
    }

    /**
     * Uses a certain amount of nuisance gas.
     * @param _amount Amount of nuisance gas to use.
     */
    function _useNuisanceGas(
        uint256 _amount
    )
        internal
    {
        // Essentially the same as a standard OUT_OF_GAS, except we also retain a record of the gas
        // refund to be given at the end of the transaction.
        if (messageRecord.nuisanceGasLeft < _amount) {
            _revertWithFlag(RevertFlag.EXCEEDS_NUISANCE_GAS);
        }

        messageRecord.nuisanceGasLeft -= _amount;
    }


    /*****************************************
     * Internal Functions: Execution Context *
     *****************************************/

    /**
     * Swaps over to a new message context.
     * @param _prevMessageContext Context we're switching from.
     * @param _nextMessageContext Context we're switching to.
     */
    function _switchMessageContext(
        MessageContext memory _prevMessageContext,
        MessageContext memory _nextMessageContext
    )
        internal
    {
        // Avoid unnecessary the SSTORE.
        if (_prevMessageContext.ovmCALLER != _nextMessageContext.ovmCALLER) {
            messageContext.ovmCALLER = _nextMessageContext.ovmCALLER;
        }

        // Avoid unnecessary the SSTORE.
        if (_prevMessageContext.isStatic != _nextMessageContext.isStatic) {
            messageContext.isStatic = _nextMessageContext.isStatic;
        }
    }

    /**
     * Checks whether we're inside contract creation code.
     * @return _inCreation Whether or not we're in a contract creation.
     */
    function _inCreationContext()
        internal
        returns (
            bool _inCreation
        )
    {
        // An interesting "hack" of sorts. Since the contract doesn't exist yet, it won't have any
        // stored contract code. A simple-but-elegant way to detect this condition.
        return (
            ovmADDRESS() != address(0)
            && ovmEXTCODESIZE(ovmADDRESS()) == 0
        );
    }
}
