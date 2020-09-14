// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_DataTypes } from "../codec/iOVM_DataTypes.sol";

interface iOVM_ExecutionManager {
    enum RevertFlag {
        DID_NOT_REVERT,
        OUT_OF_GAS,
        INTENTIONAL_REVERT,
        EXCEEDS_NUISANCE_GAS,
        INVALID_STATE_ACCESS,
        UNSAFE_BYTECODE
    }

    struct GlobalContext {
        uint256 ovmCHAINID;
    }

    struct TransactionContext {
        address ovmORIGIN;
        uint256 ovmTIMESTAMP;
        uint256 ovmGASLIMIT;
        uint256 ovmTXGASLIMIT;
        uint256 ovmQUEUEORIGIN;
    }

    struct TransactionRecord {
        uint256 ovmGasRefund;
    }

    struct MessageContext {
        address ovmCALLER;
        address ovmADDRESS;
        bool isStatic;
    }

    struct MessageRecord {
        uint256 nuisanceGasLeft;
        RevertFlag revertFlag;
    }

    function run(
        iOVM_DataTypes.OVMTransactionData calldata _transaction,
        address _txStateManager
    ) external;


    /*******************
     * Context Opcodes *
     *******************/

    function ovmCALLER() external returns (address _caller);
    function ovmADDRESS() external returns (address _address);
    function ovmORIGIN() external returns (address _origin);
    function ovmTIMESTAMP() external returns (uint256 _timestamp);
    function ovmGASLIMIT() external returns (uint256 _gasLimit);
    function ovmCHAINID() external returns (uint256 _chainId);


    /*******************
     * Halting Opcodes *
     *******************/
    
    function ovmREVERT(bytes memory _data) external;


    /*****************************
     * Contract Creation Opcodes *
     *****************************/

    function ovmCREATE(bytes memory _bytecode) external returns (address _contract);
    function ovmCREATE2(bytes memory _bytecode, bytes32 _salt) external returns (address _contract);
    function safeCREATE(address _address, bytes memory _bytecode) external;


    /****************************
     * Contract Calling Opcodes *
     ****************************/

    function ovmCALL(uint256 _gasLimit, address _address, bytes memory _calldata) external returns (bool _success, bytes memory _returndata);
    function ovmSTATICCALL(uint256 _gasLimit, address _address, bytes memory _calldata) external returns (bool _success, bytes memory _returndata);
    function ovmDELEGATECALL(uint256 _gasLimit, address _address, bytes memory _calldata) external returns (bool _success, bytes memory _returndata);


    /****************************
     * Contract Storage Opcodes *
     ****************************/

    function ovmSLOAD(bytes32 _key) external returns (bytes32 _value);
    function ovmSSTORE(bytes32 _key, bytes32 _value) external;


    /*************************
     * Contract Code Opcodes *
     *************************/

    function ovmEXTCODECOPY(address _contract, uint256 _offset, uint256 _length) external returns (bytes memory _code);
    function ovmEXTCODESIZE(address _contract) external returns (uint256 _size);
    function ovmEXTCODEHASH(address _contract) external returns (bytes32 _hash);
}
