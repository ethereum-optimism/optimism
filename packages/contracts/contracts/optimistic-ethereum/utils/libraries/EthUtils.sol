pragma solidity ^0.5.0;

/**
 * @title EthUtils
 */
library EthUtils {
    /**
     * Gets the code for a given address.
     * @param _contract Address of the contract to get code for.
     * @return Code for the given address.
     */
    function getCode(
        address _contract
    )
        internal
        view
        returns (
            bytes memory _code
        )
    {
        assembly {
            let size := extcodesize(_contract)
            _code := mload(0x40)
            mstore(0x40, add(_code, and(add(add(size, 0x20), 0x1f), not(0x1f))))
            mstore(_code, size)
            extcodecopy(_contract, add(_code, 0x20), 0, size)
        }

        return _code;
    }
}