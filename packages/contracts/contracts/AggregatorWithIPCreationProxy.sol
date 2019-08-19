pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import { Aggregator } from "./Aggregator.sol";
import { AggregatorRegistry } from "./AggregatorRegistry.sol";

/**
 * @notice Used just to create an Aggregator Registry, then destroyed
 */
contract AggregatorWithIPCreationProxy {
  AggregatorRegistry public aggregatorRegistry;

  constructor(AggregatorRegistry _aggregatorRegistry, address _authenticationAddress, string memory data) public {
    aggregatorRegistry = _aggregatorRegistry;
    aggregatorRegistry.addAggregator(_authenticationAddress);
  }

  /**
   * @notice The contract is destroyed and remaining ether balance (0) is sent to burner contract at address 0
   */
  function deleteThisContract() public {
    selfdestruct(address(0));
  }
}