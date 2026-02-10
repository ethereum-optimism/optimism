// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { GameType } from "src/dispute/lib/Types.sol";

// Interfaces
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";

/// @title OPContractsManagerUtilsCaller
/// @notice OPContractsManagerUtilsCaller is an abstract contract that exists to hide all of the
///         complexity of cheaply calling the OPContractsManagerUtils contract. Most of this logic
///         could simply live inside of the OPContractsManagerV2 contract directly, but it helps to
///         keep the OPContractsManagerV2 contract cleaner and more readable by moving this here.
/// @dev OPContractsManagerUtilsCaller and OPContractsManagerUtils operate together in a way almost
///      identical to an "external library" contract. You could use a real external library, but
///      this is much easier for humans to read and for us to validate offchain.
abstract contract OPContractsManagerUtilsCaller {
    /// @notice Address of the OPContractsManagerUtils contract.
    IOPContractsManagerUtils public immutable opcmUtils;

    /// @param _utils Address of the OPContractsManagerUtils contract.
    constructor(IOPContractsManagerUtils _utils) {
        opcmUtils = _utils;
    }

    /// @notice Maps an L2 chain ID to an L1 batch inbox address as defined by the standard
    ///         configuration's convention. This convention is
    ///         `versionByte || keccak256(bytes32(chainId))[:19]`, where || denotes concatenation,
    ///         versionByte is 0x00, and chainId is a uint256.
    ///         https://specs.optimism.io/protocol/configurability.html#consensus-parameters
    /// @param _l2ChainId The L2 chain ID to map to an L1 batch inbox address.
    /// @return Chain ID mapped to an L1 batch inbox address.
    function _chainIdToBatchInboxAddress(uint256 _l2ChainId) internal view returns (address) {
        return abi.decode(
            _staticcall(abi.encodeCall(IOPContractsManagerUtils.chainIdToBatchInboxAddress, (_l2ChainId))), (address)
        );
    }

    /// @notice Helper for computing a salt for a contract deployment.
    /// @param _l2ChainId The L2 chain ID of the chain being deployed to.
    /// @param _saltMixer The salt mixer to use for the deployment.
    /// @param _contractName The name of the contract to deploy.
    /// @return The computed salt.
    function _computeSalt(
        uint256 _l2ChainId,
        string memory _saltMixer,
        string memory _contractName
    )
        internal
        view
        returns (bytes32)
    {
        return abi.decode(
            _staticcall(abi.encodeCall(IOPContractsManagerUtils.computeSalt, (_l2ChainId, _saltMixer, _contractName))),
            (bytes32)
        );
    }

    /// @notice Helper function to check if an instruction matches a given key.
    /// @param _instruction The instruction to check.
    /// @param _key The key of the instruction to check for.
    /// @return True if the instruction matches, false otherwise.
    function _isMatchingInstructionByKey(
        IOPContractsManagerUtils.ExtraInstruction memory _instruction,
        string memory _key
    )
        internal
        view
        returns (bool)
    {
        return abi.decode(
            _staticcall(abi.encodeCall(IOPContractsManagerUtils.isMatchingInstructionByKey, (_instruction, _key))),
            (bool)
        );
    }

    /// @notice Helper function to check if an instruction matches a given key and data.
    /// @param _instruction The instruction to check.
    /// @param _key The key of the instruction to check for.
    /// @param _data The data of the instruction to check for.
    /// @return True if the instruction matches, false otherwise.
    function _isMatchingInstruction(
        IOPContractsManagerUtils.ExtraInstruction memory _instruction,
        string memory _key,
        bytes memory _data
    )
        internal
        view
        returns (bool)
    {
        return abi.decode(
            _staticcall(abi.encodeCall(IOPContractsManagerUtils.isMatchingInstruction, (_instruction, _key, _data))),
            (bool)
        );
    }

    /// @notice Helper function to load data from a source contract as bytes.
    /// @param _source The source contract to load the data from.
    /// @param _selector The selector of the function to call on the source contract.
    /// @param _name The name of the field to load.
    /// @param _instructions The extra upgrade instructions for the data load.
    /// @return Data retrieved from the source contract.
    function _loadBytes(
        address _source,
        bytes4 _selector,
        string memory _name,
        IOPContractsManagerUtils.ExtraInstruction[] memory _instructions
    )
        internal
        view
        returns (bytes memory)
    {
        return abi.decode(
            _staticcall(abi.encodeCall(IOPContractsManagerUtils.loadBytes, (_source, _selector, _name, _instructions))),
            (bytes)
        );
    }

    /// @notice Attempts to load a proxy from a source function where the proxy should be found. If
    ///         the proxy isn't found at the source, or the call to the source fails, we build a
    ///         new proxy instead. Calls to source contracts MUST NOT fail under any circumstances
    ///         other than the function not existing (which can happen in an upgrade scenario).
    /// @param _source The source contract to load the proxy from.
    /// @param _selector The selector of the function to call on the source contract.
    /// @param _args The basic arguments for the proxy deployment.
    /// @param _contractName The name of the contract to deploy.
    /// @param _instructions The extra upgrade instructions for the proxy deployment.
    /// @return The address of the loaded or built proxy.
    function _loadOrDeployProxy(
        address _source,
        bytes4 _selector,
        IOPContractsManagerUtils.ProxyDeployArgs memory _args,
        string memory _contractName,
        IOPContractsManagerUtils.ExtraInstruction[] memory _instructions
    )
        internal
        returns (address payable)
    {
        return payable(
            abi.decode(
                _delegatecall(
                    abi.encodeCall(
                        IOPContractsManagerUtils.loadOrDeployProxy,
                        (_source, _selector, _args, _contractName, _instructions)
                    )
                ),
                (address)
            )
        );
    }

    /// @notice Upgrades a contract by resetting the initialized slot and calling the initializer.
    /// @param _proxyAdmin The proxy admin of the contract.
    /// @param _target The target of the contract.
    /// @param _implementation The implementation of the contract.
    /// @param _data The data to call the initializer with.
    function _upgrade(IProxyAdmin _proxyAdmin, address _target, address _implementation, bytes memory _data) internal {
        _upgrade(_proxyAdmin, _target, _implementation, _data, bytes32(0), 0);
    }

    /// @notice Upgrades a contract by resetting the initialized slot and calling the initializer.
    /// @param _proxyAdmin The proxy admin of the contract.
    /// @param _target The target of the contract.
    /// @param _implementation The implementation of the contract.
    /// @param _data The data to call the initializer with.
    /// @param _slot The slot where the initialized value is located.
    /// @param _offset The offset of the initializer value in the slot.
    function _upgrade(
        IProxyAdmin _proxyAdmin,
        address _target,
        address _implementation,
        bytes memory _data,
        bytes32 _slot,
        uint8 _offset
    )
        internal
    {
        _delegatecall(
            abi.encodeCall(
                IOPContractsManagerUtils.upgrade, (_proxyAdmin, _target, _implementation, _data, _slot, _offset)
            )
        );
    }

    /// @notice Helper calling the utils contract (via delegatecall).
    /// @param _data Calldata to send to the utils contract.
    /// @return Result of the call.
    function _delegatecall(bytes memory _data) internal returns (bytes memory) {
        (bool success, bytes memory result) = address(opcmUtils).delegatecall(_data);
        if (!success) {
            assembly {
                revert(add(result, 0x20), mload(result))
            }
        }
        return result;
    }

    /// @notice Helper calling the utils contract (via staticcall).
    /// @param _data Calldata to send to the utils contract.
    /// @return Result of the call.
    function _staticcall(bytes memory _data) internal view returns (bytes memory) {
        (bool success, bytes memory result) = address(opcmUtils).staticcall(_data);
        if (!success) {
            assembly {
                revert(add(result, 0x20), mload(result))
            }
        }
        return result;
    }

    /// @notice Helper for retrieving the dispute game implementation for a given game type.
    /// @param _gameType The game type to retrieve the implementation for.
    /// @return The dispute game implementation.
    function _getGameImpl(GameType _gameType) internal view returns (IDisputeGame) {
        return
            abi.decode(_staticcall(abi.encodeCall(IOPContractsManagerUtils.getGameImpl, (_gameType))), (IDisputeGame));
    }

    /// @notice Helper for creating game constructor arguments.
    /// @param _l2ChainId The L2 chain ID.
    /// @param _anchorStateRegistry The AnchorStateRegistry to use for dispute games.
    /// @param _delayedWETH The DelayedWETH to use for dispute games.
    /// @param _gcfg Configuration for the dispute game to create.
    /// @return The game constructor arguments.
    function _makeGameArgs(
        uint256 _l2ChainId,
        IAnchorStateRegistry _anchorStateRegistry,
        IDelayedWETH _delayedWETH,
        IOPContractsManagerUtils.DisputeGameConfig memory _gcfg
    )
        internal
        view
        returns (bytes memory)
    {
        return abi.decode(
            _staticcall(
                abi.encodeCall(
                    IOPContractsManagerUtils.makeGameArgs, (_l2ChainId, _anchorStateRegistry, _delayedWETH, _gcfg)
                )
            ),
            (bytes)
        );
    }
}
