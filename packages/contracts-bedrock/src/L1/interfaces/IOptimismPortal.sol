// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IInitializable } from "src/universal/interfaces/IInitializable.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { IResourceMetering } from "src/L1/interfaces/IResourceMetering.sol";
import { Types } from "src/libraries/Types.sol";

/// @title IOptimismPortal
/// @notice Interface for the OptimismPortal contract.
interface IOptimismPortal is IInitializable, IResourceMetering, ISemver {
    event TransactionDeposited(address indexed from, address indexed to, uint256 indexed version, bytes opaqueData);
    event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success);
    event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to);

    receive() external payable;

    function balance() external view returns (uint256);
    function depositERC20Transaction(
        address _to,
        uint256 _mint,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    )
        external;
    function depositTransaction(
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    )
        external
        payable;
    function donateETH() external payable;
    function finalizeWithdrawalTransaction(Types.WithdrawalTransaction memory _tx) external;
    function finalizedWithdrawals(bytes32) external view returns (bool);
    function guardian() external view returns (address);
    function initialize(address _l2Oracle, address _systemConfig, address _superchainConfig) external;
    function isOutputFinalized(uint256 _l2OutputIndex) external view returns (bool);
    function l2Oracle() external view returns (address);
    function l2Sender() external view returns (address);
    function minimumGasLimit(uint64 _byteCount) external pure returns (uint64);
    function paused() external view returns (bool paused_);
    function proveWithdrawalTransaction(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof memory _outputRootProof,
        bytes[] memory _withdrawalProof
    )
        external;
    function provenWithdrawals(bytes32)
        external
        view
        returns (bytes32 outputRoot_, uint128 timestamp_, uint128 l2OutputIndex_);
    function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) external;
    function superchainConfig() external view returns (address);
    function systemConfig() external view returns (address);
}
