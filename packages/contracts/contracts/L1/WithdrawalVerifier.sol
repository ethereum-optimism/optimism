//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { L2OutputOracle } from "./L2OutputOracle.sol";

/**
 * @title WithdrawalVerifier
 */
contract WithdrawalVerifier {
    L2OutputOracle public immutable l2Oracle;
    address public immutable withdrawalsPredeploy;
    // todo: add an immutable finalization window var here

    struct OutputRootProof {
        uint256 timestamp;
        bytes32 version;
        bytes32 stateRoot;
        bytes32 withdrawerRoot;
        bytes32 latestBlockhash;
    }

    // struct WithdrawalProof {
    //     ;
    // }

    constructor(L2OutputOracle _l2Oracle, address _withdrawalsPredeploy) {
        l2Oracle = _l2Oracle;
        withdrawalsPredeploy = _withdrawalsPredeploy;
    }

    function verifyWithdrawal(
        uint256 nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes calldata _data,
        OutputRootProof calldata _outputRootProof
    )
        external
        returns (
            // WithdrawalProof _withdrawalProof
            bool
        )
    {
        // check that the timestamp is 7 days old
        // hash _outputRootProof and compare with the outputOracle's value
        // how do I get the withdrawal root itself?
        require(_outputRootProof.timestamp <= block.timestamp - 7 days, "Too soon");

        bytes32 outputRoot = l2Oracle.getL2Output(_outputRootProof.timestamp);
        require(
            outputRoot ==
                keccak256(
                    abi.encode(
                        _outputRootProof.version,
                        _outputRootProof.stateRoot,
                        _outputRootProof.withdrawerRoot,
                        _outputRootProof.latestBlockhash
                    )
                ),
            "Calculated output root does not match expected value"
        );
    }
}
