// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/* Library Imports */
import { Lib_ECDSAUtils } from "../../optimistic-ethereum/libraries/utils/Lib_ECDSAUtils.sol";

/**
 * @title TestLib_ECDSAUtils
 */
contract TestLib_ECDSAUtils {

    function recover(
        bytes memory _message,
        bool _isEthSignedMessage,
        uint8 _v,
        bytes32 _r,
        bytes32 _s
    )
        public
        pure
        returns (
            address _sender
        )
    {
        return Lib_ECDSAUtils.recover(
            _message,
            _isEthSignedMessage,
            _v,
            _r,
            _s
        );
    }
}
