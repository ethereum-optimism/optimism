// SPDX-License-Identifier: MIT
pragma solidity ^0.8.8;

/* Interface Imports */
import { IBondManager } from "./IBondManager.sol";

/* Contract Imports */
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";

/**
 * @title BondManager
 * @dev This contract is, for now, a stub of the "real" BondManager that does nothing but
 * allow the "OVM_Proposer" to submit state root batches.
 *
 * Runtime target: EVM
 */
contract BondManager is IBondManager, Lib_AddressResolver {

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
        public
    {}

    function finalize(
        bytes32 _preStateRoot,
        address _publisher,
        uint256 _timestamp
    )
        public
    {}

    function deposit()
        public
    {}

    function startWithdrawal()
        public
    {}

    function finalizeWithdrawal()
        public
    {}

    function claim(
        address _who
    )
        public
    {}

    function isCollateralized(
        address _who
    )
        public
        view
        returns (
            bool
        )
    {
        // Only authenticate sequencer to submit state root batches.
        return _who == resolve("OVM_Proposer");
    }

    function getGasSpent(
        bytes32 _preStateRoot,
        address _who
    )
        public
        pure
        returns (
            uint256
        )
    {
        return 0;
    }
}
