// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { L2ToL1MessagePasser } from "src/L2/L2ToL1MessagePasser.sol";

// Interfaces
import { IL1Block } from "interfaces/L2/IL1Block.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000016
/// @title L2ToL1MessagePasserCGT
/// @notice The L2ToL1MessagePasserCGT is a dedicated contract where messages that are being sent from
///         L2 to L1 can be stored. The storage root of this contract is pulled up to the top level
///         of the L2 output to reduce the cost of proving the existence of sent messages.
contract L2ToL1MessagePasserCGT is L2ToL1MessagePasser {
    /// @notice The error thrown when a withdrawal is initiated with value and custom gas token is used.
    error L2ToL1MessagePasserCGT_NotAllowedOnCGTMode();

    /// @custom:semver +custom-gas-token
    function version() public pure override returns (string memory) {
        return string.concat(super.version(), "+custom-gas-token");
    }

    /// @notice Sends a message from L2 to L1.
    /// @param _target   Address to call on L1 execution.
    /// @param _gasLimit Minimum gas limit for executing the message on L1.
    /// @param _data     Data to forward to L1 target.
    function initiateWithdrawal(address _target, uint256 _gasLimit, bytes memory _data) public payable override {
        if (IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).isCustomGasToken() && msg.value > 0) {
            revert L2ToL1MessagePasserCGT_NotAllowedOnCGTMode();
        }
        super.initiateWithdrawal(_target, _gasLimit, _data);
    }
}
