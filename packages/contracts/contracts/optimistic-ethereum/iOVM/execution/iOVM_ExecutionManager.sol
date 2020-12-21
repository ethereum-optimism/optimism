// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

interface iOVM_ExecutionManager {
    /**********
     * Enums *
     *********/

    enum RevertFlag {
        DID_NOT_REVERT,
        OUT_OF_GAS,
        INTENTIONAL_REVERT,
        EXCEEDS_NUISANCE_GAS,
        INVALID_STATE_ACCESS,
        UNSAFE_BYTECODE,
        CREATE_COLLISION,
        STATIC_VIOLATION,
        CREATE_EXCEPTION,
        CREATOR_NOT_ALLOWED
    }

    enum GasMetadataKey {
        CURRENT_EPOCH_START_TIMESTAMP,
        CUMULATIVE_SEQUENCER_QUEUE_GAS,
        CUMULATIVE_L1TOL2_QUEUE_GAS,
        PREV_EPOCH_SEQUENCER_QUEUE_GAS,
        PREV_EPOCH_L1TOL2_QUEUE_GAS
    }

    /***********
     * Structs *
     ***********/

    struct GasMeterConfig {
        uint256 minTransactionGasLimit;
        uint256 maxTransactionGasLimit;
        uint256 maxGasPerQueuePerEpoch;
        uint256 secondsPerEpoch;
    }

    struct GlobalContext {
        uint256 ovmCHAINID;
    }

    struct TransactionContext {
        Lib_OVMCodec.QueueOrigin ovmL1QUEUEORIGIN;
        uint256 ovmTIMESTAMP;
        uint256 ovmNUMBER;
        uint256 ovmGASLIMIT;
        uint256 ovmTXGASLIMIT;
        address ovmL1TXORIGIN;
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


    /************************************
     * Transaction Execution Entrypoint *
     ************************************/

    function run(
        Lib_OVMCodec.Transaction calldata _transaction,
        address _txStateManager
    ) external;


    /*******************
     * Context Opcodes *
     *******************/

    function ovmCALLER() external view returns (address _caller);
    function ovmADDRESS() external view returns (address _address);
    function ovmTIMESTAMP() external view returns (uint256 _timestamp);
    function ovmNUMBER() external view returns (uint256 _number);
    function ovmGASLIMIT() external view returns (uint256 _gasLimit);
    function ovmCHAINID() external view returns (uint256 _chainId);


    /**********************
     * L2 Context Opcodes *
     **********************/

    function ovmL1QUEUEORIGIN() external view returns (Lib_OVMCodec.QueueOrigin _queueOrigin);
    function ovmL1TXORIGIN() external view returns (address _l1TxOrigin);


    /*******************
     * Halting Opcodes *
     *******************/

    function ovmREVERT(bytes memory _data) external;


    /*****************************
     * Contract Creation Opcodes *
     *****************************/

    function ovmCREATE(bytes memory _bytecode) external returns (address _contract);
    function ovmCREATE2(bytes memory _bytecode, bytes32 _salt) external returns (address _contract);


    /*******************************
     * Account Abstraction Opcodes *
     ******************************/

    function ovmGETNONCE() external returns (uint256 _nonce);
    function ovmSETNONCE(uint256 _nonce) external;
    function ovmCREATEEOA(bytes32 _messageHash, uint8 _v, bytes32 _r, bytes32 _s) external;


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


    /**************************************
     * Public Functions: Execution Safety *
     **************************************/

    function safeCREATE(address _address, bytes memory _bytecode) external;

    /***************************************
     * Public Functions: Execution Context *
     ***************************************/

    function getMaxTransactionGasLimit() external view returns (uint _maxTransactionGasLimit);
}
