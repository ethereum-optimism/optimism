// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Owned } from "@rari-capital/solmate/src/auth/Owned.sol";

/**
 * @title Transactor
 * @notice Transactor is a minimal contract that can send transactions.
 */
contract Transactor is Owned {
    /**
     * @param _owner Initial contract owner.
     */
    constructor(address _owner) Owned(_owner) {}

    /**
     * Sends a CALL to a target address.
     *
     * @param _target Address to call.
     * @param _data   Data to send with the call.
     * @param _value  ETH value to send with the call.
     *
     * @return Boolean success value.
     * @return Bytes data returned by the call.
     */
    function CALL(
        address _target,
        bytes memory _data,
        uint256 _value
    ) external payable onlyOwner returns (bool, bytes memory) {
        return _target.call{ value: _value }(_data);
    }

    /**
     * Sends a DELEGATECALL to a target address.
     *
     * @param _target Address to call.
     * @param _data   Data to send with the call.
     *
     * @return Boolean success value.
     * @return Bytes data returned by the call.
     */
    function DELEGATECALL(address _target, bytes memory _data)
        external
        payable
        onlyOwner
        returns (bool, bytes memory)
    {
        // slither-disable-next-line controlled-delegatecall
        return _target.delegatecall(_data);
    }
}
