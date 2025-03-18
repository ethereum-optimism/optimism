// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "forge-std/Script.sol";
import "forge-std/console.sol";
import { GameTypes, GameType } from "src/dispute/lib/Types.sol";

/// @notice Contains getters for arbitrary methods from all L1 contracts, including legacy getters
/// that have since been deprecated.
interface IFetcher {
    function guardian() external view returns (address);
    function GUARDIAN() external view returns (address);
    function systemConfig() external view returns (address);
    function SYSTEM_CONFIG() external view returns (address);
    function disputeGameFactory() external view returns (address);
    function superchainConfig() external view returns (address);
    function messenger() external view returns (address);
    function addressManager() external view returns (address);
    function PORTAL() external view returns (address);
    function portal() external view returns (address);
    function l1ERC721Bridge() external view returns (address);
    function optimismMintableERC20Factory() external view returns (address);
    function gameImpls(GameType _gameType) external view returns (address);
    function respectedGameType() external view returns (GameType);
    function anchorStateRegistry() external view returns (address);
    function L2_ORACLE() external view returns (address);
    function vm() external view returns (address);
    function oracle() external view returns (address);
    function challenger() external view returns (address);
    function proposer() external view returns (address);
    function PROPOSER() external view returns (address);
    function batcherHash() external view returns (bytes32);
    function admin() external view returns (address);
    function owner() external view returns (address);
    function unsafeBlockSigner() external view returns (address);
    function weth() external view returns (address);
}

contract FetchChainInfoInput {
    address internal _systemConfigProxy;
    address internal _l1StandardBridgeProxy;

    function set(bytes4 _sel, address _addr) public {
        require(_addr != address(0), "FetchChainInfoInput: cannot set zero address");
        if (_sel == this.systemConfigProxy.selector) _systemConfigProxy = _addr;
        else if (_sel == this.l1StandardBridgeProxy.selector) _l1StandardBridgeProxy = _addr;
        else revert("FetchChainInfoInput: unknown selector");
    }

    function systemConfigProxy() public view returns (address) {
        require(_systemConfigProxy != address(0), "FetchChainInfoInput: systemConfigProxy not set");
        return _systemConfigProxy;
    }

    function l1StandardBridgeProxy() public view returns (address) {
        require(_l1StandardBridgeProxy != address(0), "FetchChainInfoInput: l1StandardBridgeProxy not set");
        return _l1StandardBridgeProxy;
    }
}

contract FetchChainInfoOutput {
    // contract addresses
    address internal _addressManager;
    address internal _l1CrossDomainMessengerProxy;
    address internal _l1ERC721BridgeProxy;
    address internal _l1StandardBridgeProxy;
    address internal _l2OutputOracleProxy;
    address internal _optimismMintableERC20FactoryProxy;
    address internal _optimismPortalProxy;
    address internal _systemConfigProxy;
    address internal _proxyAdmin;
    address internal _superchainConfig;
    address internal _anchorStateRegistryProxy;
    address internal _permissionedWethProxy;
    address internal _permissionlessWethProxy;
    address internal _disputeGameFactoryProxy;
    address internal _faultDisputeGame;
    address internal _mips;
    address internal _permissionedDisputeGame;
    address internal _preimageOracle;
    address internal _daChallengeAddress;

    // roles
    address internal _systemConfigOwner;
    address internal _proxyAdminOwner;
    address internal _guardian;
    address internal _challenger;
    address internal _proposer;
    address internal _unsafeBlockSigner;
    address internal _batchSubmitter;

    // fault proof status
    bool internal _faultProofPermissioned;
    bool internal _faultProofPermissionless;
    GameType internal _respectedGameType;

    function set(bytes4 _sel, address _addr) public {
        if (_sel == this.addressManager.selector) _addressManager = _addr;
        else if (_sel == this.l1CrossDomainMessengerProxy.selector) _l1CrossDomainMessengerProxy = _addr;
        else if (_sel == this.l1ERC721BridgeProxy.selector) _l1ERC721BridgeProxy = _addr;
        else if (_sel == this.l1StandardBridgeProxy.selector) _l1StandardBridgeProxy = _addr;
        else if (_sel == this.l2OutputOracleProxy.selector) _l2OutputOracleProxy = _addr;
        else if (_sel == this.optimismMintableERC20FactoryProxy.selector) _optimismMintableERC20FactoryProxy = _addr;
        else if (_sel == this.optimismPortalProxy.selector) _optimismPortalProxy = _addr;
        else if (_sel == this.systemConfigProxy.selector) _systemConfigProxy = _addr;
        else if (_sel == this.proxyAdmin.selector) _proxyAdmin = _addr;
        else if (_sel == this.superchainConfig.selector) _superchainConfig = _addr;
        else if (_sel == this.anchorStateRegistryProxy.selector) _anchorStateRegistryProxy = _addr;
        else if (_sel == this.permissionedWethProxy.selector) _permissionedWethProxy = _addr;
        else if (_sel == this.permissionlessWethProxy.selector) _permissionlessWethProxy = _addr;
        else if (_sel == this.disputeGameFactoryProxy.selector) _disputeGameFactoryProxy = _addr;
        else if (_sel == this.faultDisputeGame.selector) _faultDisputeGame = _addr;
        else if (_sel == this.mips.selector) _mips = _addr;
        else if (_sel == this.permissionedDisputeGame.selector) _permissionedDisputeGame = _addr;
        else if (_sel == this.preimageOracle.selector) _preimageOracle = _addr;
        else if (_sel == this.daChallengeAddress.selector) _daChallengeAddress = _addr;
        else if (_sel == this.systemConfigOwner.selector) _systemConfigOwner = _addr;
        else if (_sel == this.proxyAdminOwner.selector) _proxyAdminOwner = _addr;
        else if (_sel == this.guardian.selector) _guardian = _addr;
        else if (_sel == this.challenger.selector) _challenger = _addr;
        else if (_sel == this.proposer.selector) _proposer = _addr;
        else if (_sel == this.unsafeBlockSigner.selector) _unsafeBlockSigner = _addr;
        else if (_sel == this.batchSubmitter.selector) _batchSubmitter = _addr;
        else revert("FetchChainInfoOutput: unknown address selector");
    }

    function set(bytes4 _sel, bool _bool) public {
        if (_sel == this.faultProofPermissioned.selector) _faultProofPermissioned = _bool;
        else if (_sel == this.faultProofPermissionless.selector) _faultProofPermissionless = _bool;
        else revert("FetchChainInfoOutput: unknown bool selector");
    }

    function set(bytes4 _sel, GameType _gameType) public {
        if (_sel == this.respectedGameType.selector) _respectedGameType = _gameType;
        else revert("FetchChainInfoOutput: unknown GameType selector");
    }

    function addressManager() public view returns (address) {
        require(_addressManager != address(0), "FetchChainInfoOutput: addressManager not set");
        return _addressManager;
    }

    function l1CrossDomainMessengerProxy() public view returns (address) {
        require(_l1CrossDomainMessengerProxy != address(0), "FetchChainInfoOutput: l1CrossDomainMessengerProxy not set");
        return _l1CrossDomainMessengerProxy;
    }

    function l1ERC721BridgeProxy() public view returns (address) {
        require(_l1ERC721BridgeProxy != address(0), "FetchChainInfoOutput: l1ERC721BridgeProxy not set");
        return _l1ERC721BridgeProxy;
    }

    function l1StandardBridgeProxy() public view returns (address) {
        require(_l1StandardBridgeProxy != address(0), "FetchChainInfoOutput: l1StandardBridgeProxy not set");
        return _l1StandardBridgeProxy;
    }

    function l2OutputOracleProxy() public view returns (address) {
        require(_l2OutputOracleProxy != address(0), "FetchChainInfoOutput: l2OutputOracleProxy not set");
        return _l2OutputOracleProxy;
    }

    function optimismMintableERC20FactoryProxy() public view returns (address) {
        require(
            _optimismMintableERC20FactoryProxy != address(0),
            "FetchChainInfoOutput: optimismMintableERC20FactoryProxy not set"
        );
        return _optimismMintableERC20FactoryProxy;
    }

    function optimismPortalProxy() public view returns (address) {
        require(_optimismPortalProxy != address(0), "FetchChainInfoOutput: optimismPortalProxy not set");
        return _optimismPortalProxy;
    }

    function systemConfigProxy() public view returns (address) {
        require(_systemConfigProxy != address(0), "FetchChainInfoOutput: systemConfigProxy not set");
        return _systemConfigProxy;
    }

    function proxyAdmin() public view returns (address) {
        require(_proxyAdmin != address(0), "FetchChainInfoOutput: proxyAdmin not set");
        return _proxyAdmin;
    }

    function superchainConfig() public view returns (address) {
        require(_superchainConfig != address(0), "FetchChainInfoOutput: superchainConfig not set");
        return _superchainConfig;
    }

    function anchorStateRegistryProxy() public view returns (address) {
        require(_anchorStateRegistryProxy != address(0), "FetchChainInfoOutput: anchorStateRegistryProxy not set");
        return _anchorStateRegistryProxy;
    }

    function permissionedWethProxy() public view returns (address) {
        return _permissionedWethProxy;
    }

    function permissionlessWethProxy() public view returns (address) {
        return _permissionlessWethProxy;
    }

    function disputeGameFactoryProxy() public view returns (address) {
        require(_disputeGameFactoryProxy != address(0), "FetchChainInfoOutput: disputeGameFactoryProxy not set");
        return _disputeGameFactoryProxy;
    }

    function faultDisputeGame() public view returns (address) {
        require(_faultDisputeGame != address(0), "FetchChainInfoOutput: faultDisputeGame not set");
        return _faultDisputeGame;
    }

    function mips() public view returns (address) {
        require(_mips != address(0), "FetchChainInfoOutput: mips not set");
        return _mips;
    }

    function permissionedDisputeGame() public view returns (address) {
        require(_permissionedDisputeGame != address(0), "FetchChainInfoOutput: permissionedDisputeGame not set");
        return _permissionedDisputeGame;
    }

    function preimageOracle() public view returns (address) {
        require(_preimageOracle != address(0), "FetchChainInfoOutput: preimageOracle not set");
        return _preimageOracle;
    }

    function daChallengeAddress() public view returns (address) {
        require(_daChallengeAddress != address(0), "FetchChainInfoOutput: daChallengeAddress not set");
        return _daChallengeAddress;
    }

    function systemConfigOwner() public view returns (address) {
        require(_systemConfigOwner != address(0), "FetchChainInfoOutput: systemConfigOwner not set");
        return _systemConfigOwner;
    }

    function proxyAdminOwner() public view returns (address) {
        require(_proxyAdminOwner != address(0), "FetchChainInfoOutput: proxyAdminOwner not set");
        return _proxyAdminOwner;
    }

    function guardian() public view returns (address) {
        require(_guardian != address(0), "FetchChainInfoOutput: guardian not set");
        return _guardian;
    }

    function challenger() public view returns (address) {
        require(_challenger != address(0), "FetchChainInfoOutput: challenger not set");
        return _challenger;
    }

    function proposer() public view returns (address) {
        require(_proposer != address(0), "FetchChainInfoOutput: proposer not set");
        return _proposer;
    }

    function unsafeBlockSigner() public view returns (address) {
        require(_unsafeBlockSigner != address(0), "FetchChainInfoOutput: unsafeBlockSigner not set");
        return _unsafeBlockSigner;
    }

    function batchSubmitter() public view returns (address) {
        require(_batchSubmitter != address(0), "FetchChainInfoOutput: batchSubmitter not set");
        return _batchSubmitter;
    }

    function faultProofPermissioned() public view returns (bool) {
        return _faultProofPermissioned;
    }

    function faultProofPermissionless() public view returns (bool) {
        return _faultProofPermissionless;
    }

    function respectedGameType() public view returns (GameType) {
        return _respectedGameType;
    }
}

contract FetchChainInfo is Script {
    function run(FetchChainInfoInput _fi, FetchChainInfoOutput _fo) public {
        _processSystemConfig(_fi, _fo);
        _processMessengerAndPortal(_fi, _fo);
        _processDisputeGameFactory(_fo);
        _processProofType(_fo);
    }

    function _processSystemConfig(FetchChainInfoInput _fi, FetchChainInfoOutput _fo) internal {
        address systemConfigProxy = _fi.systemConfigProxy();
        _fo.set(_fo.systemConfigProxy.selector, systemConfigProxy);

        address systemConfigOwner = IFetcher(systemConfigProxy).owner();
        _fo.set(_fo.systemConfigOwner.selector, systemConfigOwner);

        address unsafeBlockSigner = IFetcher(systemConfigProxy).unsafeBlockSigner();
        _fo.set(_fo.unsafeBlockSigner.selector, unsafeBlockSigner);

        address batchSubmitter = _getBatchSubmitter(systemConfigProxy);
        _fo.set(_fo.batchSubmitter.selector, batchSubmitter);

        address proxyAdmin = _getProxyAdmin(systemConfigProxy);
        _fo.set(_fo.proxyAdmin.selector, proxyAdmin);

        address proxyAdminOwner = IFetcher(proxyAdmin).owner();
        _fo.set(_fo.proxyAdminOwner.selector, proxyAdminOwner);

        address l1ERC721BridgeProxy = _getL1ERC721BridgeProxy(systemConfigProxy);
        _fo.set(_fo.l1ERC721BridgeProxy.selector, l1ERC721BridgeProxy);

        address optimismMintableERC20FactoryProxy = _getOptimismMintableERC20FactoryProxy(systemConfigProxy);
        _fo.set(_fo.optimismMintableERC20FactoryProxy.selector, optimismMintableERC20FactoryProxy);
    }

    function _processMessengerAndPortal(FetchChainInfoInput _fi, FetchChainInfoOutput _fo) internal {
        address l1StandardBridgeProxy = _fi.l1StandardBridgeProxy();
        _fo.set(_fo.l1StandardBridgeProxy.selector, l1StandardBridgeProxy);

        address l1CrossDomainMessengerProxy = IFetcher(l1StandardBridgeProxy).messenger();
        _fo.set(_fo.l1CrossDomainMessengerProxy.selector, l1CrossDomainMessengerProxy);

        address addressManager = _getAddressManager(l1CrossDomainMessengerProxy);
        _fo.set(_fo.addressManager.selector, addressManager);

        address optimismPortalProxy = _getOptimismPortalProxy(l1CrossDomainMessengerProxy);
        _fo.set(_fo.optimismPortalProxy.selector, optimismPortalProxy);

        address guardian = _getGuardian(optimismPortalProxy);
        _fo.set(_fo.guardian.selector, guardian);

        address superchainConfig = _getSuperchainConfig(optimismPortalProxy);
        _fo.set(_fo.superchainConfig.selector, superchainConfig);
    }

    function _processDisputeGameFactory(FetchChainInfoOutput _fo) internal {
        address systemConfigProxy = _fo.systemConfigProxy();
        address optimismPortalProxy = _fo.optimismPortalProxy();

        address disputeGameFactoryProxy = _getDisputeGameFactoryProxy(systemConfigProxy);
        if (disputeGameFactoryProxy != address(0)) {
            _fo.set(_fo.disputeGameFactoryProxy.selector, disputeGameFactoryProxy);

            address faultDisputeGame = _getFaultDisputeGame(disputeGameFactoryProxy);
            if (faultDisputeGame != address(0)) {
                _fo.set(_fo.faultDisputeGame.selector, faultDisputeGame);

                address permissionlessWethProxy = _getDelayedWETHProxy(faultDisputeGame);
                _fo.set(_fo.permissionlessWethProxy.selector, permissionlessWethProxy);
            }

            address permissionedDisputeGame = _getPermissionedDisputeGame(disputeGameFactoryProxy);
            if (permissionedDisputeGame != address(0)) {
                _fo.set(_fo.permissionedDisputeGame.selector, permissionedDisputeGame);

                address challenger = IFetcher(permissionedDisputeGame).challenger();
                _fo.set(_fo.challenger.selector, challenger);

                address anchorStateRegistryProxy = _getAnchorStateRegistryProxy(permissionedDisputeGame);
                _fo.set(_fo.anchorStateRegistryProxy.selector, anchorStateRegistryProxy);

                address permissionedWethProxy = _getDelayedWETHProxy(permissionedDisputeGame);
                _fo.set(_fo.permissionedWethProxy.selector, permissionedWethProxy);

                address mips = _getMips(permissionedDisputeGame);
                _fo.set(_fo.mips.selector, mips);

                address preimageOracle = IFetcher(mips).oracle();
                _fo.set(_fo.preimageOracle.selector, preimageOracle);

                address proposer = IFetcher(permissionedDisputeGame).proposer();
                _fo.set(_fo.proposer.selector, proposer);
            }
        } else {
            // Some older chains have L2OutputOracle instead of DisputeGameFactory.
            address l2OutputOracleProxy = IFetcher(optimismPortalProxy).L2_ORACLE();
            _fo.set(_fo.l2OutputOracleProxy.selector, l2OutputOracleProxy);
            address proposer = IFetcher(l2OutputOracleProxy).PROPOSER();
            _fo.set(_fo.proposer.selector, proposer);
        }
    }

    function _getGuardian(address portal) internal view returns (address) {
        try IFetcher(portal).guardian() returns (address guardian) {
            return guardian;
        } catch {
            return IFetcher(portal).GUARDIAN();
        }
    }

    function _getSystemConfigProxy(address portal) internal view returns (address) {
        try IFetcher(portal).systemConfig() returns (address systemConfig) {
            return systemConfig;
        } catch {
            return IFetcher(portal).SYSTEM_CONFIG();
        }
    }

    function _getOptimismPortalProxy(address l1CrossDomainMessengerProxy) internal view returns (address) {
        try IFetcher(l1CrossDomainMessengerProxy).PORTAL() returns (address optimismPortal) {
            return optimismPortal;
        } catch {
            return IFetcher(l1CrossDomainMessengerProxy).portal();
        }
    }

    function _getAddressManager(address l1CrossDomainMessengerProxy) internal view returns (address addressManager) {
        uint256 ADDRESS_MANAGER_MAPPING_SLOT = 1;
        bytes32 slot = keccak256(abi.encode(l1CrossDomainMessengerProxy, ADDRESS_MANAGER_MAPPING_SLOT));
        addressManager = address(uint160(uint256((vm.load(l1CrossDomainMessengerProxy, slot)))));
    }

    function _getL1ERC721BridgeProxy(address systemConfigProxy) internal view returns (address) {
        try IFetcher(systemConfigProxy).l1ERC721Bridge() returns (address l1ERC721BridgeProxy) {
            return l1ERC721BridgeProxy;
        } catch {
            return address(0);
        }
    }

    function _getOptimismMintableERC20FactoryProxy(address systemConfigProxy) internal view returns (address) {
        try IFetcher(systemConfigProxy).optimismMintableERC20Factory() returns (
            address optimismMintableERC20FactoryProxy
        ) {
            return optimismMintableERC20FactoryProxy;
        } catch {
            return address(0);
        }
    }

    function _getDisputeGameFactoryProxy(address systemConfigProxy) internal view returns (address) {
        try IFetcher(systemConfigProxy).disputeGameFactory() returns (address disputeGameFactoryProxy) {
            return disputeGameFactoryProxy;
        } catch {
            // Some older chains have L2OutputOracle instead of DisputeGameFactory
            return address(0);
        }
    }

    function _getSuperchainConfig(address optimismPortalProxy) internal view returns (address) {
        try IFetcher(optimismPortalProxy).superchainConfig() returns (address superchainConfig) {
            return superchainConfig;
        } catch {
            return address(0);
        }
    }

    function _getFaultDisputeGame(address disputeGameFactoryProxy) internal view returns (address) {
        try IFetcher(disputeGameFactoryProxy).gameImpls(GameTypes.CANNON) returns (address faultDisputeGame) {
            return faultDisputeGame;
        } catch {
            return address(0);
        }
    }

    function _getPermissionedDisputeGame(address disputeGameFactoryProxy) internal view returns (address) {
        try IFetcher(disputeGameFactoryProxy).gameImpls(GameTypes.PERMISSIONED_CANNON) returns (
            address permissionedDisputeGame
        ) {
            return permissionedDisputeGame;
        } catch {
            return address(0);
        }
    }

    function _getAnchorStateRegistryProxy(address permissionedDisputeGame) internal view returns (address) {
        return IFetcher(permissionedDisputeGame).anchorStateRegistry();
    }

    function _getDelayedWETHProxy(address disputeGame) internal view returns (address) {
        (bool ok, bytes memory data) = address(disputeGame).staticcall(abi.encodeWithSelector(IFetcher.weth.selector));
        if (ok && data.length == 32) return abi.decode(data, (address));
        else return address(0);
    }

    function _getMips(address permissionedDisputeGame) internal view returns (address) {
        return IFetcher(permissionedDisputeGame).vm();
    }

    function _getBatchSubmitter(address systemConfigProxy) internal view returns (address) {
        bytes32 batcherHash = IFetcher(systemConfigProxy).batcherHash();
        return address(uint160(uint256(batcherHash)));
    }

    function _getProxyAdmin(address systemConfigProxy) internal returns (address) {
        vm.prank(address(0));
        return IFetcher(systemConfigProxy).admin();
    }

    function _processProofType(FetchChainInfoOutput _fo) internal {
        address disputeFactory = _fo.disputeGameFactoryProxy();
        if (disputeFactory == address(0)) {
            return;
        }
        _fo.set(_fo.faultProofPermissioned.selector, true);

        address gameImpls = IFetcher(disputeFactory).gameImpls(GameTypes.CANNON);
        if (gameImpls == address(0)) {
            return;
        }
        _fo.set(_fo.faultProofPermissionless.selector, true);
        _fo.set(_fo.respectedGameType.selector, IFetcher(_fo.optimismPortalProxy()).respectedGameType());
    }
}
