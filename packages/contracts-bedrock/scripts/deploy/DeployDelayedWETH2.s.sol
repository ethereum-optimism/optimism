// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Forge
import { Script } from "forge-std/Script.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { LibString } from "@solady/utils/LibString.sol";

// Interfaces
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

/// @title DeployDelayedWETH
contract DeployDelayedWETH2 is Script {
    struct Input {
        string release;
        address proxyAdmin;
        ISuperchainConfig superchainConfigProxy;
        address delayedWethImpl;
        address delayedWethOwner;
        uint256 delayedWethDelay;
    }

    struct Output {
        IDelayedWETH delayedWethImpl;
        IDelayedWETH delayedWethProxy;
    }

    function run(Input memory _input) public returns (Output memory output_) {
        assertValidInput(_input);

        deployDelayedWethProxy(_input, output_);

        assertValidOutput(_input, output_);
    }

    function deployDelayedWethImpl(Input memory _input, Output memory _output) internal virtual {
        string memory release = _input.release;

        IDelayedWETH impl;

        address existingImplementation = _input.delayedWethImpl;
        if (existingImplementation != address(0)) {
            impl = IDelayedWETH(payable(existingImplementation));
        } else if (isDevelopRelease(release)) {
            vm.broadcast(msg.sender);
            impl = IDelayedWETH(
                DeployUtils.create1({
                    _name: "DelayedWETH",
                    _args: DeployUtils.encodeConstructor(
                        abi.encodeCall(IDelayedWETH.__constructor__, (_input.delayedWethDelay))
                    )
                })
            );
        } else {
            revert(string.concat("DeployDelayedWETH: failed to deploy release ", release));
        }

        vm.label(address(impl), "DelayedWETHImpl");
        _output.delayedWethImpl = impl;
    }

    function deployDelayedWethProxy(Input memory _input, Output memory _output) internal virtual {
        vm.broadcast(msg.sender);
        IProxy proxy = IProxy(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (msg.sender)))
            })
        );

        deployDelayedWethImpl(_input, _output);
        IDelayedWETH impl = _output.delayedWethImpl;

        vm.startBroadcast(msg.sender);
        proxy.upgradeToAndCall(
            address(impl), abi.encodeCall(impl.initialize, (_input.delayedWethOwner, _input.superchainConfigProxy))
        );
        proxy.changeAdmin(_input.proxyAdmin);
        vm.stopBroadcast();

        vm.label(address(proxy), "DelayedWETHProxy");
        _output.delayedWethProxy = IDelayedWETH(payable(address(proxy)));
    }

    // A release is considered a 'develop' release if it does not start with 'op-contracts'.
    function isDevelopRelease(string memory _release) internal pure returns (bool) {
        return !LibString.startsWith(_release, "op-contracts");
    }

    function assertValidInput(Input memory _input) internal pure {
        require(_input.delayedWethDelay != 0, "DeployDelayedWETH: delayedWethDelay not set");
        require(_input.proxyAdmin != address(0), "DeployDelayedWETH: proxyAdmin not set");
        require(address(_input.superchainConfigProxy) != address(0), "DeployDelayedWETH: superchainConfigProxy not set");
        require(_input.delayedWethOwner != address(0), "DeployDelayedWETH: delayedWethOwner not set");
        require(!LibString.eq(_input.release, ""), "DeployDelayedWETH: release not set");
    }

    function assertValidOutput(Input memory _input, Output memory _output) internal {
        DeployUtils.assertValidContractAddress(address(_output.delayedWethImpl));
        DeployUtils.assertValidContractAddress(address(_output.delayedWethProxy));

        assertValidDelayedWethImpl(_input, _output);
        assertValidDelayedWethProxy(_input, _output);
    }

    function assertValidDelayedWethImpl(Input memory _input, Output memory _output) internal {
        IProxy proxy = IProxy(payable(address(_output.delayedWethProxy)));
        vm.prank(address(0));
        address impl = proxy.implementation();
        require(impl == address(_output.delayedWethImpl), "DWI-10");
        DeployUtils.assertInitialized({
            _contractAddress: address(_output.delayedWethImpl),
            _isProxy: false,
            _slot: 0,
            _offset: 0
        });
        require(_output.delayedWethImpl.owner() == address(0), "DWI-20");
        require(_output.delayedWethImpl.delay() == _input.delayedWethDelay, "DWI-30");
        require(address(_output.delayedWethImpl.config()) == address(0), "DWI-30");
    }

    function assertValidDelayedWethProxy(Input memory _input, Output memory _output) internal {
        // Check as proxy.
        IProxy proxy = IProxy(payable(address(_output.delayedWethProxy)));
        vm.prank(address(0));
        address admin = proxy.admin();
        require(admin == _input.proxyAdmin, "DWP-10");

        // Check as implementation.
        DeployUtils.assertInitialized({
            _contractAddress: address(_output.delayedWethProxy),
            _isProxy: true,
            _slot: 0,
            _offset: 0
        });
        require(_output.delayedWethProxy.owner() == _input.delayedWethOwner, "DWP-20");
        require(_output.delayedWethProxy.delay() == _input.delayedWethDelay, "DWP-30");
        require(_output.delayedWethProxy.config() == _input.superchainConfigProxy, "DWP-40");
    }
}
