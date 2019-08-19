pragma solidity ^0.5.1;

import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import { CommitmentChain } from "./CommitmentChain.sol";
import {DataTypes as types} from "./DataTypes.sol";

/**
 * @notice Represents a single aggregator
 */
contract Aggregator {
  address public authenticationAddress;
  CommitmentChain public commitmentContract;
  uint public id;
  mapping(string => string) public metadata;

  constructor(address _authenticationAddress, uint _id) public {
    authenticationAddress = _authenticationAddress;
    commitmentContract = new CommitmentChain();
    id = _id;
  }

  /**
   * @notice Sets this aggregator's metadata at a certain key
   * @param key The key at which this data is stored in metadata
   * @param _data The data we want to add into metadata
   */
  function setMetadata(string memory key, string memory _data) public {
    require(msg.sender == authenticationAddress, "setMetadata can only be called by authenticated address.");
    metadata[key] = _data;
  }

  /**
   * @notice Deletes this aggregator's metadata at a certain key
   * @param key The location of the metadata to delete
   */
  function deleteMetadata(string memory key) public {
    require(msg.sender == authenticationAddress, "deleteMetadata can only be called by authenticated address.");
    delete metadata[key];
  }
}