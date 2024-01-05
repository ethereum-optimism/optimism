// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { Constants } from "src/libraries/Constants.sol";

contract SystemConfig_GasLimitLowerBound_Invariant is Test {
    SystemConfig public config;

    function setUp() external {
        Proxy proxy = new Proxy(msg.sender);
        SystemConfig configImpl = new SystemConfig({
            _owner: address(0xbeef), // owner
            _overhead: 2100, // overhead
            _scalar: 1000000, // scalar
            _batcherHash: bytes32(hex"abcd"), // batcher hash
            _gasLimit: 30_000_000, // gas limit
            _unsafeBlockSigner: address(1), // unsafe block signer
            _config: Constants.DEFAULT_RESOURCE_CONFIG()
        });

        vm.prank(msg.sender);
        proxy.upgradeToAndCall(
            address(configImpl),
            abi.encodeCall(
                configImpl.initialize,
                (
                    address(0xbeef), // owner
                    2100, // overhead
                    1000000, // scalar
                    bytes32(hex"abcd"), // batcher hash
                    30_000_000, // gas limit
                    address(1), // unsafe block signer
                    Constants.DEFAULT_RESOURCE_CONFIG()
                )
            )
        );

        config = SystemConfig(address(proxy));

        // Set the target contract to the `config`
        targetContract(address(config));
        // Set the target sender to the `config`'s owner (0xbeef)
        targetSender(address(0xbeef));
        // Set the target selector for `setGasLimit`
        // `setGasLimit` is the only function we care about, as it is the only function
        // that can modify the gas limit within the SystemConfig.
        bytes4[] memory selectors = new bytes4[](1);
        selectors[0] = config.setGasLimit.selector;
        FuzzSelector memory selector = FuzzSelector({ addr: address(config), selectors: selectors });
        targetSelector(selector);

        /// Allows the SystemConfig contract to be the target of the invariant test
        /// when it is behind a proxy. Foundry calls this function under the hood to
        /// know the ABI to use when calling the target contract.
        string[] memory artifacts = new string[](1);
        artifacts[0] = "SystemConfig";
        FuzzInterface memory target = FuzzInterface(address(config), artifacts);
        targetInterface(target);
    }

    /// @custom:invariant The gas limit of the `SystemConfig` contract can never be lower
    ///                   than the hard-coded lower bound.
    function invariant_gasLimitLowerBound() external {
        assertTrue(config.gasLimit() >= config.minimumGasLimit());
    }
}
