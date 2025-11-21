// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title DummyCaller
/// @notice Generic delegatecall forwarder. Reads target from storage slot 0,
///         forwards calldata via delegatecall, reverts on failure, returns on success.
/// @dev This contract is used to mimic the contract that is used as the source of the
///      delegatecall to the OPCM. In practice this will be the governance 2/2 or similar.
///      The target address must be stored in storage slot 0 before calling any function.
contract DummyCaller {
    address internal _opcmAddr;

    fallback() external {
        address target = _opcmAddr;
        assembly {
            calldatacopy(0, 0, calldatasize())
            let result := delegatecall(gas(), target, 0, calldatasize(), 0, 0)
            returndatacopy(0, 0, returndatasize())
            switch result
            case 0 { revert(0, returndatasize()) }
            default { return(0, returndatasize()) }
        }
    }
}
