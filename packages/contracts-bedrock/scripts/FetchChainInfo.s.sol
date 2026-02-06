// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { GameTypes, GameType } from "src/dispute/lib/Types.sol";

/// @notice Contains getters for arbitrary methods from all L1 contracts, including legacy getters
/// that have since been deprecated.
interface IFetcher {
    function guardian() external view returns (address);
    function GUARDIAN() external view returns (address);
    function systemConfig() external view returns (address);
    function SYSTEM_CONFIG() external view returns (address);
    function disputeGameFactory() external view returns (address);
    function ethLockbox() external view returns (address);
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
    function l2Oracle() external view returns (address);
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
    //  - SuperchainContracts
    address internal _superchainConfigProxy;
    //  - ImplementationsContracts
    address internal _mipsImpl;
    address internal _preimageOracleImpl;
    //  - OpChainContracts
    address internal _addressManagerImpl;
    address internal _ethLockboxProxy;
    address internal _l1CrossDomainMessengerProxy;
    address internal _l1Erc721BridgeProxy;
    address internal _l1StandardBridgeProxy;
    address internal _l2OutputOracleProxy;
    address internal _optimismMintableErc20FactoryProxy;
    address internal _optimismPortalProxy;
    address internal _systemConfigProxy;
    address internal _opChainProxyAdminImpl;
    address internal _anchorStateRegistryProxy;
    address internal _delayedWethPermissionedGameProxy;
    address internal _delayedWethPermissionlessGameProxy;
    address internal _disputeGameFactoryProxy;
    address internal _faultDisputeGameImpl;
    address internal _faultDisputeGameCannonKonaImpl;
    address internal _permissionedDisputeGameImpl;

    // roles
    address internal _systemConfigOwner;
    address internal _opChainProxyAdminOwner;
    address internal _opChainGuardian;
    address internal _challenger;
    address internal _proposer;
    address internal _unsafeBlockSigner;
    address internal _batchSubmitter;

    // fault proof status
    bool internal _permissioned;
    bool internal _permissionless;
    GameType internal _respectedGameType;

    function set(bytes4 _sel, address _addr) public {
        // SuperchainContracts
        if (_sel == this.superchainConfigProxy.selector) _superchainConfigProxy = _addr;
        //  - ImplementationsContracts
        else if (_sel == this.mipsImpl.selector) _mipsImpl = _addr;
        else if (_sel == this.preimageOracleImpl.selector) _preimageOracleImpl = _addr;
        //  - OpChainContracts
        else if (_sel == this.addressManagerImpl.selector) _addressManagerImpl = _addr;
        else if (_sel == this.ethLockboxProxy.selector) _ethLockboxProxy = _addr;
        else if (_sel == this.l1CrossDomainMessengerProxy.selector) _l1CrossDomainMessengerProxy = _addr;
        else if (_sel == this.l1Erc721BridgeProxy.selector) _l1Erc721BridgeProxy = _addr;
        else if (_sel == this.l1StandardBridgeProxy.selector) _l1StandardBridgeProxy = _addr;
        else if (_sel == this.l2OutputOracleProxy.selector) _l2OutputOracleProxy = _addr;
        else if (_sel == this.optimismMintableErc20FactoryProxy.selector) _optimismMintableErc20FactoryProxy = _addr;
        else if (_sel == this.optimismPortalProxy.selector) _optimismPortalProxy = _addr;
        else if (_sel == this.systemConfigProxy.selector) _systemConfigProxy = _addr;
        else if (_sel == this.opChainProxyAdminImpl.selector) _opChainProxyAdminImpl = _addr;
        else if (_sel == this.anchorStateRegistryProxy.selector) _anchorStateRegistryProxy = _addr;
        else if (_sel == this.delayedWethPermissionedGameProxy.selector) _delayedWethPermissionedGameProxy = _addr;
        else if (_sel == this.delayedWethPermissionlessGameProxy.selector) _delayedWethPermissionlessGameProxy = _addr;
        else if (_sel == this.disputeGameFactoryProxy.selector) _disputeGameFactoryProxy = _addr;
        else if (_sel == this.faultDisputeGameImpl.selector) _faultDisputeGameImpl = _addr;
        else if (_sel == this.faultDisputeGameCannonKonaImpl.selector) _faultDisputeGameCannonKonaImpl = _addr;
        else if (_sel == this.permissionedDisputeGameImpl.selector) _permissionedDisputeGameImpl = _addr;
        // roles
        else if (_sel == this.systemConfigOwner.selector) _systemConfigOwner = _addr;
        else if (_sel == this.opChainProxyAdminOwner.selector) _opChainProxyAdminOwner = _addr;
        else if (_sel == this.opChainGuardian.selector) _opChainGuardian = _addr;
        else if (_sel == this.challenger.selector) _challenger = _addr;
        else if (_sel == this.proposer.selector) _proposer = _addr;
        else if (_sel == this.unsafeBlockSigner.selector) _unsafeBlockSigner = _addr;
        else if (_sel == this.batchSubmitter.selector) _batchSubmitter = _addr;
        else revert("FetchChainInfoOutput: unknown address selector test");
    }

    function set(bytes4 _sel, bool _bool) public {
        if (_sel == this.permissioned.selector) _permissioned = _bool;
        else if (_sel == this.permissionless.selector) _permissionless = _bool;
        else revert("FetchChainInfoOutput: unknown bool selector");
    }

    function set(bytes4 _sel, GameType _gameType) public {
        if (_sel == this.respectedGameType.selector) _respectedGameType = _gameType;
        else revert("FetchChainInfoOutput: unknown GameType selector");
    }

    function addressManagerImpl() public view returns (address) {
        require(_addressManagerImpl != address(0), "FetchChainInfoOutput: addressManagerImpl not set");
        return _addressManagerImpl;
    }

    function ethLockboxProxy() public view returns (address) {
        return _ethLockboxProxy;
    }

    function l1CrossDomainMessengerProxy() public view returns (address) {
        require(_l1CrossDomainMessengerProxy != address(0), "FetchChainInfoOutput: l1CrossDomainMessengerProxy not set");
        return _l1CrossDomainMessengerProxy;
    }

    function l1Erc721BridgeProxy() public view returns (address) {
        require(_l1Erc721BridgeProxy != address(0), "FetchChainInfoOutput: l1Erc721BridgeProxy not set");
        return _l1Erc721BridgeProxy;
    }

    function l1StandardBridgeProxy() public view returns (address) {
        require(_l1StandardBridgeProxy != address(0), "FetchChainInfoOutput: l1StandardBridgeProxy not set");
        return _l1StandardBridgeProxy;
    }

    function l2OutputOracleProxy() public view returns (address) {
        require(_l2OutputOracleProxy != address(0), "FetchChainInfoOutput: l2OutputOracleProxy not set");
        return _l2OutputOracleProxy;
    }

    function optimismMintableErc20FactoryProxy() public view returns (address) {
        require(
            _optimismMintableErc20FactoryProxy != address(0),
            "FetchChainInfoOutput: optimismMintableErc20FactoryProxy not set"
        );
        return _optimismMintableErc20FactoryProxy;
    }

    function optimismPortalProxy() public view returns (address) {
        require(_optimismPortalProxy != address(0), "FetchChainInfoOutput: optimismPortalProxy not set");
        return _optimismPortalProxy;
    }

    function systemConfigProxy() public view returns (address) {
        require(_systemConfigProxy != address(0), "FetchChainInfoOutput: systemConfigProxy not set");
        return _systemConfigProxy;
    }

    function opChainProxyAdminImpl() public view returns (address) {
        require(_opChainProxyAdminImpl != address(0), "FetchChainInfoOutput: opChainProxyAdminImpl not set");
        return _opChainProxyAdminImpl;
    }

    function superchainConfigProxy() public view returns (address) {
        require(_superchainConfigProxy != address(0), "FetchChainInfoOutput: superchainConfigProxy not set");
        return _superchainConfigProxy;
    }

    function anchorStateRegistryProxy() public view returns (address) {
        require(_anchorStateRegistryProxy != address(0), "FetchChainInfoOutput: anchorStateRegistryProxy not set");
        return _anchorStateRegistryProxy;
    }

    function delayedWethPermissionedGameProxy() public view returns (address) {
        return _delayedWethPermissionedGameProxy;
    }

    function delayedWethPermissionlessGameProxy() public view returns (address) {
        return _delayedWethPermissionlessGameProxy;
    }

    function disputeGameFactoryProxy() public view returns (address) {
        return _disputeGameFactoryProxy;
    }

    function faultDisputeGameImpl() public view returns (address) {
        require(_faultDisputeGameImpl != address(0), "FetchChainInfoOutput: faultDisputeGameImpl not set");
        return _faultDisputeGameImpl;
    }

    function faultDisputeGameCannonKonaImpl() public view returns (address) {
        require(
            _faultDisputeGameCannonKonaImpl != address(0),
            "FetchChainInfoOutput: faultDisputeGameCannonKonaImpl not set"
        );
        return _faultDisputeGameCannonKonaImpl;
    }

    function mipsImpl() public view returns (address) {
        require(_mipsImpl != address(0), "FetchChainInfoOutput: mipsImpl not set");
        return _mipsImpl;
    }

    function permissionedDisputeGameImpl() public view returns (address) {
        require(_permissionedDisputeGameImpl != address(0), "FetchChainInfoOutput: permissionedDisputeGameImpl not set");
        return _permissionedDisputeGameImpl;
    }

    function preimageOracleImpl() public view returns (address) {
        require(_preimageOracleImpl != address(0), "FetchChainInfoOutput: preimageOracleImpl not set");
        return _preimageOracleImpl;
    }

    function systemConfigOwner() public view returns (address) {
        require(_systemConfigOwner != address(0), "FetchChainInfoOutput: systemConfigOwner not set");
        return _systemConfigOwner;
    }

    function opChainProxyAdminOwner() public view returns (address) {
        require(_opChainProxyAdminOwner != address(0), "FetchChainInfoOutput: opChainProxyAdminOwner not set");
        return _opChainProxyAdminOwner;
    }

    function opChainGuardian() public view returns (address) {
        require(_opChainGuardian != address(0), "FetchChainInfoOutput: opChainGuardian not set");
        return _opChainGuardian;
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

    function permissioned() public view returns (bool) {
        return _permissioned;
    }

    function permissionless() public view returns (bool) {
        return _permissionless;
    }

    function respectedGameType() public view returns (GameType) {
        return _respectedGameType;
    }
}

contract FetchChainInfo is Script {
    function run(FetchChainInfoInput _fi, FetchChainInfoOutput _fo) public {
        _processSystemConfig(_fi, _fo);
        _processMessengerAndPortal(_fi, _fo);
        _processFaultProofs(_fo);
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

        address opChainProxyAdminImpl = _getProxyAdmin(systemConfigProxy);
        _fo.set(_fo.opChainProxyAdminImpl.selector, opChainProxyAdminImpl);

        address opChainProxyAdminOwner = IFetcher(opChainProxyAdminImpl).owner();
        _fo.set(_fo.opChainProxyAdminOwner.selector, opChainProxyAdminOwner);

        address l1Erc721BridgeProxy = _getL1ERC721BridgeProxy(systemConfigProxy);
        _fo.set(_fo.l1Erc721BridgeProxy.selector, l1Erc721BridgeProxy);

        address optimismMintableErc20FactoryProxy = _getOptimismMintableERC20FactoryProxy(systemConfigProxy);
        _fo.set(_fo.optimismMintableErc20FactoryProxy.selector, optimismMintableErc20FactoryProxy);
    }

    function _processMessengerAndPortal(FetchChainInfoInput _fi, FetchChainInfoOutput _fo) internal {
        address l1StandardBridgeProxy = _fi.l1StandardBridgeProxy();
        _fo.set(_fo.l1StandardBridgeProxy.selector, l1StandardBridgeProxy);

        address l1CrossDomainMessengerProxy = IFetcher(l1StandardBridgeProxy).messenger();
        _fo.set(_fo.l1CrossDomainMessengerProxy.selector, l1CrossDomainMessengerProxy);

        address addressManagerImpl = _getAddressManager(l1CrossDomainMessengerProxy);
        _fo.set(_fo.addressManagerImpl.selector, addressManagerImpl);

        address optimismPortalProxy = _getOptimismPortalProxy(l1CrossDomainMessengerProxy);
        _fo.set(_fo.optimismPortalProxy.selector, optimismPortalProxy);

        address opChainGuardian = _getGuardian(optimismPortalProxy);
        _fo.set(_fo.opChainGuardian.selector, opChainGuardian);

        address ethLockboxProxy = _getEthLockboxProxy(optimismPortalProxy);
        _fo.set(_fo.ethLockboxProxy.selector, ethLockboxProxy);

        address superchainConfigProxy = _getSuperchainConfigProxy(optimismPortalProxy);
        _fo.set(_fo.superchainConfigProxy.selector, superchainConfigProxy);
    }

    function _processFaultProofs(FetchChainInfoOutput _fo) internal {
        address systemConfigProxy = _fo.systemConfigProxy();
        address optimismPortalProxy = _fo.optimismPortalProxy();

        try IFetcher(optimismPortalProxy).respectedGameType() returns (GameType gameType_) {
            _fo.set(_fo.respectedGameType.selector, gameType_);
        } catch {
            // default respectedGameType to uint32.max since 0 == CANNON
            _fo.set(_fo.respectedGameType.selector, GameType.wrap(type(uint32).max));
            address l2OutputOracleProxy;
            try IFetcher(optimismPortalProxy).l2Oracle() returns (address l2Oracle_) {
                l2OutputOracleProxy = l2Oracle_;
            } catch {
                l2OutputOracleProxy = IFetcher(optimismPortalProxy).L2_ORACLE();
            }
            _fo.set(_fo.l2OutputOracleProxy.selector, l2OutputOracleProxy);

            address proposer = IFetcher(l2OutputOracleProxy).PROPOSER();
            _fo.set(_fo.proposer.selector, proposer);

            // no fault proofs installed so we're done
            return;
        }

        address disputeGameFactoryProxy = _getDisputeGameFactoryProxy(systemConfigProxy);
        if (disputeGameFactoryProxy != address(0)) {
            _fo.set(_fo.disputeGameFactoryProxy.selector, disputeGameFactoryProxy);

            address permissionedDisputeGameImpl = _getPermissionedDisputeGame(disputeGameFactoryProxy);
            if (permissionedDisputeGameImpl != address(0)) {
                // permissioned fault proofs installed
                _fo.set(_fo.permissioned.selector, true);
                _fo.set(_fo.permissionedDisputeGameImpl.selector, permissionedDisputeGameImpl);

                address challenger = IFetcher(permissionedDisputeGameImpl).challenger();
                _fo.set(_fo.challenger.selector, challenger);

                address anchorStateRegistryProxy = IFetcher(permissionedDisputeGameImpl).anchorStateRegistry();
                _fo.set(_fo.anchorStateRegistryProxy.selector, anchorStateRegistryProxy);

                address proposer = IFetcher(permissionedDisputeGameImpl).proposer();
                _fo.set(_fo.proposer.selector, proposer);

                address delayedWethPermissionedGameProxy = _getDelayedWETHProxy(permissionedDisputeGameImpl);
                _fo.set(_fo.delayedWethPermissionedGameProxy.selector, delayedWethPermissionedGameProxy);

                address mipsImpl = IFetcher(permissionedDisputeGameImpl).vm();
                _fo.set(_fo.mipsImpl.selector, mipsImpl);

                address preimageOracleImpl = IFetcher(mipsImpl).oracle();
                _fo.set(_fo.preimageOracleImpl.selector, preimageOracleImpl);
            }

            address faultDisputeGameImpl = _getFaultDisputeGame(disputeGameFactoryProxy, GameTypes.CANNON);
            if (faultDisputeGameImpl != address(0)) {
                // permissionless fault proofs installed
                _fo.set(_fo.faultDisputeGameImpl.selector, faultDisputeGameImpl);
                _fo.set(_fo.permissionless.selector, true);

                address delayedWethPermissionlessGameProxy = _getDelayedWETHProxy(faultDisputeGameImpl);
                _fo.set(_fo.delayedWethPermissionlessGameProxy.selector, delayedWethPermissionlessGameProxy);
            }

            address faultDisputeGameCannonKonaImpl =
                _getFaultDisputeGame(disputeGameFactoryProxy, GameTypes.CANNON_KONA);
            if (faultDisputeGameCannonKonaImpl != address(0)) {
                _fo.set(_fo.faultDisputeGameCannonKonaImpl.selector, faultDisputeGameCannonKonaImpl);
                // if we have CANNON_KONA, we must also have CANNON
            }
        } else {
            // some older chains have L2OutputOracle instead of DisputeGameFactory.
            address l2OutputOracleProxy = IFetcher(optimismPortalProxy).L2_ORACLE();
            _fo.set(_fo.l2OutputOracleProxy.selector, l2OutputOracleProxy);
            address proposer = IFetcher(l2OutputOracleProxy).PROPOSER();
            _fo.set(_fo.proposer.selector, proposer);
        }
    }

    function _getGuardian(address _portal) internal view returns (address) {
        try IFetcher(_portal).guardian() returns (address guardian_) {
            return guardian_;
        } catch {
            return IFetcher(_portal).GUARDIAN();
        }
    }

    function _getSystemConfigProxy(address _portal) internal view returns (address) {
        try IFetcher(_portal).systemConfig() returns (address systemConfig_) {
            return systemConfig_;
        } catch {
            return IFetcher(_portal).SYSTEM_CONFIG();
        }
    }

    function _getOptimismPortalProxy(address _l1CrossDomainMessengerProxy) internal view returns (address) {
        try IFetcher(_l1CrossDomainMessengerProxy).portal() returns (address optimismPortal_) {
            return optimismPortal_;
        } catch {
            return IFetcher(_l1CrossDomainMessengerProxy).PORTAL();
        }
    }

    function _getAddressManager(address _l1CrossDomainMessengerProxy) internal view returns (address) {
        uint256 ADDRESS_MANAGER_MAPPING_SLOT = 1;
        bytes32 slot = keccak256(abi.encode(_l1CrossDomainMessengerProxy, ADDRESS_MANAGER_MAPPING_SLOT));
        return address(uint160(uint256((vm.load(_l1CrossDomainMessengerProxy, slot)))));
    }

    function _getL1ERC721BridgeProxy(address _systemConfigProxy) internal view returns (address) {
        try IFetcher(_systemConfigProxy).l1ERC721Bridge() returns (address l1ERC721BridgeProxy_) {
            return l1ERC721BridgeProxy_;
        } catch {
            return address(0);
        }
    }

    function _getOptimismMintableERC20FactoryProxy(address _systemConfigProxy) internal view returns (address) {
        try IFetcher(_systemConfigProxy).optimismMintableERC20Factory() returns (
            address optimismMintableERC20FactoryProxy_
        ) {
            return optimismMintableERC20FactoryProxy_;
        } catch {
            return address(0);
        }
    }

    function _getDisputeGameFactoryProxy(address _systemConfigProxy) internal view returns (address) {
        try IFetcher(_systemConfigProxy).disputeGameFactory() returns (address disputeGameFactoryProxy_) {
            return disputeGameFactoryProxy_;
        } catch {
            // Some older chains have L2OutputOracle instead of DisputeGameFactory
            return address(0);
        }
    }

    function _getSuperchainConfigProxy(address _optimismPortalProxy) internal view returns (address) {
        try IFetcher(_optimismPortalProxy).superchainConfig() returns (address superchainConfigProxy_) {
            return superchainConfigProxy_;
        } catch {
            return address(0);
        }
    }

    function _getFaultDisputeGame(address _disputeGameFactoryProxy, GameType _gameType)
        internal
        view
        returns (address)
    {
        try IFetcher(_disputeGameFactoryProxy).gameImpls(_gameType) returns (address faultDisputeGame_) {
            return faultDisputeGame_;
        } catch {
            return address(0);
        }
    }

    function _getPermissionedDisputeGame(address _disputeGameFactoryProxy) internal view returns (address) {
        try IFetcher(_disputeGameFactoryProxy).gameImpls(GameTypes.PERMISSIONED_CANNON) returns (
            address permissionedDisputeGame_
        ) {
            return permissionedDisputeGame_;
        } catch {
            return address(0);
        }
    }

    function _getEthLockboxProxy(address _optimismPortalProxy) internal view returns (address) {
        try IFetcher(_optimismPortalProxy).ethLockbox() returns (address ethLockbox_) {
            return ethLockbox_;
        } catch {
            return address(0);
        }
    }

    function _getDelayedWETHProxy(address _disputeGame) internal view returns (address) {
        (bool ok, bytes memory data) = address(_disputeGame).staticcall(abi.encodeCall(IFetcher.weth, ()));
        if (ok && data.length == 32) return abi.decode(data, (address));
        else return address(0);
    }

    function _getBatchSubmitter(address _systemConfigProxy) internal view returns (address) {
        bytes32 batcherHash = IFetcher(_systemConfigProxy).batcherHash();
        return address(uint160(uint256(batcherHash)));
    }

    function _getProxyAdmin(address _systemConfigProxy) internal returns (address) {
        vm.prank(address(0));
        return IFetcher(_systemConfigProxy).admin();
    }
}
