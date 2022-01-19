// SPDX-License-Identifier: MIT
pragma solidity ^0.8.10;

/**
 * For use in testing with a call from a contract rather than an EOA.
 */
contract Dummy {
    error Failed();

    /**
     * Forwards a call.
     * @param _target Address to call
     * @param _data Data to forward
     */
    function forward(address _target, bytes calldata _data) external payable {
        (bool success, ) = _target.call{ value: msg.value }(_data);
        // Silence the 'Return value of low-level calls not used' warning.
        if (!success) {
            revert Failed();
        }
    }
}
