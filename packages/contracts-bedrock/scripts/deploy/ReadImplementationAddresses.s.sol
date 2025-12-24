// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IProxy } from "interfaces/universal/IProxy.sol";
import { Script } from "forge-std/Script.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IStaticL1ChugSplashProxy } from "interfaces/legacy/IL1ChugSplashProxy.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

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
        address anchorStateRegistry;
        address l1CrossDomainMessenger;
        address l1ERC721Bridge;
        address l1StandardBridge;
        address optimismMintableERC20Factory;
        address disputeGameFactory;
        address mipsSingleton;
        address preimageOracleSingleton;
        address faultDisputeGameV2;
        address permissionedDisputeGameV2;
        address superFaultDisputeGame;
        address superPermissionedDisputeGame;
        address opcmDeployer;
        address opcmUpgrader;
        address opcmGameTypeAdder;
        address opcmStandardValidator;
        address opcmInteropMigrator;
    }

    function run(Input memory _input) public returns (Output memory output_) {
        // Get implementations from EIP-1967 proxies
        output_.delayedWETH = getEIP1967Impl(_input.delayedWETHPermissionedGameProxy);
        output_.optimismPortal = getEIP1967Impl(_input.optimismPortalProxy);
        output_.systemConfig = getEIP1967Impl(_input.systemConfigProxy);
        output_.l1ERC721Bridge = getEIP1967Impl(_input.l1ERC721BridgeProxy);
        output_.optimismMintableERC20Factory = getEIP1967Impl(_input.optimismMintableERC20FactoryProxy);
        output_.disputeGameFactory = getEIP1967Impl(_input.disputeGameFactoryProxy);

        // Get L1StandardBridge implementation (uses different proxy type)
        vm.prank(address(0));
        output_.l1StandardBridge = IStaticL1ChugSplashProxy(_input.l1StandardBridgeProxy).getImplementation();

        // Check if OPCM v2 is being used
        bool useV2 = IOPContractsManager(_input.opcm).isDevFeatureEnabled(DevFeatures.OPCM_V2);

        if (useV2) {
            // Get implementations from OPCM V2
            IOPContractsManagerV2 opcmV2 = IOPContractsManagerV2(_input.opcm);

            // OPCMV2 doesn't expose these addresses directly, so we set them to zero
            // These are internal to the OPCM container and not meant to be accessed externally
            output_.opcmGameTypeAdder = address(0);
            output_.opcmDeployer = address(0);
            output_.opcmUpgrader = address(0);
            output_.opcmInteropMigrator = address(0);

            // StandardValidator is accessible via the standardValidator() method
            output_.opcmStandardValidator = address(opcmV2.opcmStandardValidator());

            IOPContractsManagerContainer.Implementations memory impls = opcmV2.implementations();
            output_.mipsSingleton = impls.mipsImpl;
            output_.delayedWETH = impls.delayedWETHImpl;
            output_.ethLockbox = impls.ethLockboxImpl;
            output_.anchorStateRegistry = impls.anchorStateRegistryImpl;
            output_.optimismPortalInterop = impls.optimismPortalInteropImpl;
            output_.faultDisputeGameV2 = impls.faultDisputeGameV2Impl;
            output_.permissionedDisputeGameV2 = impls.permissionedDisputeGameV2Impl;
            output_.superFaultDisputeGame = impls.superFaultDisputeGameImpl;
            output_.superPermissionedDisputeGame = impls.superPermissionedDisputeGameImpl;
        } else {
            // Get implementations from OPCM V1
            IOPContractsManager opcm = IOPContractsManager(_input.opcm);
            output_.opcmGameTypeAdder = address(opcm.opcmGameTypeAdder());
            output_.opcmDeployer = address(opcm.opcmDeployer());
            output_.opcmUpgrader = address(opcm.opcmUpgrader());
            output_.opcmInteropMigrator = address(opcm.opcmInteropMigrator());
            output_.opcmStandardValidator = address(opcm.opcmStandardValidator());

            IOPContractsManager.Implementations memory impls = opcm.implementations();
            output_.mipsSingleton = impls.mipsImpl;
            output_.delayedWETH = impls.delayedWETHImpl;
            output_.ethLockbox = impls.ethLockboxImpl;
            output_.anchorStateRegistry = impls.anchorStateRegistryImpl;
            output_.optimismPortalInterop = impls.optimismPortalInteropImpl;
            output_.faultDisputeGameV2 = impls.faultDisputeGameV2Impl;
            output_.permissionedDisputeGameV2 = impls.permissionedDisputeGameV2Impl;
            output_.superFaultDisputeGame = impls.superFaultDisputeGameImpl;
            output_.superPermissionedDisputeGame = impls.superPermissionedDisputeGameImpl;
        }

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

    /// @notice Gets the implementation address from an EIP-1967 proxy
    /// @param _proxy The proxy address to read from
    /// @return impl_ The implementation address
    function getEIP1967Impl(address _proxy) private returns (address impl_) {
        IProxy proxy = IProxy(payable(_proxy));
        vm.prank(address(0));
        impl_ = proxy.implementation();
    }
}
