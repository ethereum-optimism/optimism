// SPDX-License-Identifier: MIT
pragma solidity 0.8.24;

import { Test } from "forge-std/Test.sol";

// Target contract
import { Transient } from "src/libraries/Transient.sol";

contract Base {
    function setTransientValue(uint256 value, address target, bytes memory payload) public {
        Transient.setTransientValue(value, target, payload);
    }

    function getTransientValue() public view returns (uint256 value) {
        return Transient.getTransientValue();
    }
}

contract NonReentrant {
    uint256 public tVariable;

    function setTVariable() public {
        tVariable = Base(msg.sender).getTransientValue();
    }
}

contract Reentrant {
    uint256 public tVariable;

    function reentrant() public {
        Base(msg.sender).setTransientValue(696969, address(this), abi.encodeWithSelector(this.getTVariable.selector));
    }

    function getTVariable() public {
        tVariable = Base(msg.sender).getTransientValue();
    }
}

contract TransientTest is Test {
    Base base;
    NonReentrant nonReentrant;
    Reentrant reentrant;

    function setUp() public {
        base = new Base();
        nonReentrant = new NonReentrant();
        reentrant = new Reentrant();
    }

    function testTransientVariableNonReentrant(uint256 _value) public {
        base.setTransientValue(_value, address(nonReentrant), abi.encodeCall(NonReentrant.setTVariable, ()));

        assertEq(_value, nonReentrant.tVariable());
    }

    function testTransientVariableReentrant() public {
        base.setTransientValue(69_420, address(reentrant), abi.encodeWithSelector(Reentrant.reentrant.selector));

        assertEq(696969, reentrant.tVariable());
    }
}
