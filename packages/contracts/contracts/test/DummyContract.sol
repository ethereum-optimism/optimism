// SPDX-License-Identifier: MIT
pragma solidity ^0.8.10;

/**
 * For use in testing with a call from a contract rather than an EOA.
 */
contract Dummy {
    /**
     * Forwards a call.
     * @param _target Address to call
     * @param _data Data to forward
     */
    function forward(address _target, bytes calldata _data) external payable {
        (bool success, bytes memory ret) = _target.call{ value: msg.value }(_data);
        // Silence the 'Return value of low-level calls not used' warning.
        success;
        ret;
    }
}
