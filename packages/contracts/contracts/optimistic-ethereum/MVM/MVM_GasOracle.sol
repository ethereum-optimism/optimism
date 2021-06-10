// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

contract MVM_GasOracle
    uint _l2price;
    address _setter;
    constructor(uint price, address setter) {
      _l2price = price;
      _setter = setter;
    }

    function setPrice(uint price) {
       require (msg.sender == _setter, "NOT ALLOWED");
       _l2price = price;
    }

    function transferSetter(address newsetter) {
       require (msg.sender == _setter, "NOT ALLOWED");
       _setter = newsetter;
    }
}
