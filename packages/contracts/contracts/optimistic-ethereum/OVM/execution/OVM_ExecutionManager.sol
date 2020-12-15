// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_EthUtils } from "../../libraries/utils/Lib_EthUtils.sol";

/* Interface Imports */
import { iOVM_ExecutionManager } from "../../iOVM/execution/iOVM_ExecutionManager.sol";
import { iOVM_StateManager } from "../../iOVM/execution/iOVM_StateManager.sol";
import { iOVM_SafetyChecker } from "../../iOVM/execution/iOVM_SafetyChecker.sol";

/* Contract Imports */
import { OVM_ECDSAContractAccount } from "../accounts/OVM_ECDSAContractAccount.sol";
import { OVM_ProxyEOA } from "../accounts/OVM_ProxyEOA.sol";

/**
 * @title OVM_ExecutionManager
 */
contract OVM_ExecutionManager is iOVM_ExecutionManager, Lib_AddressResolver {

    /********************************
     * External Contract References *
     ********************************/

    iOVM_SafetyChecker internal ovmSafetyChecker;
    iOVM_StateManager internal ovmStateManager;


    /*******************************
     * Execution Context Variables *
     *******************************/

    GasMeterConfig internal gasMeterConfig;
    GlobalContext internal globalContext;
    TransactionContext internal transactionContext;
    MessageContext internal messageContext;
    TransactionRecord internal transactionRecord;
    MessageRecord internal messageRecord;


    /**************************
     * Gas Metering Constants *
     **************************/

    address constant GAS_METADATA_ADDRESS = 0x06a506A506a506A506a506a506A506A506A506A5;
    uint256 constant NUISANCE_GAS_SLOAD = 20000;
    uint256 constant NUISANCE_GAS_SSTORE = 20000;
    uint256 constant MIN_NUISANCE_GAS_PER_CONTRACT = 30000;
    uint256 constant NUISANCE_GAS_PER_CONTRACT_BYTE = 100;
    uint256 constant MIN_GAS_FOR_INVALID_STATE_ACCESS = 30000;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _libAddressManager Address of the Address Manager.
     */
    constructor(
        address _libAddressManager,
        GasMeterConfig memory _gasMeterConfig,
        GlobalContext memory _globalContext
    )
        Lib_AddressResolver(_libAddressManager)
    {
        ovmSafetyChecker = iOVM_SafetyChecker(resolve("OVM_SafetyChecker"));
        gasMeterConfig = _gasMeterConfig;
        globalContext = _globalContext;
    }


    /**********************
     * Function Modifiers *
     **********************/

    /**
     * Applies dynamically-sized refund to a transaction to account for the difference in execution
     * between L1 and L2, so that the overall cost of the ovmOPCODE is fixed.
     * @param _cost Desired gas cost for the function after the refund.
     */
    modifier netGasCost(
        uint256 _cost
    ) {
        uint256 gasProvided = gasleft();
        _;
        uint256 gasUsed = gasProvided - gasleft();

        // We want to refund everything *except* the specified cost.
        if (_cost < gasUsed) {
            transactionRecord.ovmGasRefund += gasUsed - _cost;
        }
    }

    /**
     * Applies a fixed-size gas refund to a transaction to account for the difference in execution
     * between L1 and L2, so that the overall cost of an ovmOPCODE can be lowered.
     * @param _discount Amount of gas cost to refund for the ovmOPCODE.
     */
    modifier fixedGasDiscount(
        uint256 _discount
    ) {
        uint256 gasProvided = gasleft();
        _;
        uint256 gasUsed = gasProvided - gasleft();

        // We want to refund the specified _discount, unless this risks underflow.
        if (_discount < gasUsed) {
            transactionRecord.ovmGasRefund += _discount;
        } else {
            // refund all we can without risking underflow.
            transactionRecord.ovmGasRefund += gasUsed;
        }
    }

    /**
     * Makes sure we're not inside a static context.
     */
    modifier notStatic() {
        if (messageContext.isStatic == true) {
            _revertWithFlag(RevertFlag.STATIC_VIOLATION);
        }
        _;
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
        Lib_OVMCodec.Transaction memory _transaction,
        address _ovmStateManager
    )
        override
        public
    {
        // Store our OVM_StateManager instance (significantly easier than attempting to pass the
        // address around in calldata).
        ovmStateManager = iOVM_StateManager(_ovmStateManager);

        // Make sure this function can't be called by anyone except the owner of the
        // OVM_StateManager (expected to be an OVM_StateTransitioner). We can revert here because
        // this would make the `run` itself invalid.
        require(
            ovmStateManager.isAuthenticated(msg.sender),
            "Only authenticated addresses in ovmStateManager can call this function"
        );

        // Initialize the execution context, must be initialized before we perform any gas metering
        // or we'll throw a nuisance gas error.
        _initContext(_transaction);

        // TEMPORARY: Gas metering is disabled for minnet.
        // // Check whether we need to start a new epoch, do so if necessary.
        // _checkNeedsNewEpoch(_transaction.timestamp);

        // Make sure the transaction's gas limit is valid. We don't revert here because we reserve
        // reverts for INVALID_STATE_ACCESS.
        if (_isValidGasLimit(_transaction.gasLimit, _transaction.l1QueueOrigin) == false) {
            return;
        }

        // Run the transaction, make sure to meter the gas usage.
        uint256 gasProvided = gasleft();
        ovmCALL(
            _transaction.gasLimit - gasMeterConfig.minTransactionGasLimit,
            _transaction.entrypoint,
            _transaction.data
        );
        uint256 gasUsed = gasProvided - gasleft();

        // TEMPORARY: Gas metering is disabled for minnet.
        // // Update the cumulative gas based on the amount of gas used.
        // _updateCumulativeGas(gasUsed, _transaction.l1QueueOrigin);

        // Wipe the execution context.
        _resetContext();

        // Reset the ovmStateManager.
        ovmStateManager = iOVM_StateManager(address(0));
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
        view
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
        view
        returns (
            address _ADDRESS
        )
    {
        return messageContext.ovmADDRESS;
    }

    /**
     * @notice Overrides TIMESTAMP.
     * @return _TIMESTAMP Value of the TIMESTAMP within the transaction context.
     */
    function ovmTIMESTAMP()
        override
        public
        view
        returns (
            uint256 _TIMESTAMP
        )
    {
        return transactionContext.ovmTIMESTAMP;
    }

    /**
     * @notice Overrides NUMBER.
     * @return _NUMBER Value of the NUMBER within the transaction context.
     */
    function ovmNUMBER()
        override
        public
        view
        returns (
            uint256 _NUMBER
        )
    {
        return transactionContext.ovmNUMBER;
    }

    /**
     * @notice Overrides GASLIMIT.
     * @return _GASLIMIT Value of the block's GASLIMIT within the transaction context.
     */
    function ovmGASLIMIT()
        override
        public
        view
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
        view
        returns (
            uint256 _CHAINID
        )
    {
        return globalContext.ovmCHAINID;
    }

    /*********************************
     * Opcodes: L2 Execution Context *
     *********************************/

    /**
     * @notice Specifies from which L1 rollup queue this transaction originated from.
     * @return _queueOrigin Address of the ovmL1QUEUEORIGIN within the current message context.
     */
    function ovmL1QUEUEORIGIN()
        override
        public
        view
        returns (
            Lib_OVMCodec.QueueOrigin _queueOrigin
        )
    {
        return transactionContext.ovmL1QUEUEORIGIN;
    }

    /**
     * @notice Specifies what L1 EOA, if any, sent this transaction.
     * @return _l1TxOrigin Address of the EOA which send the tx into L2 from L1.
     */
    function ovmL1TXORIGIN()
        override
        public
        view
        returns (
            address _l1TxOrigin
        )
    {
        return transactionContext.ovmL1TXORIGIN;
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
        notStatic
        fixedGasDiscount(40000)
        returns (
            address _contract
        )
    {
        // Creator is always the current ADDRESS.
        address creator = ovmADDRESS();

        // Generate the correct CREATE address.
        address contractAddress = Lib_EthUtils.getAddressForCREATE(
            creator,
            _getAccountNonce(creator)
        );

        return _createContract(
            contractAddress,
            _bytecode
        );
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
        notStatic
        fixedGasDiscount(40000)
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

        return _createContract(
            contractAddress,
            _bytecode
        );
    }


    /*******************************
     * Account Abstraction Opcodes *
     ******************************/

    /**
     * Retrieves the nonce of the current ovmADDRESS.
     * @return _nonce Nonce of the current contract.
     */
    function ovmGETNONCE()
        override
        public
        returns (
            uint256 _nonce
        )
    {
        return _getAccountNonce(ovmADDRESS());
    }

    /**
     * Sets the nonce of the current ovmADDRESS.
     * @param _nonce New nonce for the current contract.
     */
    function ovmSETNONCE(
        uint256 _nonce
    )
        override
        public
    {
        _setAccountNonce(ovmADDRESS(), _nonce);
    }

    /**
     * Creates a new EOA contract account, for account abstraction.
     * @dev Essentially functions like ovmCREATE or ovmCREATE2, but we can bypass a lot of checks
     *      because the contract we're creating is trusted (no need to do safety checking or to
     *      handle unexpected reverts). Doesn't need to return an address because the address is
     *      assumed to be the user's actual address.
     * @param _messageHash Hash of a message signed by some user, for verification.
     * @param _v Signature `v` parameter.
     * @param _r Signature `r` parameter.
     * @param _s Signature `s` parameter.
     */
    function ovmCREATEEOA(
        bytes32 _messageHash,
        uint8 _v,
        bytes32 _r,
        bytes32 _s
    )
        override
        public
        notStatic
    {
        // Recover the EOA address from the message hash and signature parameters. Since we do the
        // hashing in advance, we don't have handle different message hashing schemes. Even if this
        // function were to return the wrong address (rather than explicitly returning the zero
        // address), the rest of the transaction would simply fail (since there's no EOA account to
        // actually execute the transaction).
        address eoa = ecrecover(
            _messageHash,
            _v + 27,
            _r,
            _s
        );

        // Invalid signature is a case we proactively handle with a revert. We could alternatively
        // have this function return a `success` boolean, but this is just easier.
        if (eoa == address(0)) {
            ovmREVERT(bytes("Signature provided for EOA contract creation is invalid."));
        }

        // If the user already has an EOA account, then there's no need to perform this operation.
        if (_hasEmptyAccount(eoa) == false) {
            return;
        }

        // We always need to initialize the contract with the default account values.
        _initPendingAccount(eoa);

        // Temporarily set the current address so it's easier to access on L2.
        address prevADDRESS = messageContext.ovmADDRESS;
        messageContext.ovmADDRESS = eoa;

        // Now actually create the account and get its bytecode. We're not worried about reverts
        // (other than out of gas, which we can't capture anyway) because this contract is trusted.
        OVM_ProxyEOA proxyEOA = new OVM_ProxyEOA(0x4200000000000000000000000000000000000003);

        // Reset the address now that we're done deploying.
        messageContext.ovmADDRESS = prevADDRESS;

        // Commit the account with its final values.
        _commitPendingAccount(
            eoa,
            address(proxyEOA),
            keccak256(Lib_EthUtils.getCode(address(proxyEOA)))
        );

        _setAccountNonce(eoa, 0);
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
        fixedGasDiscount(100000)
        returns (
            bool _success,
            bytes memory _returndata
        )
    {
        // CALL updates the CALLER and ADDRESS.
        MessageContext memory nextMessageContext = messageContext;
        nextMessageContext.ovmCALLER = nextMessageContext.ovmADDRESS;
        nextMessageContext.ovmADDRESS = _address;
        bool isStaticEntrypoint = false;

        return _callContract(
            nextMessageContext,
            _gasLimit,
            _address,
            _calldata,
            isStaticEntrypoint
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
        fixedGasDiscount(80000)
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
        bool isStaticEntrypoint = true;

        return _callContract(
            nextMessageContext,
            _gasLimit,
            _address,
            _calldata,
            isStaticEntrypoint
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
        fixedGasDiscount(40000)
        returns (
            bool _success,
            bytes memory _returndata
        )
    {
        // DELEGATECALL does not change anything about the message context.
        MessageContext memory nextMessageContext = messageContext;
        bool isStaticEntrypoint = false;

        return _callContract(
            nextMessageContext,
            _gasLimit,
            _address,
            _calldata,
            isStaticEntrypoint
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
        notStatic
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
            _getAccountEthAddress(_contract),
            _offset,
            length
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
            _getAccountEthAddress(_contract)
        );
    }

    /**
     * @notice Overrides EXTCODEHASH.
     * @param _contract Address of the contract to query the hash of.
     * @return _EXTCODEHASH Hash of the requested contract.
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
            _getAccountEthAddress(_contract)
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

        // We need to be sure that the user isn't trying to use a contract creation to overwrite
        // some existing contract. On L1, users will prove that no contract exists at the address
        // and the OVM_FraudVerifier will populate the code hash of this address with a special
        // value that represents "known to be an empty account."
        if (_hasEmptyAccount(_address) == false) {
            _revertWithFlag(RevertFlag.CREATE_COLLISION);
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

        // Contract creation returns the zero address when it fails, which should only be possible
        // if the user intentionally runs out of gas. However, we might still have a bit of gas
        // left over since contract calls can only be passed 63/64ths of total gas,  so we need to
        // explicitly handle this case here.
        if (ethAddress == address(0)) {
            _revertWithFlag(RevertFlag.CREATE_EXCEPTION);
        }

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
            Lib_EthUtils.getCodeHash(ethAddress)
        );
    }


    /***************************************
     * Public Functions: Execution Context *
     ***************************************/

    function getMaxTransactionGasLimit()
        external
        view
        override
        returns (
            uint256 _maxTransactionGasLimit
        )
    {
        return gasMeterConfig.maxTransactionGasLimit;
    }



    /********************************************
     * Internal Functions: Contract Interaction *
     ********************************************/

    /**
     * Creates a new contract and associates it with some contract address.
     * @param _contractAddress Address to associate the created contract with.
     * @param _bytecode Bytecode to be used to create the contract.
     * @return _created Final OVM contract address.
     */
    function _createContract(
        address _contractAddress,
        bytes memory _bytecode
    )
        internal
        returns (
            address _created
        )
    {
        // We always update the nonce of the creating account, even if the creation fails.
        _setAccountNonce(ovmADDRESS(), _getAccountNonce(ovmADDRESS()) + 1);

        // We're stepping into a CREATE or CREATE2, so we need to update ADDRESS to point
        // to the contract's associated address and CALLER to point to the previous ADDRESS.
        MessageContext memory nextMessageContext = messageContext;
        nextMessageContext.ovmCALLER = messageContext.ovmADDRESS;
        nextMessageContext.ovmADDRESS = _contractAddress;

        // Run `safeCREATE` in a new EVM message so that our changes can be reflected even if
        // `safeCREATE` reverts.
        (bool _success, ) = _handleExternalInteraction(
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

        // Yellow paper requires that address returned is zero if the contract deployment fails.
        return _success ? _contractAddress : address(0);
    }

    /**
     * Calls the deployed contract associated with a given address.
     * @param _nextMessageContext Message context to be used for the call.
     * @param _gasLimit Amount of gas to be passed into this call.
     * @param _contract Address used to resolve the deployed contract.
     * @param _calldata Data to send along with the call.
     * @param _isStaticEntrypoint Whether or not this is coming from ovmSTATICCALL.
     * @return _success Whether or not the call returned (rather than reverted).
     * @return _returndata Data returned by the call.
     */
    function _callContract(
        MessageContext memory _nextMessageContext,
        uint256 _gasLimit,
        address _contract,
        bytes memory _calldata,
        bool _isStaticEntrypoint
    )
        internal
        returns (
            bool _success,
            bytes memory _returndata
        )
    {
        // EVM precompiles have the same address on L1 and L2 --> no trie lookup needed.
        address codeContractAddress =
            uint(_contract) < 100
            ? _contract
            : _getAccountEthAddress(_contract);

        return _handleExternalInteraction(
            _nextMessageContext,
            _gasLimit,
            codeContractAddress,
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

        // Switch back to the original message context now that we're out of the call.
        _switchMessageContext(_nextMessageContext, prevMessageContext);

        // Assuming there were no reverts, the message record should be accurate here. We'll update
        // this value in the case of a revert.
        uint256 nuisanceGasLeft = messageRecord.nuisanceGasLeft;

        // Reverts at this point are completely OK, but we need to make a few updates based on the
        // information passed through the revert.
        if (success == false) {
            (
                RevertFlag flag,
                uint256 nuisanceGasLeftPostRevert,
                uint256 ovmGasRefund,
                bytes memory returndataFromFlag
            ) = _decodeRevertData(returndata);

            // INVALID_STATE_ACCESS is the only flag that triggers an immediate abort of the
            // parent EVM message. This behavior is necessary because INVALID_STATE_ACCESS must
            // halt any further transaction execution that could impact the execution result.
            if (flag == RevertFlag.INVALID_STATE_ACCESS) {
                _revertWithFlag(flag);
            }

            // INTENTIONAL_REVERT, UNSAFE_BYTECODE, and STATIC_VIOLATION aren't dependent on the
            // input state, so we can just handle them like standard reverts. Our only change here
            // is to record the gas refund reported by the call (enforced by safety checking).
            if (
                flag == RevertFlag.INTENTIONAL_REVERT
                || flag == RevertFlag.UNSAFE_BYTECODE
                || flag == RevertFlag.STATIC_VIOLATION
            ) {
                transactionRecord.ovmGasRefund = ovmGasRefund;
            }

            // INTENTIONAL_REVERT needs to pass up the user-provided return data encoded into the
            // flag, *not* the full encoded flag. All other revert types return no data.
            if (flag == RevertFlag.INTENTIONAL_REVERT) {
                returndata = returndataFromFlag;
            } else {
                returndata = hex'';
            }

            // Reverts mean we need to use up whatever "nuisance gas" was used by the call.
            // EXCEEDS_NUISANCE_GAS explicitly reduces the remaining nuisance gas for this message
            // to zero. OUT_OF_GAS is a "pseudo" flag given that messages return no data when they
            // run out of gas, so we have to treat this like EXCEEDS_NUISANCE_GAS. All other flags
            // will simply pass up the remaining nuisance gas.
            nuisanceGasLeft = nuisanceGasLeftPostRevert;
        }

        // We need to reset the nuisance gas back to its original value minus the amount used here.
        messageRecord.nuisanceGasLeft = prevNuisanceGasLeft - (nuisanceGasLimit - nuisanceGasLeft);

        return (
            success,
            returndata
        );
    }


    /******************************************
     * Internal Functions: State Manipulation *
     ******************************************/

    /**
     * Checks whether an account exists within the OVM_StateManager.
     * @param _address Address of the account to check.
     * @return _exists Whether or not the account exists.
     */
    function _hasAccount(
        address _address
    )
        internal
        returns (
            bool _exists
        )
    {
        _checkAccountLoad(_address);
        return ovmStateManager.hasAccount(_address);
    }

    /**
     * Checks whether a known empty account exists within the OVM_StateManager.
     * @param _address Address of the account to check.
     * @return _exists Whether or not the account empty exists.
     */
    function _hasEmptyAccount(
        address _address
    )
        internal
        returns (
            bool _exists
        )
    {
        _checkAccountLoad(_address);
        return ovmStateManager.hasEmptyAccount(_address);
    }

    /**
     * Sets the nonce of an account.
     * @param _address Address of the account to modify.
     * @param _nonce New account nonce.
     */
    function _setAccountNonce(
        address _address,
        uint256 _nonce
    )
        internal
    {
        _checkAccountChange(_address);
        ovmStateManager.setAccountNonce(_address, _nonce);
    }

    /**
     * Gets the nonce of an account.
     * @param _address Address of the account to access.
     * @return _nonce Nonce of the account.
     */
    function _getAccountNonce(
        address _address
    )
        internal
        returns (
            uint256 _nonce
        )
    {
        _checkAccountLoad(_address);
        return ovmStateManager.getAccountNonce(_address);
    }

    /**
     * Retrieves the Ethereum address of an account.
     * @param _address Address of the account to access.
     * @return _ethAddress Corresponding Ethereum address.
     */
    function _getAccountEthAddress(
        address _address
    )
        internal
        returns (
            address _ethAddress
        )
    {
        _checkAccountLoad(_address);
        return ovmStateManager.getAccountEthAddress(_address);
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
        // Although it seems like `_checkAccountChange` would be more appropriate here, we don't
        // actually consider an account "changed" until it's inserted into the state (in this case
        // by `_commitPendingAccount`).
        _checkAccountLoad(_address);
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
        _checkAccountChange(_address);
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
        _checkContractStorageLoad(_contract, _key);
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
        // We don't set storage if the value didn't change. Although this acts as a convenient
        // optimization, it's also necessary to avoid the case in which a contract with no storage
        // attempts to store the value "0" at any key. Putting this value (and therefore requiring
        // that the value be committed into the storage trie after execution) would incorrectly
        // modify the storage root.
        if (_getContractStorage(_contract, _key) == _value) {
            return;
        }

        _checkContractStorageChange(_contract, _key);
        ovmStateManager.putContractStorage(_contract, _key, _value);
    }

    /**
     * Validation whenever a contract needs to be loaded. Checks that the account exists, charges
     * nuisance gas if the account hasn't been loaded before.
     * @param _address Address of the account to load.
     */
    function _checkAccountLoad(
        address _address
    )
        internal
    {
        // See `_checkContractStorageLoad` for more information.
        if (gasleft() < MIN_GAS_FOR_INVALID_STATE_ACCESS) {
            _revertWithFlag(RevertFlag.OUT_OF_GAS);
        }

        // See `_checkContractStorageLoad` for more information.
        if (ovmStateManager.hasAccount(_address) == false) {
            _revertWithFlag(RevertFlag.INVALID_STATE_ACCESS);
        }

        // Check whether the account has been loaded before and mark it as loaded if not. We need
        // this because "nuisance gas" only applies to the first time that an account is loaded.
        (
            bool _wasAccountAlreadyLoaded
        ) = ovmStateManager.testAndSetAccountLoaded(_address);

        // If we hadn't already loaded the account, then we'll need to charge "nuisance gas" based
        // on the size of the contract code.
        if (_wasAccountAlreadyLoaded == false) {
            _useNuisanceGas(
                (Lib_EthUtils.getCodeSize(_getAccountEthAddress(_address)) * NUISANCE_GAS_PER_CONTRACT_BYTE) + MIN_NUISANCE_GAS_PER_CONTRACT
            );
        }
    }

    /**
     * Validation whenever a contract needs to be changed. Checks that the account exists, charges
     * nuisance gas if the account hasn't been changed before.
     * @param _address Address of the account to change.
     */
    function _checkAccountChange(
        address _address
    )
        internal
    {
        // Start by checking for a load as we only want to charge nuisance gas proportional to
        // contract size once.
        _checkAccountLoad(_address);

        // Check whether the account has been changed before and mark it as changed if not. We need
        // this because "nuisance gas" only applies to the first time that an account is changed.
        (
            bool _wasAccountAlreadyChanged
        ) = ovmStateManager.testAndSetAccountChanged(_address);

        // If we hadn't already loaded the account, then we'll need to charge "nuisance gas" based
        // on the size of the contract code.
        if (_wasAccountAlreadyChanged == false) {
            ovmStateManager.incrementTotalUncommittedAccounts();
            _useNuisanceGas(
                (Lib_EthUtils.getCodeSize(_getAccountEthAddress(_address)) * NUISANCE_GAS_PER_CONTRACT_BYTE) + MIN_NUISANCE_GAS_PER_CONTRACT
            );
        }
    }

    /**
     * Validation whenever a slot needs to be loaded. Checks that the account exists, charges
     * nuisance gas if the slot hasn't been loaded before.
     * @param _contract Address of the account to load from.
     * @param _key 32 byte key to load.
     */
    function _checkContractStorageLoad(
        address _contract,
        bytes32 _key
    )
        internal
    {
        // Another case of hidden complexity. If we didn't enforce this requirement, then a
        // contract could pass in just enough gas to cause the INVALID_STATE_ACCESS check to fail
        // on L1 but not on L2. A contract could use this behavior to prevent the
        // OVM_ExecutionManager from detecting an invalid state access. Reverting with OUT_OF_GAS
        // allows us to also charge for the full message nuisance gas, because you deserve that for
        // trying to break the contract in this way.
        if (gasleft() < MIN_GAS_FOR_INVALID_STATE_ACCESS) {
            _revertWithFlag(RevertFlag.OUT_OF_GAS);
        }

        // We need to make sure that the transaction isn't trying to access storage that hasn't
        // been provided to the OVM_StateManager. We'll immediately abort if this is the case.
        // We know that we have enough gas to do this check because of the above test.
        if (ovmStateManager.hasContractStorage(_contract, _key) == false) {
            _revertWithFlag(RevertFlag.INVALID_STATE_ACCESS);
        }

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
    }

    /**
     * Validation whenever a slot needs to be changed. Checks that the account exists, charges
     * nuisance gas if the slot hasn't been changed before.
     * @param _contract Address of the account to change.
     * @param _key 32 byte key to change.
     */
    function _checkContractStorageChange(
        address _contract,
        bytes32 _key
    )
        internal
    {
        // Start by checking for load to make sure we have the storage slot and that we charge the
        // "nuisance gas" necessary to prove the storage slot state.
        _checkContractStorageLoad(_contract, _key);

        // Check whether the slot has been changed before and mark it as changed if not. We need
        // this because "nuisance gas" only applies to the first time that a slot is changed.
        (
            bool _wasContractStorageAlreadyChanged
        ) = ovmStateManager.testAndSetContractStorageChanged(_contract, _key);

        // If we hadn't already changed the account, then we'll need to charge some fixed amount of
        // "nuisance gas".
        if (_wasContractStorageAlreadyChanged == false) {
            // Changing a storage slot means that we're also going to have to change the
            // corresponding account, so do an account change check.
            _checkAccountChange(_contract);

            ovmStateManager.incrementTotalUncommittedContractStorage();
            _useNuisanceGas(NUISANCE_GAS_SSTORE);
        }
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
        view
        returns (
            bytes memory _revertdata
        )
    {
        // Out of gas and create exceptions will fundamentally return no data, so simulating it shouldn't either.
        if (
            _flag == RevertFlag.OUT_OF_GAS
            || _flag == RevertFlag.CREATE_EXCEPTION
        ) {
            return bytes('');
        }

        // INVALID_STATE_ACCESS doesn't need to return any data other than the flag.
        if (_flag == RevertFlag.INVALID_STATE_ACCESS) {
            return abi.encode(
                _flag,
                0,
                0,
                bytes('')
            );
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
        pure
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
        bool isCreation;
        assembly {
            isCreation := eq(extcodesize(caller()), 0)
        }

        if (isCreation) {
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


    /************************************
     * Internal Functions: Gas Metering *
     ************************************/

    /**
     * Checks whether a transaction needs to start a new epoch and does so if necessary.
     * @param _timestamp Transaction timestamp.
     */
    function _checkNeedsNewEpoch(
        uint256 _timestamp
    )
        internal
    {
        if (
            _timestamp >= (
                _getGasMetadata(GasMetadataKey.CURRENT_EPOCH_START_TIMESTAMP)
                + gasMeterConfig.secondsPerEpoch
            )
        ) {
            _putGasMetadata(
                GasMetadataKey.CURRENT_EPOCH_START_TIMESTAMP,
                _timestamp
            );

            _putGasMetadata(
                GasMetadataKey.PREV_EPOCH_SEQUENCER_QUEUE_GAS,
                _getGasMetadata(
                    GasMetadataKey.CUMULATIVE_SEQUENCER_QUEUE_GAS
                )
            );

            _putGasMetadata(
                GasMetadataKey.PREV_EPOCH_L1TOL2_QUEUE_GAS,
                _getGasMetadata(
                    GasMetadataKey.CUMULATIVE_L1TOL2_QUEUE_GAS
                )
            );
        }
    }

    /**
     * Validates the gas limit for a given transaction.
     * @param _gasLimit Gas limit provided by the transaction.
     * @param _queueOrigin Queue from which the transaction originated.
     * @return _valid Whether or not the gas limit is valid.
     */
    function _isValidGasLimit(
        uint256 _gasLimit,
        Lib_OVMCodec.QueueOrigin _queueOrigin
    )
        internal
        returns (
            bool _valid
        )
    {
        // Always have to be below the maximum gas limit.
        if (_gasLimit > gasMeterConfig.maxTransactionGasLimit) {
            return false;
        }

        // Always have to be above the minimum gas limit.
        if (_gasLimit < gasMeterConfig.minTransactionGasLimit) {
            return false;
        }

        // TEMPORARY: Gas metering is disabled for minnet.
        return true;
        // GasMetadataKey cumulativeGasKey;
        // GasMetadataKey prevEpochGasKey;
        // if (_queueOrigin == Lib_OVMCodec.QueueOrigin.SEQUENCER_QUEUE) {
        //     cumulativeGasKey = GasMetadataKey.CUMULATIVE_SEQUENCER_QUEUE_GAS;
        //     prevEpochGasKey = GasMetadataKey.PREV_EPOCH_SEQUENCER_QUEUE_GAS;
        // } else {
        //     cumulativeGasKey = GasMetadataKey.CUMULATIVE_L1TOL2_QUEUE_GAS;
        //     prevEpochGasKey = GasMetadataKey.PREV_EPOCH_L1TOL2_QUEUE_GAS;
        // }

        // return (
        //     (
        //         _getGasMetadata(cumulativeGasKey)
        //         - _getGasMetadata(prevEpochGasKey)
        //         + _gasLimit
        //     ) < gasMeterConfig.maxGasPerQueuePerEpoch
        // );
    }

    /**
     * Updates the cumulative gas after a transaction.
     * @param _gasUsed Gas used by the transaction.
     * @param _queueOrigin Queue from which the transaction originated.
     */
    function _updateCumulativeGas(
        uint256 _gasUsed,
        Lib_OVMCodec.QueueOrigin _queueOrigin
    )
        internal
    {
        GasMetadataKey cumulativeGasKey;
        if (_queueOrigin == Lib_OVMCodec.QueueOrigin.SEQUENCER_QUEUE) {
            cumulativeGasKey = GasMetadataKey.CUMULATIVE_SEQUENCER_QUEUE_GAS;
        } else {
            cumulativeGasKey = GasMetadataKey.CUMULATIVE_L1TOL2_QUEUE_GAS;
        }

        _putGasMetadata(
            cumulativeGasKey,
            (
                _getGasMetadata(cumulativeGasKey)
                + gasMeterConfig.minTransactionGasLimit
                + _gasUsed
                - transactionRecord.ovmGasRefund
            )
        );
    }

    /**
     * Retrieves the value of a gas metadata key.
     * @param _key Gas metadata key to retrieve.
     * @return _value Value stored at the given key.
     */
    function _getGasMetadata(
        GasMetadataKey _key
    )
        internal
        returns (
            uint256 _value
        )
    {
        return uint256(_getContractStorage(
            GAS_METADATA_ADDRESS,
            bytes32(uint256(_key))
        ));
    }

    /**
     * Sets the value of a gas metadata key.
     * @param _key Gas metadata key to set.
     * @param _value Value to store at the given key.
     */
    function _putGasMetadata(
        GasMetadataKey _key,
        uint256 _value
    )
        internal
    {
        _putContractStorage(
            GAS_METADATA_ADDRESS,
            bytes32(uint256(_key)),
            bytes32(uint256(_value))
        );
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
        if (_prevMessageContext.ovmADDRESS != _nextMessageContext.ovmADDRESS) {
            messageContext.ovmADDRESS = _nextMessageContext.ovmADDRESS;
        }

        // Avoid unnecessary the SSTORE.
        if (_prevMessageContext.isStatic != _nextMessageContext.isStatic) {
            messageContext.isStatic = _nextMessageContext.isStatic;
        }
    }

    /**
     * Initializes the execution context.
     * @param _transaction OVM transaction being executed.
     */
    function _initContext(
        Lib_OVMCodec.Transaction memory _transaction
    )
        internal
    {
        transactionContext.ovmTIMESTAMP = _transaction.timestamp;
        transactionContext.ovmNUMBER = _transaction.blockNumber;
        transactionContext.ovmTXGASLIMIT = _transaction.gasLimit;
        transactionContext.ovmL1QUEUEORIGIN = _transaction.l1QueueOrigin;
        transactionContext.ovmL1TXORIGIN = _transaction.l1TxOrigin;
        transactionContext.ovmGASLIMIT = gasMeterConfig.maxGasPerQueuePerEpoch;

        messageRecord.nuisanceGasLeft = _getNuisanceGasLimit(_transaction.gasLimit);
    }

    /**
     * Resets the transaction and message context.
     */
    function _resetContext()
        internal
    {
        transactionContext.ovmL1TXORIGIN = address(0);
        transactionContext.ovmTIMESTAMP = 0;
        transactionContext.ovmNUMBER = 0;
        transactionContext.ovmGASLIMIT = 0;
        transactionContext.ovmTXGASLIMIT = 0;
        transactionContext.ovmL1QUEUEORIGIN = Lib_OVMCodec.QueueOrigin.SEQUENCER_QUEUE;

        transactionRecord.ovmGasRefund = 0;

        messageContext.ovmCALLER = address(0);
        messageContext.ovmADDRESS = address(0);
        messageContext.isStatic = false;

        messageRecord.nuisanceGasLeft = 0;
        messageRecord.revertFlag = RevertFlag.DID_NOT_REVERT;
    }
}
