// SPDX-License-Identifier: MIT
pragma solidity >=0.7.0;

import { Lib_IntrinsicGas } from "../../optimistic-ethereum/libraries/utils/Lib_IntrinsicGas.sol";

contract TestLib_IntrinsicGas {
    function ecdsaContractAccount(
        uint256 _datalength
    )
        public
        pure
        returns (
            uint256
        )
    {
       return Lib_IntrinsicGas.ecdsaContractAccount(_datalength);
    }
}
