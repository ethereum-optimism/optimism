// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import { Script } from "forge-std/Script.sol";
import { console } from "forge-std/console.sol";
import { Safe as GnosisSafe } from "safe-contracts/Safe.sol";
import { SafeProxyFactory as GnosisSafeProxyFactory } from "safe-contracts/proxies/SafeProxyFactory.sol";
import { TestDelegateCall } from "src/TestDelegateCall.sol";

contract DeploySafe is Script {
    function run() external {
        vm.startBroadcast();

        // Deploy Safe singleton (implementation)
        GnosisSafe safeSingleton = new GnosisSafe();

        // Deploy Safe Proxy Factory
        GnosisSafeProxyFactory proxyFactory = new GnosisSafeProxyFactory();

        // Set up Safe configuration
        address[] memory owners = new address[](2);
        owners[0] = 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266; // Anvil account #0
        owners[1] = 0x70997970C51812dc3A010C7d01b50e0d17dc79C8; // Anvil account #1

        uint256 threshold = 1; // Only need 1 signature for testing

        // Create Safe setup data
        bytes memory setupData = abi.encodeCall(
            GnosisSafe.setup,
            (
                owners,
                threshold,
                address(0),         // to (for setup call)
                "",                 // data (for setup call)
                address(0),         // fallbackHandler
                address(0),         // paymentToken
                0,                  // payment
                payable(address(0)) // paymentReceiver
            )
        );

        // Deploy Safe proxy
        GnosisSafe safe = GnosisSafe(payable(proxyFactory.createProxyWithNonce(
            address(safeSingleton),
            setupData,
            123456 // saltNonce - can be any number
        )));

        // Deploy a test contract
        TestDelegateCall testContract = new TestDelegateCall();

        vm.stopBroadcast();

        console.log("Safe Singleton deployed at:", address(safeSingleton));
        console.log("Safe Proxy Factory deployed at:", address(proxyFactory));
        console.log("Safe instance deployed at:", address(safe));
        console.log("Safe owners:", owners[0], owners[1]);
        console.log("Safe threshold:", threshold);
        console.log("TestDelegateCall deployed at:", address(testContract));
    }
}