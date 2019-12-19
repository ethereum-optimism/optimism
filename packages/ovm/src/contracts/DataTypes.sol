pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title DataTypes
 * @notice TODO
 */
contract DataTypes {
    struct Transaction {
        bytes ovmEntrypoint;
        bytes ovmCalldata;
    }

    struct StorageElement {
        address ovmContractAddress;
        bytes32 ovmStorageSlot;
        bytes32 ovmStorageValue;
    }
}
