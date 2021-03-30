// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_SafeExecutionManagerWrapper } from "../../libraries/wrappers/Lib_SafeExecutionManagerWrapper.sol";

/**
 * @title OVM_ProxySequencerEntrypoint 
 * @dev The Proxy Sequencer Entrypoint is a predeployed proxy to the implementation of the 
 * Sequencer Entrypoint. This will enable the Optimism team to upgrade the Sequencer Entrypoint 
 * contract.
 * 
 * Compiler used: solc
 * Runtime target: OVM
 */
contract OVM_ProxySequencerEntrypoint {

    /*********************
     * Fallback Function *
     *********************/

    fallback()
        external
    {
        Lib_SafeExecutionManagerWrapper.safeDELEGATECALL(
            gasleft(),
            _getImplementation(),
            msg.data
        );
    }


    /********************
     * Public Functions *
     ********************/

    function init(
        address _implementation,
        address _owner
    )
        external
    {
        Lib_SafeExecutionManagerWrapper.safeREQUIRE(
            _getOwner() == address(0),
            "ProxySequencerEntrypoint has already been inited"
        );
        _setOwner(_owner);
        _setImplementation(_implementation);
    }

    function upgrade(
        address _implementation
    )
        external
    {
        Lib_SafeExecutionManagerWrapper.safeREQUIRE(
            _getOwner() == Lib_SafeExecutionManagerWrapper.safeCALLER(),
            "Only owner can upgrade the Entrypoint"
        );

        _setImplementation(_implementation);
    }


    /**********************
     * Internal Functions *
     **********************/

    function _setImplementation(
        address _implementation
    )
        internal
    {
        Lib_SafeExecutionManagerWrapper.safeSSTORE(
            bytes32(uint256(0)),
            bytes32(uint256(uint160(_implementation)))
        );
    }

    function _getImplementation()
        internal
        returns (
            address _implementation
        )
    {
        return address(uint160(uint256(
            Lib_SafeExecutionManagerWrapper.safeSLOAD(
                bytes32(uint256(0))
            )
        )));
    }

    function _setOwner(
        address _owner
    )
        internal
    {
        Lib_SafeExecutionManagerWrapper.safeSSTORE(
            bytes32(uint256(1)),
            bytes32(uint256(uint160(_owner)))
        );
    }

    function _getOwner()
        internal
        returns (
            address _owner
        )
    {
        return address(uint160(uint256(
            Lib_SafeExecutionManagerWrapper.safeSLOAD(
                bytes32(uint256(1))
            )
        )));
    }
}
