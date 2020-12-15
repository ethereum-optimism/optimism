// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;

/* Interface Imports */
import { iOVM_BondManager } from "../../iOVM/verification/iOVM_BondManager.sol";

/* Contract Imports */
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";

/**
 * @title mockOVM_BondManager
 */
contract mockOVM_BondManager is iOVM_BondManager, Lib_AddressResolver {
    constructor(
        address _libAddressManager
    )
        Lib_AddressResolver(_libAddressManager)
    {}

    function recordGasSpent(
        bytes32 _preStateRoot,
        bytes32 _txHash,
        address _who,
        uint256 _gasSpent
    )
        override
        public
    {}

    function finalize(
        bytes32 _preStateRoot,
        address _publisher,
        uint256 _timestamp
    )
        override
        public
    {}

    function deposit()
        override
        public
    {}

    function startWithdrawal()
        override
        public
    {}

    function finalizeWithdrawal()
        override
        public
    {}

    function claim(
        address _who
    )
        override
        public
    {}

    function isCollateralized(
        address _who
    )
        override
        public
        view
        returns (
            bool
        )
    {
        // Only authenticate sequencer to submit state root batches.
        return _who == resolve("OVM_Sequencer");
    }

    function getGasSpent(
        bytes32 _preStateRoot,
        address _who
    )
        override
        public
        view
        returns (
            uint256
        )
    {
        return 0;
    }
}
