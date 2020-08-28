pragma solidity ^0.5.0;

import { ExecutionManager } from "../ovm/ExecutionManager.sol";
import { ECDSAUtils } from "../utils/libraries/ECDSAUtils.sol";

contract ECDSAContractAccount {
    struct EOATransaction {
        bool isCreate;
        address target;
        uint256 nonce;
        bytes data;
    }

    address public owner;

    constructor(
        address _owner
    )
        public
    {
        owner = _owner;
    }

    function call(
        bytes memory _transaction
    )
        public
        view
        returns (
            bytes memory _ret
        )
    {
        EOATransaction memory decodedTx = _decodeTransaction(_transaction);

        (bool success, bytes memory returndata) = decodedTx.target.staticcall(decodedTx.data);
        if (success == false) {
            revert(string(returndata));
        }

        return returndata;
    }

    function execute(
        bytes memory _transaction
    )
        public
        returns (
            bytes memory _ret
        )
    {
        EOATransaction memory decodedTx = _decodeTransaction(_transaction);

        ExecutionManager executionManager = ExecutionManager(msg.sender);
        uint256 expectedNonce = executionManager.ovmGETNONCE() + 1;
        require(decodedTx.nonce == expectedNonce);
        executionManager.ovmSETNONCE(expectedNonce);

        if (decodedTx.isCreate) {
            bytes memory bytecode = decodedTx.data;
            address created;
            assembly {
                created := create(0, add(bytecode, 0x20), mload(bytecode))
                if iszero(extcodesize(created)) {
                    revert(0, 0)
                }
            }

            _ret = abi.encode(created);
        } else {
            (bool success, bytes memory returndata) = decodedTx.target.call(decodedTx.data);
            if (success == false) {
                revert(string(returndata));
            }

            _ret = returndata;
        }

        return _ret;
    }

    /*
     * Internal Functions
     */

    function _decodeTransaction(
        bytes memory _transaction
    )
        internal
        pure
        returns (
            EOATransaction memory _decoded
        )
    {
        (
            bool isCreate,
            address target,
            uint256 nonce,
            bytes data
        ) = abi.decode(_transaction, (bool, address, uint256, bytes));

        return EOATransaction({
            isCreate: isCreate,
            target: target,
            nonce: nonce,
            data: data
        });
    }
}