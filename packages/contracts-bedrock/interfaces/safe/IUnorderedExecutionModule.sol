// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

interface IUnorderedExecutionModule is ISemver {
    struct ExecTransactionParams {
        address to;
        uint256 value;
        bytes data;
        Enum.Operation operation;
    }

    error UnorderedExecutionModule_ModuleNotEnabled();
    error UnorderedExecutionModule_HashOnceTooSmall();
    error UnorderedExecutionModule_HashOnceAlreadyUsed();
    error UnorderedExecutionModule_UnsupportedSafe();
    error UnorderedExecutionModule_ExecutionFailed(bytes);

    event TransactionExecuted(Safe indexed safe, bytes32 indexed txHash, uint256 indexed hashOnce);

    function version() external view returns (string memory);
    function __constructor__() external;
    function executed(Safe _safe, uint256 _hashOnce) external view returns (bool executed_);
    function deriveHashOnce(string memory _input) external pure returns (uint256 hashOnce_);
    function transactionHash(
        Safe _safe,
        ExecTransactionParams calldata _params,
        uint256 _hashOnce
    )
        external
        view
        returns (bytes32 txHash_);
    function execute(
        Safe _safe,
        ExecTransactionParams calldata _params,
        uint256 _hashOnce,
        bytes calldata _signatures
    )
        external
        returns (bytes memory returnData_);
}
