// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
/* Contract Imports */
import { MVM_Coinbase } from "./MVM_Coinbase.sol";
contract MVM_GasOracle{

    uint _l2price;
    address _setter;
    MVM_Coinbase constant coinbase = MVM_Coinbase(0x4200000000000000000000000000000000000006);
    
    constructor() {
      _l2price = 0;
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
    
    function transferTo(address target, uint256 amount) public {
       require (msg.sender == _setter, "NOT ALLOWED");
        // Transfer fee to relayer.
        require(
            coinbase.transfer(
                target,
                amount
            ),
            "transfer failed."
        );
    }
}