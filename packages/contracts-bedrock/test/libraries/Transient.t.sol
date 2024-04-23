// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Target contract
import { Transient } from "src/libraries/Transient.sol";

/// @title Base
/// @notice This contract uses the Transient library to set and get transient values.
contract Base {
    /// @notice Set a transient value.
    /// @param _value   Value to set.
    /// @param _target  Target contract to call.
    /// @param _payload Payload to call target with.
    function setTransientValue(uint256 _value, address _target, bytes memory _payload) public {
        Transient.setTransientValue(_value, _target, _payload);
    }

    /// @notice Get value in transient context.
    /// @return _value Transient value.
    function getTransientValue() public view returns (uint256 _value) {
        return Transient.getTransientValue();
    }
}

/// @title NonReentrant
/// @notice This contract uses the Base contract to set a transient value.
contract NonReentrant {
    /// @notice Transient variable.
    uint256 public tVariable;

    /// @notice Set the transient variable.
    function setTVariable() public {
        tVariable = Base(msg.sender).getTransientValue();
    }
}

/// @title Reentrant
/// @notice This contract uses the Base contract to set a transient value and call a function that reads it.
contract Reentrant {
    /// @notice Value to set in msg.sender by reentrant function.
    uint256 public reentrancy_value;

    /// @notice Transient variable.
    uint256 public tVariable;

    /// @notice Set the transient variable and call a function that reads it.
    function reentrant() public {
        Base(msg.sender).setTransientValue(
            reentrancy_value, address(this), abi.encodeWithSelector(this.getTVariable.selector)
        );
    }

    /// @notice Get the transient variable.
    function getTVariable() public {
        tVariable = Base(msg.sender).getTransientValue();
    }

    /// @notice Sets reentrancy value.
    function setReentrancyValue(uint256 _value) public {
        reentrancy_value = _value;
    }
}

/// @title TransientTest
/// @notice Tests the Transient library.
contract TransientTest is Test {
    /// @notice Base contract.
    Base base;

    /// @notice NonReentrant contract.
    NonReentrant nonReentrant;

    /// @notice Reentrant contract.
    Reentrant reentrant;

    /// @notice Set up the test environment.
    function setUp() public {
        base = new Base();
        nonReentrant = new NonReentrant();
        reentrant = new Reentrant();
    }

    /// @notice Test setting a transient variable in a non-reentrant function.
    /// @param _value Value to set.
    function testFuzz_transient_nonReentrant_succeeds(uint256 _value) public {
        // Set the transient value in the base contract.
        base.setTransientValue(_value, address(nonReentrant), abi.encodeCall(NonReentrant.setTVariable, ()));

        // Ensure the transient value is set in the non-reentrant contract.
        assertEq(_value, nonReentrant.tVariable());
    }

    /// @notice Test setting a transient variable in a reentrant function fails.
    /// @param _value Value to fail to set.
    function test_transient_reentrant_fails(uint256 _value, uint256 _reentrancyValue) public {
        // Ensure the value is not the reentrancy value, otherwise the values will match.
        vm.assume(_value != _reentrancyValue);

        // Set the reentrancy value in the reentrant contract.
        reentrant.setReentrancyValue(_reentrancyValue);

        // Set the transient value in the base contract and call the reentrant function in the reentrant contract.
        base.setTransientValue(_value, address(reentrant), abi.encodeWithSelector(Reentrant.reentrant.selector));

        // Ensure the reentrancy value is set in the reentrant contract.
        assertEq(_reentrancyValue, reentrant.tVariable());

        // Ensure the transient value is not set in the reentrant contract.
        assertNotEq(_value, reentrant.tVariable());
    }
}
