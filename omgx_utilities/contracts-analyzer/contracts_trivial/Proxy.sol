// SPDX-License-Identifier: MIT
pragma solidity >0.7.0;

contract Proxy {
  address facetA;  
  address owner;
  // uint256 balance;

  constructor() {
    owner = msg.sender;
    facetA = 0x0b22380B7c423470979AC3eD7d3c07696773dEa1;
    // not implement can't be compiled
    // balance = msg.sender.balance;
  }

  fallback() external payable {
    assembly     {
    
    }
    assembly{
    
    }

    assembly {
    




    
    }
  }
}