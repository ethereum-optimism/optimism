// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Forge
import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";

// Testing
import { FFIInterface } from "test/setup/FFIInterface.sol";

// Contracts
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { BundleExecutorFactory, BundleTransaction } from "scripts/bundles/BundleExecutor.sol";

// Interfaces
import { IProxyAdmin } from "src/universal/interfaces/IProxyAdmin.sol";

struct SimulationStorageOverride {
    bytes32 key;
    bytes32 value;
}

struct SimulationStateOverride {
    address contractAddress;
    SimulationStorageOverride[] overrides;
}

contract ExecuteBundle is Script {
    /// @notice Executes a bundle of transactions from a Safe JSON file.
    /// @param _bundleJsonPath Path to the JSON file containing the bundle of transactions.
    /// @param _proxyAdmin Address of the ProxyAdmin to transfer ownership to (if zero, does nothing).
    function run(string memory _bundleJsonPath, IProxyAdmin _proxyAdmin) public {
        // Etch the FFIInterface.
        FFIInterface ffi = FFIInterface(address(uint160(uint256(keccak256(abi.encode("optimism.ffi"))))));
        vm.etch(address(ffi), vm.getDeployedCode("FFIInterface.sol:FFIInterface"));
        vm.label(address(ffi), "FFIInterface");
        vm.allowCheatcodes(address(ffi));

        // For clarity, store a boolean indicating if we should transfer ownership.
        bool useProxyAdmin = address(_proxyAdmin) != address(0);

        // Start broadcasting transactions.
        vm.startBroadcast();

        // Get the BundleExecutorFactory reference.
        // TODO: Make this dynamic.
        BundleExecutorFactory factory = BundleExecutorFactory(0xBF43aFd221fD3eB8692f55051cE4D8fD0D81EA4b);

        // Dummy salt for now.
        string memory salt = "dummy";

        // Predict the executor address.
        address executor = factory.predict(salt);

        // Transfer ownership of the ProxyAdmin to the BundleExecutor if requested.
        if (useProxyAdmin) {
            require(_proxyAdmin.owner() == msg.sender, "ExecuteBundle: owner mismatch");
            _proxyAdmin.transferOwnership(executor);
        }

        // Generate the bundle transactions.
        BundleTransaction[] memory b = abi.decode(ffi.encodeBundleTransactions(_bundleJsonPath), (BundleTransaction[]));

        // Add extra transaction to return ownership if necessary.
        BundleTransaction[] memory bundle;
        if (useProxyAdmin) {
            // Change the length and copy over original transactions.
            bundle = new BundleTransaction[](b.length + 1);
            for (uint256 i = 0; i < b.length; i++) {
                bundle[i] = b[i];
            }

            // Add the transaction to transfer ownership.
            bundle[b.length] = BundleTransaction({
                to: address(_proxyAdmin),
                value: 0,
                data: abi.encodeWithSelector(IProxyAdmin.transferOwnership.selector, msg.sender)
            });
        } else {
            // Otherwise just use the original bundle.
            bundle = b;
        }

        // Execute the bundle.
        factory.execute(salt, bundle);

        // Verify that ownership was returned if necessary.
        if (useProxyAdmin) {
            require(IProxyAdmin(address(_proxyAdmin)).owner() == msg.sender, "BundleExecutor: ownership not returned");
        }

        // Stop broadcasting transactions.
        vm.stopBroadcast();

        // Generate storage overrides.
        SimulationStorageOverride[] memory storageOverrides1 = new SimulationStorageOverride[](1);
        storageOverrides1[0] =
            SimulationStorageOverride({ key: bytes32(0), value: bytes32(uint256(uint160(address(executor)))) });

        // Collect into state overrides.
        SimulationStateOverride[] memory overrides = new SimulationStateOverride[](1);
        overrides[0] = SimulationStateOverride({ contractAddress: address(_proxyAdmin), overrides: storageOverrides1 });

        // Log the simulation link.
        logSimulationLink(msg.sender, address(factory), abi.encodeCall(BundleExecutorFactory.execute, (salt, bundle)), overrides);
    }

    /// @notice Executes a bundle of transactions from a Safe JSON file.
    /// @param _bundleJsonPath Path to the JSON file containing the bundle of transactions.
    function run(string memory _bundleJsonPath) external {
        run(_bundleJsonPath, IProxyAdmin(address(0)));
    }

    function logSimulationLink(
        address _from,
        address _to,
        bytes memory _data,
        SimulationStateOverride[] memory _overrides
    )
        internal
        view
    {
        // the following characters are url encoded: []{}
        string memory stateOverrides = "%5B";
        for (uint256 i; i < _overrides.length; i++) {
            SimulationStateOverride memory _override = _overrides[i];
            if (i > 0) stateOverrides = string.concat(stateOverrides, ",");
            stateOverrides = string.concat(
                stateOverrides,
                "%7B\"contractAddress\":\"",
                vm.toString(_override.contractAddress),
                "\",\"storage\":%5B"
            );
            for (uint256 j; j < _override.overrides.length; j++) {
                if (j > 0) stateOverrides = string.concat(stateOverrides, ",");
                stateOverrides = string.concat(
                    stateOverrides,
                    "%7B\"key\":\"",
                    vm.toString(_override.overrides[j].key),
                    "\",\"value\":\"",
                    vm.toString(_override.overrides[j].value),
                    "\"%7D"
                );
            }
            stateOverrides = string.concat(stateOverrides, "%5D%7D");
        }
        stateOverrides = string.concat(stateOverrides, "%5D");

        string memory str = string.concat(
            "https://dashboard.tenderly.co/",
            "TENDERLY_PROJECT",
            "/",
            "TENDERLY_USERNAME",
            "/simulator/new?network=",
            vm.toString(block.chainid),
            "&contractAddress=",
            vm.toString(_to),
            "&from=",
            vm.toString(_from),
            "&stateOverrides=",
            stateOverrides
        );
        if (bytes(str).length + _data.length * 2 > 7980) {
            // tenderly's nginx has issues with long URLs, so print the raw input data separately
            str = string.concat(str, "\nInsert the following hex into the 'Raw input data' field:");
            console.log(str);
            console.log(vm.toString(_data));
        } else {
            str = string.concat(str, "&rawFunctionInput=", vm.toString(_data));
            console.log(str);
        }
    }
}
