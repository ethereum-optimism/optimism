pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {StateManager} from "./StateManager.sol";

/**
 * @title PartialStateManager
 * @notice The PartialStateManager is used for the on-chain fraud proof checker.
 *         It is supplied with only the state which is used to execute a single transaction. This
 *         is unlike the FullStateManager which has access to every storage slot.
 */
contract PartialStateManager is StateManager {
  // solium-disable-previous-line no-empty-blocks
  // TODO
}
