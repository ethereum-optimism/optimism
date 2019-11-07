pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";

interface TransitionEvaluator {
    function evaluateTransition(
        bytes calldata _transition,
        dt.StorageSlot[2] calldata _storageSlots
    ) external view returns(bytes32[2] memory);

    function getTransitionStateRootAndAccessList(
        bytes calldata _rawTransition
    ) external view returns(bytes32, uint32[2] memory);
}
