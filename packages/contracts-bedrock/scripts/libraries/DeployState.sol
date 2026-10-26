// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Scripts
import { Vm } from "forge-std/Vm.sol";
import { Config } from "scripts/libraries/Config.sol";

/// @title DeployState
/// @notice Carries the L1 state of a genesis deployment from one offline forge script run to the next.
///         op-deployer runs each deploy script in its own forge process. The script loads the state that
///         the previous script left, and dumps its own result for the next script. In a live deployment,
///         op-deployer sets none of the environment variables, and both functions do nothing.
library DeployState {
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    /// @notice Loads the allocs from the file that OP_DEPLOYER_STATE_IN names, if it is set.
    function load() internal {
        string memory path = Config.opDeployerStateIn();
        if (bytes(path).length > 0) vm.loadAllocs(path);
    }

    /// @notice Dumps the state to the file that OP_DEPLOYER_STATE_OUT names, if it is set. The dump leaves
    ///         out the script sender, so the sender's nonce and balance go to the file that
    ///         OP_DEPLOYER_SENDER_OUT names.
    /// @param _sender The script sender.
    function dump(address _sender) internal {
        string memory path = Config.opDeployerStateOut();
        if (bytes(path).length == 0) return;
        vm.dumpState(path);
        // Decimal strings, because forge writes a uint as a JSON number only when it fits in 64 bits.
        vm.serializeString("sender", "nonce", vm.toString(vm.getNonce(_sender)));
        string memory sender = vm.serializeString("sender", "balance", vm.toString(_sender.balance));
        vm.writeJson(sender, Config.opDeployerSenderOut());
    }
}
