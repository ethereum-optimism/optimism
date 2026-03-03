// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { Claim, GameType } from "src/dispute/lib/Types.sol";

interface IOPContractsManagerUtils {
    struct ProxyDeployArgs {
        IProxyAdmin proxyAdmin;
        IAddressManager addressManager;
        uint256 l2ChainId;
        string saltMixer;
    }

    struct ExtraInstruction {
        string key;
        bytes data;
    }

    /// @notice Configuration struct for the FaultDisputeGame.
    struct FaultDisputeGameConfig {
        Claim absolutePrestate;
    }

    /// @notice Configuration struct for the PermissionedDisputeGame.
    struct PermissionedDisputeGameConfig {
        Claim absolutePrestate;
        address proposer;
        address challenger;
    }

    /// @notice Generic dispute game configuration data.
    struct DisputeGameConfig {
        bool enabled;
        uint256 initBond;
        GameType gameType;
        bytes gameArgs;
    }

    event ProxyCreation(string name, address proxy);

    error OPContractsManagerUtils_DowngradeNotAllowed(address _contract);
    error OPContractsManagerUtils_ExtraTagInProd(address _contract);
    error OPContractsManagerUtils_ConfigLoadFailed(string _name);
    error OPContractsManagerUtils_ProxyMustLoad(string _name);
    error OPContractsManagerUtils_UnsupportedGameType();
    error ReservedBitsSet();
    error UnsupportedERCVersion(uint8 version);
    error SemverComp_InvalidSemverParts();
    error DeploymentFailed();
    error UnexpectedPreambleData(bytes data);
    error NotABlueprint();
    error EmptyInitcode();
    error BytesArrayTooLong();
    error IdentityPrecompileCallFailed();

    function implementations() external view returns (IOPContractsManagerContainer.Implementations memory);
    function blueprints() external view returns (IOPContractsManagerContainer.Blueprints memory);
    function contractsContainer() external view returns (IOPContractsManagerContainer);
    function chainIdToBatchInboxAddress(uint256 _l2ChainId) external pure returns (address);
    function computeSalt(
        uint256 _l2ChainId,
        string memory _saltMixer,
        string memory _contractName
    )
        external
        pure
        returns (bytes32);

    function isMatchingInstructionByKey(
        ExtraInstruction memory _instruction,
        string memory _key
    )
        external
        pure
        returns (bool);

    function isMatchingInstruction(
        ExtraInstruction memory _instruction,
        string memory _key,
        bytes memory _data
    )
        external
        pure
        returns (bool);

    function hasInstruction(
        ExtraInstruction[] memory _instructions,
        string memory _key,
        bytes memory _data
    )
        external
        pure
        returns (bool);

    function getInstructionByKey(
        ExtraInstruction[] memory _instructions,
        string memory _key
    )
        external
        pure
        returns (ExtraInstruction memory);

    function loadBytes(
        address _source,
        bytes4 _selector,
        string memory _name,
        ExtraInstruction[] memory _instructions
    )
        external
        view
        returns (bytes memory);

    function loadOrDeployProxy(
        address _source,
        bytes4 _selector,
        ProxyDeployArgs memory _args,
        string memory _contractName,
        ExtraInstruction[] memory _instructions
    )
        external
        returns (address payable);

    function upgrade(
        IProxyAdmin _proxyAdmin,
        address _target,
        address _implementation,
        bytes memory _data,
        bytes32 _slot,
        uint8 _offset
    )
        external;

    function getGameImpl(GameType _gameType) external view returns (IDisputeGame);

    function makeGameArgs(
        uint256 _l2ChainId,
        IAnchorStateRegistry _anchorStateRegistry,
        IDelayedWETH _delayedWETH,
        DisputeGameConfig memory _gcfg
    )
        external
        view
        returns (bytes memory);

    function __constructor__(IOPContractsManagerContainer _contractsContainer) external;
}
