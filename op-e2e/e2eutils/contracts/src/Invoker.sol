// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

contract Invoker {
    event PrecompileInvoked(address indexed precompile, bytes result);

    function invokePrecompile(address _precompile, bytes memory _input) external {
        // Call the precompile contract with the provided input
        (bool success, bytes memory result) = _precompile.call(_input);
        require(success, "precompile call failed");

        emit PrecompileInvoked(_precompile, result);
    }
}
