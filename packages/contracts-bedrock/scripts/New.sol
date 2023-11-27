// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Vm } from "forge-std/Vm.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import "src/libraries/DisputeTypes.sol";

library New {
    Vm internal constant vm = Vm(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    error CreateFailed(string);
    error Create2Failed(string);

    function _create(bytes memory _code) internal returns (address addr_) {
        assembly {
            addr_ := create(0, add(_code, 0x20), mload(_code))
        }
        require(addr_ != address(0), "New: cannot create");
    }

    function _create2(bytes memory _code, bytes32 _salt) internal returns (address addr_) {
        assembly {
            addr_ := create2(0, add(_code, 0x20), mload(_code), _salt)
        }
    }

    function addressManager() internal returns (address addr_) {
        bytes memory code = vm.getCode("AddressManager.sol:AddressManager");
        addr_ = _create({ _code: code });
        if (addr_ == address(0)) revert CreateFailed("AddressManager");
    }

    function proxyAdmin(address _owner) internal returns (address addr_) {
        bytes memory code = vm.getCode("ProxyAdmin.sol:ProxyAdmin");
        bytes memory args = abi.encode(_owner);
        code = bytes.concat(code, args);
        addr_ = _create({ _code: code });
        if (addr_ == address(0)) revert CreateFailed("ProxyAdmin");
    }

    function l1CrossDomainMessenger(address _optimismPortal, bytes32 _salt) internal returns (address addr_) {
        bytes memory code = vm.getCode("L1CrossDomainMessenger.sol:L1CrossDomainMessenger");
        bytes memory args = abi.encode(_optimismPortal);
        code = bytes.concat(code, args);
        addr_ = _create2({
            _code: code,
            _salt: _salt
        });
        if (addr_ == address(0)) revert CreateFailed("L1CrossDomainMessenger");
    }

    function optimismPortal(
        address _l2Oracle,
        address _guardian,
        bool _paused,
        address _systemConfig,
        bytes32 _salt
    ) internal returns (address addr_) {
        bytes memory code = vm.getCode("OptimismPortal.sol:OptimismPortal");
        code = bytes.concat(code, abi.encode(_l2Oracle, _guardian, _paused, _systemConfig));
        addr_ = _create2({ _code: code, _salt: _salt });
        if (addr_ == address(0)) revert CreateFailed("OptimismPortal");
    }

    function l2OutputOracle(
        uint256 _submissionInterval,
        uint256 _l2BlockTime,
        uint256 _startingBlockNumber,
        uint256 _startingTimestamp,
        address _proposer,
        address _challenger,
        uint256 _finalizationPeriodSeconds,
        bytes32 _salt
    ) internal returns (address _addr) {
        bytes memory code = vm.getCode("L2OutputOracle.sol:L2OutputOracle");
        bytes memory args = abi.encode(_submissionInterval, _l2BlockTime, _startingBlockNumber, _startingTimestamp, _proposer, _challenger, _finalizationPeriodSeconds);
        code = bytes.concat(code, args);
        _addr = _create2({ _code: code, _salt: _salt });
        if (_addr == address(0)) revert CreateFailed("L2OutputOracle");
    }

    function l1StandardBridge(address _messenger, bytes32 _salt) internal returns (address _addr) {
        bytes memory code = vm.getCode("L1StandardBridge.sol:L1StandardBridge");
        bytes memory args = abi.encode(_messenger);
        code = bytes.concat(code, args);
        _addr = _create2({ _code: code, _salt: _salt });
        if (_addr == address(0)) revert CreateFailed("L1StandardBridge");
    }

    function l1ERC721Bridge(address _messenger, address _otherBridge, bytes32 _salt) internal returns (address _addr) {
        bytes memory code = vm.getCode("L1ERC721Bridge.sol:L1ERC721Bridge");
        bytes memory args = abi.encode(_messenger, _otherBridge);
        code = bytes.concat(code, args);
        _addr = _create2({ _code: code, _salt: _salt });
        if (_addr == address(0)) revert CreateFailed("L1ERC721Bridge");
    }

    function systemConfig(
        address _owner,
        uint256 _overhead,
        uint256 _scalar,
        bytes32 _batcherHash,
        uint64 _gasLimit,
        address _unsafeBlockSigner,
        ResourceMetering.ResourceConfig memory _config,
        bytes32 _salt
    ) internal returns (address _addr) {
        bytes memory code = vm.getCode("SystemConfig.sol:SystemConfig");
        bytes memory args = abi.encode(_owner, _overhead, _scalar, _batcherHash, _gasLimit, _unsafeBlockSigner, _config);
        code = bytes.concat(code, args);
        _addr = _create2({ _code: code, _salt: _salt });
        if (_addr == address(0)) revert CreateFailed("SystemConfig");
    }

    function optimismMintableERC20Factory(address _bridge, bytes32 _salt) internal returns (address _addr) {
        bytes memory code = vm.getCode("OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory");
        bytes memory args = abi.encode(_bridge);
        code = bytes.concat(code, args);
        _addr = _create2({ _code: code, _salt: _salt });
        if (_addr == address(0)) revert CreateFailed("OptimismMintableERC20Factory");
    }

    function storageSetter(bytes32 _salt) internal returns (address addr_) {
        bytes memory code = vm.getCode("StorageSetter.sol:StorageSetter");
        addr_ = _create2({ _code: code, _salt: _salt });
        if (addr_ == address(0)) revert CreateFailed("StorageSetter");
    }

    function l1ChugSplashProxy(address _owner) internal returns (address addr_) {
        bytes memory code = vm.getCode("L1ChugSplashProxy.sol:L1ChugSplashProxy");
        bytes memory args = abi.encode(_owner);
        code = bytes.concat(code, args);
        addr_ = _create({ _code: code });
        if (addr_ == address(0)) revert CreateFailed("L1ChugSplashProxy");
    }

    function resolvedDelegateProxy(address _addressManager, string memory _implementationName) internal returns (address addr_) {
        bytes memory code = vm.getCode("ResolvedDelegateProxy.sol:ResolvedDelegateProxy");
        bytes memory args = abi.encode(_addressManager, _implementationName);
        code = bytes.concat(code, args);
        addr_ = _create({ _code: code });
        if (addr_ == address(0)) revert CreateFailed("ResolvedDelegateProxy");
    }

    function proxy(address _admin) internal returns (address addr_) {
        bytes memory code = vm.getCode("Proxy.sol:Proxy");
        bytes memory args = abi.encode(_admin);
        code = bytes.concat(code, args);
        addr_ = _create({ _code: code });
        if (addr_ == address(0)) revert CreateFailed("Proxy");
    }

    function disputeGameFactory(bytes32 _salt) internal returns (address addr_) {
        bytes memory code = vm.getCode("DisputeGameFactory.sol:DisputeGameFactory");
        addr_ = _create2({ _code: code, _salt: _salt });
        if (addr_ == address(0)) revert CreateFailed("DisputeGameFactory");
    }

    function blockOracle(bytes32 _salt) internal returns (address addr_) {
        bytes memory code = vm.getCode("BlockOracle.sol:BlockOracle");
        addr_ = _create2({ _code: code, _salt: _salt });
        if (addr_ == address(0)) revert CreateFailed("BlockOracle");
    }

    function preimageOracle(bytes32 _salt) internal returns (address addr_) {
        bytes memory code = vm.getCode("PreimageOracle.sol:PreimageOracle");
        addr_ = _create2({ _code: code, _salt: _salt });
        if (addr_ == address(0)) revert CreateFailed("PreimageOracle");
    }

    function mips(address _preimageOracle, bytes32 _salt) internal returns (address addr_) {
        bytes memory code = vm.getCode("MIPS.sol:MIPS");
        bytes memory args = abi.encode(_preimageOracle);
        code = bytes.concat(code, args);
        addr_ = _create2({ _code: code, _salt: _salt });
        if (addr_ == address(0)) revert CreateFailed("MIPS");
    }

    function protocolVersions(bytes32 _salt) internal returns (address addr_) {
        bytes memory code = vm.getCode("ProtocolVersions.sol:ProtocolVersions");
        addr_ = _create2({ _code: code, _salt: _salt });
        if (addr_ == address(0)) revert CreateFailed("ProtocolVersions");
    }

    function safeProxyFactory() internal returns (address addr_) {
        bytes memory code = vm.getCode("SafeProxyFactory.sol:SafeProxyFactory.0.8.19");
        addr_ = _create({ _code: code });
        if (addr_ == address(0)) revert CreateFailed("SafeProxyFactory");
    }

    function safe() internal returns (address addr_) {
        bytes memory code = vm.getCode("Safe.sol:Safe.0.8.19");
        addr_ = _create({ _code: code });
        if (addr_ == address(0)) revert CreateFailed("Safe");
    }

    function alphabetVM(Claim _absolutePrestate) internal returns (address addr_) {
        bytes memory code = vm.getCode("AlphabetVM.sol:AlphabetVM");
        bytes memory args = abi.encode(_absolutePrestate);
        code = bytes.concat(code, args);
        addr_ = _create({ _code: code });
        if (addr_ == address(0)) revert CreateFailed("AlphabetVM");
    }

    function faultDisputeGame(
        GameType _gameType,
        Claim _absolutePrestate,
        uint256 _maxGameDepth,
        Duration _gameDuration,
        address _vm,
        address _l2oo,
        address _blockOracle
    ) internal returns (address addr_) {
        bytes memory code = vm.getCode("FaultDisputeGame.sol:FaultDisputeGame");
        bytes memory args = abi.encode(_gameType, _absolutePrestate, _maxGameDepth, _gameDuration, _vm, _l2oo, _blockOracle);
        code = bytes.concat(code, args);
        addr_ = _create({ _code: code });
        if (addr_ == address(0)) revert CreateFailed("FaultDisputeGame");
    }
}
