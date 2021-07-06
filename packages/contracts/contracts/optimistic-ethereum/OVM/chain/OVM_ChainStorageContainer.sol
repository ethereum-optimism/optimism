// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_Buffer } from "../../libraries/utils/Lib_Buffer.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";

/* Interface Imports */
import { iOVM_ChainStorageContainer } from "../../iOVM/chain/iOVM_ChainStorageContainer.sol";

/**
 * @title OVM_ChainStorageContainer
 * @dev The Chain Storage Container provides its owner contract with read, write and delete
 * functionality. This provides gas efficiency gains by enabling it to overwrite storage slots which
 * can no longer be used in a fraud proof due to the fraud window having passed, and the associated
 * chain state or transactions being finalized.
 * Three distinct Chain Storage Containers will be deployed on Layer 1:
 * 1. Stores transaction batches for the Canonical Transaction Chain
 * 2. Stores queued transactions for the Canonical Transaction Chain
 * 3. Stores chain state batches for the State Commitment Chain
 *
 * Compiler used: solc
 * Runtime target: EVM
 */
contract OVM_ChainStorageContainer is iOVM_ChainStorageContainer, Lib_AddressResolver {

    /*************
     * Libraries *
     *************/

    using Lib_Buffer for Lib_Buffer.Buffer;


    /*************
     * Variables *
     *************/

    string public owner;
    Lib_Buffer.Buffer internal buffer;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _libAddressManager Address of the Address Manager.
     * @param _owner Name of the contract that owns this container (will be resolved later).
     */
    constructor(
        address _libAddressManager,
        string memory _owner
    )
        Lib_AddressResolver(_libAddressManager)
    {
        owner = _owner;
    }


    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyOwner() {
        require(
            msg.sender == resolve(owner),
            "OVM_ChainStorageContainer: Function can only be called by the owner."
        );
        _;
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * @inheritdoc iOVM_ChainStorageContainer
     */
    function setGlobalMetadata(
        bytes27 _globalMetadata
    )
        override
        public
        onlyOwner
    {
        return buffer.setExtraData(_globalMetadata);
    }

    /**
     * @inheritdoc iOVM_ChainStorageContainer
     */
    function getGlobalMetadata()
        override
        public
        view
        returns (
            bytes27
        )
    {
        return buffer.getExtraData();
    }

    /**
     * @inheritdoc iOVM_ChainStorageContainer
     */
    function length()
        override
        public
        view
        returns (
            uint256
        )
    {
        return uint256(buffer.getLength());
    }

    /**
     * @inheritdoc iOVM_ChainStorageContainer
     */
    function push(
        bytes32 _object
    )
        override
        public
        onlyOwner
    {
        buffer.push(_object);
    }

    /**
     * @inheritdoc iOVM_ChainStorageContainer
     */
    function push(
        bytes32 _object,
        bytes27 _globalMetadata
    )
        override
        public
        onlyOwner
    {
        buffer.push(_object, _globalMetadata);
    }

    /**
     * @inheritdoc iOVM_ChainStorageContainer
     */
    function get(
        uint256 _index
    )
        override
        public
        view
        returns (
            bytes32
        )
    {
        return buffer.get(uint40(_index));
    }

    /**
     * @inheritdoc iOVM_ChainStorageContainer
     */
    function deleteElementsAfterInclusive(
        uint256 _index
    )
        override
        public
        onlyOwner
    {
        buffer.deleteElementsAfterInclusive(
            uint40(_index)
        );
    }

    /**
     * @inheritdoc iOVM_ChainStorageContainer
     */
    function deleteElementsAfterInclusive(
        uint256 _index,
        bytes27 _globalMetadata
    )
        override
        public
        onlyOwner
    {
        buffer.deleteElementsAfterInclusive(
            uint40(_index),
            _globalMetadata
        );
    }
}
