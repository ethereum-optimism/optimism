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

    function run(Input memory _input) public returns (Output memory output_) {
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

            if (i == 0) output_.delayedWETH = impl;
            else if (i == 1) output_.optimismPortal = impl;
            else if (i == 2) output_.systemConfig = impl;
            else if (i == 3) output_.l1ERC721Bridge = impl;
            else if (i == 4) output_.optimismMintableERC20Factory = impl;
            else if (i == 5) output_.disputeGameFactory = impl;
        }

        // Get L1StandardBridge implementation (uses different proxy type)
        vm.prank(address(0));
        output_.l1StandardBridge = IStaticL1ChugSplashProxy(_input.l1StandardBridgeProxy).getImplementation();

        // Get implementations from OPCM
        IOPContractsManager opcm = IOPContractsManager(_input.opcm);
        output_.mipsSingleton = opcm.implementations().mipsImpl;
        output_.delayedWETH = opcm.implementations().delayedWETHImpl;
        output_.ethLockbox = opcm.implementations().ethLockboxImpl;
        output_.optimismPortalInterop = opcm.implementations().optimismPortalInteropImpl;

        // Get L1CrossDomainMessenger from AddressManager
        IAddressManager am = IAddressManager(_input.addressManager);
        output_.l1CrossDomainMessenger = am.getAddress("OVM_L1CrossDomainMessenger");

        // Get PreimageOracle from MIPS singleton
        output_.preimageOracleSingleton = address(IMIPS64(output_.mipsSingleton).oracle());
    }

    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        Input memory input = abi.decode(_input, (Input));
        Output memory output = run(input);
        return abi.encode(output);
    }
}
