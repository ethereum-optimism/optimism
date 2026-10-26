// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Scripts
import { Vm } from "forge-std/Vm.sol";
import { Config } from "scripts/libraries/Config.sol";

/// @title DeployState
/// @notice Carries the L1 state of a genesis deployment between op-deployer's offline forge runs:
///         each script loads the state that the previous one left and dumps its own. Without the
///         environment variables, as in a live deployment, both functions do nothing.
library DeployState {
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    /// @notice Loads the allocs from the file that OP_DEPLOYER_STATE_IN names, if it is set.
    function load() internal {
        string memory path = Config.opDeployerStateIn();
        if (bytes(path).length > 0) vm.loadAllocs(path);
    }

    /// @notice Dumps the state to OP_DEPLOYER_STATE_OUT, if set. forge leaves the sender out of the
    ///         dump, so its nonce and balance go to OP_DEPLOYER_SENDER_OUT.
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
