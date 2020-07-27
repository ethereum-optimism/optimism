pragma solidity ^0.5.0;

/* Library Imports */
import { ContractAddressGenerator } from "../ContractAddressGenerator.sol";

contract MockContractAddressGenerator {
    function getAddressFromCREATE(
        address _origin,
        uint _nonce
    )
        public
        pure
        returns (address)
    {
        return ContractAddressGenerator.getAddressFromCREATE(
            _origin,
            _nonce
        );
    }

    function getAddressFromCREATE2(
        address _origin,
        bytes32 _salt,
        bytes memory _ovmInitcode
    )
        internal
        pure
        returns (address)
    {
        return ContractAddressGenerator.getAddressFromCREATE2(
            _origin,
            _salt,
            _ovmInitcode
        );
    }
}