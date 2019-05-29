pragma solidity ^0.5.1;

import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";


// Example class - a mock class using delivering from ERC20
contract BasicTokenMock is ERC20 {
  constructor(address initialAccount, uint256 initialBalance) public {
    super._mint(initialAccount, initialBalance);
  }
}
