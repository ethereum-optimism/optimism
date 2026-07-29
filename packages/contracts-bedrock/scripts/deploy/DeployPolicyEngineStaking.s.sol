// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";

import { PolicyEngineStaking } from "src/periphery/staking/PolicyEngineStaking.sol";

/// @title DeployPolicyEngineStaking
/// @notice Script used to deploy the PolicyEngineStaking contract.
contract DeployPolicyEngineStaking is Script {
    /// @notice Deploys the PolicyEngineStaking contract.
    /// @param _owner The address that can pause and unpause staking.
    /// @param _token The ERC20 token used for staking.
    function run(address _owner, address _token) public returns (PolicyEngineStaking) {
        require(_owner != address(0), "DeployPolicyEngineStaking: owner cannot be zero address");
        require(_token != address(0), "DeployPolicyEngineStaking: token cannot be zero address");

        vm.broadcast();
        PolicyEngineStaking staking = new PolicyEngineStaking(_owner, _token);

        console.log("PolicyEngineStaking deployed at:", address(staking));

        return staking;
    }
}
