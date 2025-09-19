// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IProxy } from "interfaces/universal/IProxy.sol";
import { Script } from "forge-std/Script.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IStaticL1ChugSplashProxy } from "interfaces/legacy/IL1ChugSplashProxy.sol";

contract ReadImplementationAddresses is Script {
    struct Input {
        address addressManager;
        address l1ERC721BridgeProxy;
        address systemConfigProxy;
        address optimismMintableERC20FactoryProxy;
        address l1StandardBridgeProxy;
        address optimismPortalProxy;
        address disputeGameFactoryProxy;
        address delayedWETHPermissionedGameProxy;
        address opcm;
    }

    struct Output {
        address delayedWETH;
        address optimismPortal;
        address optimismPortalInterop;
        address ethLockbox;
        address systemConfig;
        address l1CrossDomainMessenger;
        address l1ERC721Bridge;
        address l1StandardBridge;
        address optimismMintableERC20Factory;
        address disputeGameFactory;
        address mipsSingleton;
        address preimageOracleSingleton;
    }

    function run(Input memory _input) public returns (Output memory _output) {
        // Read implementations from EIP-1967 proxies
        address[6] memory eip1967Proxies = [
            _input.delayedWETHPermissionedGameProxy,
            _input.optimismPortalProxy,
            _input.systemConfigProxy,
            _input.l1ERC721BridgeProxy,
            _input.optimismMintableERC20FactoryProxy,
            _input.disputeGameFactoryProxy
        ];

        // Get implementations from EIP-1967 proxies
        for (uint256 i = 0; i < eip1967Proxies.length; i++) {
            IProxy proxy = IProxy(payable(eip1967Proxies[i]));
            vm.prank(address(0));
            address impl = proxy.implementation();

            if (i == 0) _output.delayedWETH = impl;
            else if (i == 1) _output.optimismPortal = impl;
            else if (i == 2) _output.systemConfig = impl;
            else if (i == 3) _output.l1ERC721Bridge = impl;
            else if (i == 4) _output.optimismMintableERC20Factory = impl;
            else if (i == 5) _output.disputeGameFactory = impl;
        }

        // Get L1StandardBridge implementation (uses different proxy type)
        vm.prank(address(0));
        _output.l1StandardBridge = IStaticL1ChugSplashProxy(_input.l1StandardBridgeProxy).getImplementation();

        // Get implementations from OPCM
        IOPContractsManager opcm = IOPContractsManager(_input.opcm);
        _output.mipsSingleton = opcm.implementations().mipsImpl;
        _output.delayedWETH = opcm.implementations().delayedWETHImpl;
        _output.ethLockbox = opcm.implementations().ethLockboxImpl;
        _output.optimismPortalInterop = opcm.implementations().optimismPortalInteropImpl;

        // Get L1CrossDomainMessenger from AddressManager
        IAddressManager am = IAddressManager(_input.addressManager);
        _output.l1CrossDomainMessenger = am.getAddress("OVM_L1CrossDomainMessenger");

        // Get PreimageOracle from MIPS singleton
        _output.preimageOracleSingleton = address(IMIPS64(_output.mipsSingleton).oracle());
    }

    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        Input memory input = abi.decode(_input, (Input));
        Output memory output = run(input);
        return abi.encode(output);
    }
}
