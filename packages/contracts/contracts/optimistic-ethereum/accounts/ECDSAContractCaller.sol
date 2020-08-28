pragma solidity ^0.5.0;

import { ECDSAUtils } from "../utils/libraries/ECDSAUtils.sol";

contract ECDSAContractCaller {
    function execute(
        bool _isEthSignedMessage,
        bytes memory _transaction,
        uint8 _v,
        bytes32 _r,
        bytes32 _s
    )
        public
        returns (
            bytes memory _ret
        )
    {
        address target;
        if (_isEthSignedMessage) {
            target = ECDSAUtils.recoverEthSignedMessage(
                _transaction,
                _v,
                _r,
                _s
            );
        } else {
            target = ECDSAUtils.recoverNative(
                _transaction,
                _v,
                _r,
                _s
            );
        }

        require(
            target != address(0),
            "Provided signature is invalid."
        );

        (bool success, bytes memory returndata) = target.call(_transaction);
        if (success == false) {
            revert(string(returndata));
        }

        return returndata;
    }
}