// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { SafeCast } from "@openzeppelin/contracts/utils/math/SafeCast.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

import { ChainAssertions } from "scripts/deploy/ChainAssertions.sol";
import { Constants as ScriptConstants } from "scripts/libraries/Constants.sol";
import { Types } from "scripts/libraries/Types.sol";

import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { Claim, Duration, GameType } from "src/dispute/lib/Types.sol";

import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";

contract DeployOPChain is Script {

    struct Output {
        IProxyAdmin opChainProxyAdmin;
        IAddressManager addressManager;
        IL1ERC721Bridge l1ERC721BridgeProxy;
        ISystemConfig systemConfigProxy;
        IOptimismMintableERC20Factory optimismMintableERC20FactoryProxy;
        IL1StandardBridge l1StandardBridgeProxy;
        IL1CrossDomainMessenger l1CrossDomainMessengerProxy;
        IOptimismPortal optimismPortalProxy;
        IETHLockbox ethLockboxProxy;
        IDisputeGameFactory disputeGameFactoryProxy;
        IAnchorStateRegistry anchorStateRegistryProxy;
        IFaultDisputeGame faultDisputeGame;
        IPermissionedDisputeGame permissionedDisputeGame;
        IDelayedWETH delayedWETHPermissionedGameProxy;
        IDelayedWETH delayedWETHPermissionlessGameProxy;
    }

    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        Types.DeployOPChainInput memory doi = abi.decode(_input, (Types.DeployOPChainInput));
        Output memory doo = run(doi);
        return abi.encode(doo);
    }

    function run(Types.DeployOPChainInput memory _doi) public returns (Output memory doo_) {
        require(_doi.opChainProxyAdminOwner != address(0), "DeployOPChainInput: proxy admin owner not set");
        require(_doi.systemConfigOwner != address(0), "DeployOPChainInput: systemConfigOwner not set");
        require(_doi.batcher != address(0), "DeployOPChainInput: batcher not set");
        require(_doi.unsafeBlockSigner != address(0), "DeployOPChainInput: signer not set");
        require(_doi.proposer != address(0), "DeployOPChainInput: proposer not set");
        require(_doi.challenger != address(0), "DeployOPChainInput: challenger not set");
        require(_doi.basefeeScalar != 0, "DeployOPChainInput: basefeeScalar not set");
        require(_doi.blobBaseFeeScalar != 0, "DeployOPChainInput: blobBaseFeeScalar not set");
        require(_doi.l2ChainId != 0 && _doi.l2ChainId != block.chainid, "DeployOPChainInput: invalid l2ChainId");
        require(_doi.opcm != address(0), "DeployOPChainInput: opcm not set");

        IOPContractsManager opcm = IOPContractsManager(_doi.opcm);
        DeployUtils.assertValidContractAddress(address(opcm));

        IOPContractsManager.Roles memory roles = IOPContractsManager.Roles({
            opChainProxyAdminOwner: _doi.opChainProxyAdminOwner,
            systemConfigOwner: _doi.systemConfigOwner,
            batcher: _doi.batcher,
            unsafeBlockSigner: _doi.unsafeBlockSigner,
            proposer: _doi.proposer,
            challenger: _doi.challenger
        });

        IOPContractsManager.DeployInput memory din = IOPContractsManager.DeployInput({
            roles: roles,
            basefeeScalar: _doi.basefeeScalar,
            blobBasefeeScalar: _doi.blobBaseFeeScalar,
            l2ChainId: _doi.l2ChainId,
            startingAnchorRoot: _startingAnchorRoot(),
            saltMixer: _doi.saltMixer,
            gasLimit: _doi.gasLimit,
            disputeGameType: _doi.disputeGameType,
            disputeAbsolutePrestate: _doi.disputeAbsolutePrestate,
            disputeMaxGameDepth: _doi.disputeMaxGameDepth,
            disputeSplitDepth: _doi.disputeSplitDepth,
            disputeClockExtension: _doi.disputeClockExtension,
            disputeMaxClockDuration: _doi.disputeMaxClockDuration
        });

        vm.broadcast(msg.sender);
        IOPContractsManager.DeployOutput memory dout = opcm.deploy(din);

        vm.label(address(dout.opChainProxyAdmin), "opChainProxyAdmin");
        vm.label(address(dout.addressManager), "addressManager");
        vm.label(address(dout.l1ERC721BridgeProxy), "l1ERC721BridgeProxy");
        vm.label(address(dout.systemConfigProxy), "systemConfigProxy");
        vm.label(address(dout.optimismMintableERC20FactoryProxy), "optimismMintableERC20FactoryProxy");
        vm.label(address(dout.l1StandardBridgeProxy), "l1StandardBridgeProxy");
        vm.label(address(dout.l1CrossDomainMessengerProxy), "l1CrossDomainMessengerProxy");
        vm.label(address(dout.optimismPortalProxy), "optimismPortalProxy");
        vm.label(address(dout.ethLockboxProxy), "ethLockboxProxy");
        vm.label(address(dout.disputeGameFactoryProxy), "disputeGameFactoryProxy");
        vm.label(address(dout.anchorStateRegistryProxy), "anchorStateRegistryProxy");
        vm.label(address(dout.permissionedDisputeGame), "permissionedDisputeGame");
        vm.label(address(dout.delayedWETHPermissionedGameProxy), "delayedWETHPermissionedGameProxy");
        // TODO: Eventually switch from Permissioned to Permissionless.
        // vm.label(address(deployOutput.delayedWETHPermissionlessGameProxy), "delayedWETHPermissionlessGameProxy");
        // vm.label(address(deployOutput.faultDisputeGame), "faultDisputeGame");

        doo_.opChainProxyAdmin = dout.opChainProxyAdmin;
        doo_.addressManager = dout.addressManager;
        doo_.l1ERC721BridgeProxy = dout.l1ERC721BridgeProxy;
        doo_.systemConfigProxy = dout.systemConfigProxy;
        doo_.optimismMintableERC20FactoryProxy = dout.optimismMintableERC20FactoryProxy;
        doo_.l1StandardBridgeProxy = dout.l1StandardBridgeProxy;
        doo_.l1CrossDomainMessengerProxy = dout.l1CrossDomainMessengerProxy;
        doo_.optimismPortalProxy = dout.optimismPortalProxy;
        doo_.ethLockboxProxy = dout.ethLockboxProxy;
        doo_.disputeGameFactoryProxy = dout.disputeGameFactoryProxy;
        doo_.anchorStateRegistryProxy = dout.anchorStateRegistryProxy;
        doo_.faultDisputeGame = dout.faultDisputeGame;
        doo_.permissionedDisputeGame = dout.permissionedDisputeGame;
        doo_.delayedWETHPermissionedGameProxy = dout.delayedWETHPermissionedGameProxy;
        doo_.delayedWETHPermissionlessGameProxy = dout.delayedWETHPermissionlessGameProxy;

        checkOutput(_doi, doo_);
    }

    // -------- Validations --------

    function checkOutput(Types.DeployOPChainInput memory _i, Output memory _o) public {
        // With 16 addresses, we'd get a stack too deep error if we tried to do this inline as a
        // single call to `Solarray.addresses`. So we split it into two calls.
        address[] memory addrs1 = Solarray.addresses(
            address(_o.opChainProxyAdmin),
            address(_o.addressManager),
            address(_o.l1ERC721BridgeProxy),
            address(_o.systemConfigProxy),
            address(_o.optimismMintableERC20FactoryProxy),
            address(_o.l1StandardBridgeProxy),
            address(_o.l1CrossDomainMessengerProxy)
        );
        address[] memory addrs2 = Solarray.addresses(
            address(_o.optimismPortalProxy),
            address(_o.disputeGameFactoryProxy),
            address(_o.anchorStateRegistryProxy),
            address(_o.permissionedDisputeGame),
            address(_o.delayedWETHPermissionedGameProxy),
            address(_o.ethLockboxProxy)
            // TODO: Eventually switch from Permissioned to Permissionless. Add this address back in.
            // address(_o.delayedWETHPermissionlessGameProxy)
            // address(_o.faultDisputeGame()),
        );

        DeployUtils.assertValidContractAddresses(Solarray.extend(addrs1, addrs2));
        _assertValidDeploy(_i, _o);
    }

    function _assertValidDeploy(Types.DeployOPChainInput memory _i, Output memory _o) internal {
        Types.ContractSet memory proxies = Types.ContractSet({
            L1CrossDomainMessenger: address(_o.l1CrossDomainMessengerProxy),
            L1StandardBridge: address(_o.l1StandardBridgeProxy),
            L2OutputOracle: address(0),
            DisputeGameFactory: address(_o.disputeGameFactoryProxy),
            DelayedWETH: address(_o.delayedWETHPermissionlessGameProxy),
            PermissionedDelayedWETH: address(_o.delayedWETHPermissionedGameProxy),
            AnchorStateRegistry: address(_o.anchorStateRegistryProxy),
            OptimismMintableERC20Factory: address(_o.optimismMintableERC20FactoryProxy),
            OptimismPortal: address(_o.optimismPortalProxy),
            ETHLockbox: address(_o.ethLockboxProxy),
            SystemConfig: address(_o.systemConfigProxy),
            L1ERC721Bridge: address(_o.l1ERC721BridgeProxy),
            ProtocolVersions: address(0),
            SuperchainConfig: address(0)
        });

        ChainAssertions.checkAnchorStateRegistryProxy(_o.anchorStateRegistryProxy, true);
        ChainAssertions.checkDisputeGameFactory(
            _o.disputeGameFactoryProxy,
            _i.opChainProxyAdminOwner,
            _o.permissionedDisputeGame,
            true
        );
        ChainAssertions.checkL1CrossDomainMessenger(_o.l1CrossDomainMessengerProxy, vm, true);
        ChainAssertions.checkOptimismPortal2({
            _contracts: proxies,
            _superchainConfig: IOPContractsManager(_i.opcm).superchainConfig(),
            _opChainProxyAdminOwner: _i.opChainProxyAdminOwner,
            _isProxy: true
        });
        ChainAssertions.checkSystemConfigProxies(proxies, _i);

        DeployUtils.assertValidContractAddress(address(_o.l1CrossDomainMessengerProxy));
        DeployUtils.assertResolvedDelegateProxyImplementationSet("OVM_L1CrossDomainMessenger", _o.addressManager);

        // Proxies initialized checks
        DeployUtils.assertInitialized({ _contractAddress: address(_o.l1ERC721BridgeProxy), _isProxy: true, _slot: 0, _offset: 0 });
        DeployUtils.assertInitialized({ _contractAddress: address(_o.l1StandardBridgeProxy), _isProxy: true, _slot: 0, _offset: 0 });
        DeployUtils.assertInitialized({ _contractAddress: address(_o.optimismMintableERC20FactoryProxy), _isProxy: true, _slot: 0, _offset: 0 });
        DeployUtils.assertInitialized({ _contractAddress: address(_o.ethLockboxProxy), _isProxy: true, _slot: 0, _offset: 0 });

        // AddressManager owner
        require(_o.addressManager.owner() == address(_o.opChainProxyAdmin), "AM-10");
        assertValidOPChainProxyAdmin(_i, _o);
    }

    function assertValidOPChainProxyAdmin(Types.DeployOPChainInput memory _doi, Output memory _doo) internal {
        IProxyAdmin admin = _doo.opChainProxyAdmin;
        require(admin.owner() == _doi.opChainProxyAdminOwner, "OPCPA-10");
        require(
            admin.getProxyImplementation(address(_doo.l1CrossDomainMessengerProxy))
                == DeployUtils.assertResolvedDelegateProxyImplementationSet("OVM_L1CrossDomainMessenger", _doo.addressManager),
            "OPCPA-20"
        );
        require(address(admin.addressManager()) == address(_doo.addressManager), "OPCPA-30");
        require(
            admin.getProxyImplementation(address(_doo.l1StandardBridgeProxy))
                == DeployUtils.assertL1ChugSplashImplementationSet(address(_doo.l1StandardBridgeProxy)),
            "OPCPA-40"
        );
        require(
            admin.getProxyImplementation(address(_doo.l1ERC721BridgeProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.l1ERC721BridgeProxy)),
            "OPCPA-50"
        );
        require(
            admin.getProxyImplementation(address(_doo.optimismPortalProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.optimismPortalProxy)),
            "OPCPA-60"
        );
        require(
            admin.getProxyImplementation(address(_doo.systemConfigProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.systemConfigProxy)),
            "OPCPA-70"
        );
        require(
            admin.getProxyImplementation(address(_doo.optimismMintableERC20FactoryProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.optimismMintableERC20FactoryProxy)),
            "OPCPA-80"
        );
        require(
            admin.getProxyImplementation(address(_doo.disputeGameFactoryProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.disputeGameFactoryProxy)),
            "OPCPA-90"
        );
        require(
            admin.getProxyImplementation(address(_doo.delayedWETHPermissionedGameProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.delayedWETHPermissionedGameProxy)),
            "OPCPA-100"
        );
        require(
            admin.getProxyImplementation(address(_doo.anchorStateRegistryProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.anchorStateRegistryProxy)),
            "OPCPA-110"
        );
        require(
            admin.getProxyImplementation(address(_doo.ethLockboxProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.ethLockboxProxy)),
            "OPCPA-120"
        );
    }

    function _startingAnchorRoot() internal pure returns (bytes memory) {
        // WARNING: For now always hardcode the starting permissioned game anchor root to 0xdead,
        // and we do not set anything for the permissioned game. This is because we currently only
        // support deploying straight to permissioned games, and the starting root does not
        // matter for that, as long as it is non-zero, since no games will be played. We do not
        // deploy the permissionless game (and therefore do not set a starting root for it here)
        // because to to update to the permissionless game, we will need to update its starting
        // anchor root and deploy a new permissioned dispute game contract anyway.
        //
        // You can `console.logBytes(abi.encode(ScriptConstants.DEFAULT_OUTPUT_ROOT()))` to get the bytes that
        // are hardcoded into `op-chain-ops/deployer/opcm/opchain.go`

        return abi.encode(ScriptConstants.DEFAULT_OUTPUT_ROOT());
    }
}
