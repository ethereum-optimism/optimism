// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

contract MVM_GasOracle{

    uint _l2price;
    address _setter;
    constructor() {
      _l2price = 1_000_000_000_000_000_001;
      _setter = address(0);
    }

    function setPrice(uint price) public {
       require (msg.sender == _setter, "NOT ALLOWED");
       _l2price = price;
    }

    function transferSetter(address newsetter) public {
       require (msg.sender == _setter, "NOT ALLOWED");
       _setter = newsetter;
    }
}
