// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Deploy } from "./Deploy.s.sol";
import { Chains } from "./Chains.sol";

import { console2 as console } from "forge-std/console2.sol";

import { Proxy } from "src/universal/Proxy.sol";

/// @dev Deploys the Fault Proof Alpha contracts to the Goerli testnet.
contract DeployFaultAlpha is Deploy {
    /// @dev Override the `onlyDevnet` modifier to only allow deployment to the Goerli testnet.
    modifier onlyDevnet() override {
        if (block.chainid != Chains.Goerli) {
            revert("DeployFaultAlpha: Only deploy to Goerli testnet");
        }
        _;
    }

    /// @dev Override the run function of `Deploy` to only deploy the Fault Proof Alpha contracts.
    function run() public override {
        console.log("Deploying Fault Proof Alpha contracts to Goerli testnet");

        // Save the address of the development `ProxyAdmin`, which is the sender.
        save("ProxyAdmin", msg.sender);

        // Deploy the `DisputeGameFactoryProxy` and `DisputeGameFactory` contracts.
        deployDisputeGameFactoryProxy();
        deployDisputeGameFactory();

        // Initialize the dispute game factory proxy.
        _upgradeDisputeFactoryProxy();

        // Deploy the MIPS VM, the `PreimageOracle`, and the `BlockOracle`.
        deployBlockOracle();
        deployPreimageOracle();
        deployMips();

        // Deploy the Cannon-backed fault dispute game as `GAMEID = 0` in the `DisputeGameFactory`.
        setCannonFaultGameImplementation();

        console.log("Successful deployment.");
    }

    function _upgradeDisputeFactoryProxy() internal broadcast {
        Proxy(mustGetAddress("DisputeGameFactoryProxy")).upgradeToAndCall(
            mustGetAddress("DisputeGameFactory"), abi.encodeWithSignature("initialize(address)", msg.sender)
        );
    }
}
