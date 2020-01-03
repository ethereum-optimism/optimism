pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title DataTypes
 * @notice Main data structures which to be used in rollup smart contracts.
 */
contract DataTypes {
    struct Transaction {
        address ovmEntrypoint;
        bytes ovmCalldata;
    }

    struct StorageElement {
        address ovmContractAddress;
        bytes32 ovmStorageSlot;
        bytes32 ovmStorageValue;
    }

    struct ExecutionContext {
        uint timestamp;
        uint queueOrigin;
        address ovmActiveContract;
        address ovmMsgSender;
    }
}
