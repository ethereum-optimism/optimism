pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import {ExecutionManager} from "../ExecutionManager.sol";

/**
 * @title SimpleCopier
 * @notice A simple contract testing the execution manager's Code Opcodes.
 */
contract SimpleCopier {
  ExecutionManager executionManager;

  /**
   * Constructor currently accepts an execution manager & stores that in storage.
   * Note this should be the only storage that this contract ever uses & it should be replaced
   * by a hardcoded value once we have the transpiler.
   */
  constructor(address _executionManager) public {
    executionManager = ExecutionManager(_executionManager);
  }

  function getContractCodeSize(address _targetContract) public returns (uint) {
    return executionManager.ovmEXTCODESIZE(_targetContract);
  }

  function getContractCodeHash(address _targetContract) public returns (bytes32) {
    return executionManager.ovmEXTCODEHASH(_targetContract);
  }

  function getContractCodeCopy(address _targetContract, uint _index, uint _length) public returns (bytes memory) {
    return executionManager.ovmCODECOPY(_targetContract, _index, _length);
  }

}
