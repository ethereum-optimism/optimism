pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {ExecutionManager} from "./ExecutionManager.sol";

/**
 * @title L2ExecutionManager
 * @notice This extension of ExecutionManager that should only run in L2 because it has optimistic execution details
 *         that are unnecessary and inefficient to run in L1.
 */
contract L2ExecutionManager is ExecutionManager {
  mapping(bytes32 => bytes32) ovmHashToEvmHash;

  constructor(
      uint256 _opcodeWhitelistMask,
      address _owner,
      uint _gasLimit,
      bool _overridePurityChecker
  ) ExecutionManager(_opcodeWhitelistMask, _owner, _gasLimit, _overridePurityChecker) public {}

  /**
   @notice Associates the provided OVM transaction hash with the EVM transaction hash so that we can properly
           look up transaction receipts based on the OVM transaction hash.
   @param ovmTransactionHash The OVM transaction hash, used publicly as the reference to the transaction.
   @param internalTransactionHash The internal transaction hash of the transaction actually executed.
   */
  function mapOvmTransactionHashToInternalTransactionHash(bytes32 ovmTransactionHash, bytes32 internalTransactionHash) public {
    ovmHashToEvmHash[ovmTransactionHash] = internalTransactionHash;
  }

  /**
   @notice Gets the EVM transaction hash associated with the provided OVM transaction hash.
   @param ovmTransactionHash The OVM transaction hash.
   @return The associated EVM transaction hash.
   */
  function getInternalTransactionHash(bytes32 ovmTransactionHash) public view returns (bytes32) {
    return ovmHashToEvmHash[ovmTransactionHash];
  }
}
