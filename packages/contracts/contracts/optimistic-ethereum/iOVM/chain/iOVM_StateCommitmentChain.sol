// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_BaseChain } from "./iOVM_BaseChain.sol";

/* Library Imports */
import { Lib_OVMCodec } from "../../../libraries/codec/Lib_OVMCodec.sol";

interface iOVM_StateCommitmentChain is iOVM_BaseChain {
    function appendStateBatch(bytes32[] calldata _batch) external;
    function deleteStateBatch(Lib_OVMCodec.ChainBatchHeader memory _batchHeader) external;
}
