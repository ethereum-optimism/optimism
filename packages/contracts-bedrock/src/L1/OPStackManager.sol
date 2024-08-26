// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Unauthorized } from "src/libraries/errors/CommonErrors.sol";

import { ISemver } from "src/universal/ISemver.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";

import { AddressManager } from "src/legacy/AddressManager.sol";
import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";

import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

/// @custom:proxied TODO this is not proxied yet.
contract OPStackManager is ISemver {
    // -------- Structs --------

    /// @notice Represents the roles that can be set when deploying a standard OP Stack chain.
    struct Roles {
        address opChainProxyAdminOwner;
        address systemConfigOwner;
        address batcher;
        address unsafeBlockSigner;
        address proposer;
        address challenger;
    }

    /// @notice The full set of inputs to deploy a new OP Stack chain.
    struct DeployInput {
        Roles roles;
        uint32 basefeeScalar;
        uint32 blobBasefeeScalar;
        uint256 l2ChainId;
    }

    /// @notice The full set of outputs from deploying a new OP Stack chain.
    struct DeployOutput {
        ProxyAdmin opChainProxyAdmin;
        AddressManager addressManager;
        L1ERC721Bridge l1ERC721BridgeProxy;
        SystemConfig systemConfigProxy;
        OptimismMintableERC20Factory optimismMintableERC20FactoryProxy;
        L1StandardBridge l1StandardBridgeProxy;
        L1CrossDomainMessenger l1CrossDomainMessengerProxy;
        // Fault proof contracts below.
        OptimismPortal2 optimismPortalProxy;
        DisputeGameFactory disputeGameFactoryProxy;
        DisputeGameFactory disputeGameFactoryImpl;
        AnchorStateRegistry anchorStateRegistryProxy;
        AnchorStateRegistry anchorStateRegistryImpl;
        FaultDisputeGame faultDisputeGame;
        PermissionedDisputeGame permissionedDisputeGame;
        DelayedWETH delayedWETHPermissionedGameProxy;
        DelayedWETH delayedWETHPermissionlessGameProxy;
    }

    /// @notice The logic address and initializer selector for an implementation contract.
    struct Implementation {
        address logic; // Address containing the deployed logic contract.
        bytes4 initializer; // Function selector for the initializer.
    }

    /// @notice Used to set the implementation for a contract by mapping a contract
    /// name to the implementation data.
    struct ImplementationSetter {
        string name; // Contract name.
        Implementation info; // Implementation to set.s
    }

    // -------- Constants and Variables --------

    /// @custom:semver 1.0.0-beta.2
    string public constant version = "1.0.0-beta.2";

    /// @notice The user who can release new versions of the OP Stack contracts.
    address public immutable releaseManager;

    /// @notice The latest release of the OP Stack Manager, as a string of the format `op-contracts/vX.Y.Z`.
    string public latestVersion;

    /// @notice Maps a release version to a contract name to it's implementation data.
    mapping(string => mapping(string => Implementation)) public implementations;

    /// @notice Maps an L2 Chain ID to the SystemConfig for that chain.
    mapping(uint256 => SystemConfig) public systemConfigs;

    // -------- Events --------

    /// @notice Emitted when a new OP Stack chain is deployed.
    /// @param l2ChainId The chain ID of the new chain.
    /// @param systemConfig The address of the new chain's SystemConfig contract.
    event Deployed(uint256 indexed l2ChainId, SystemConfig indexed systemConfig);

    // -------- Errors --------

    /// @notice Thrown when a release version is already set.
    error AlreadyReleased();

    /// @notice Thrown when an invalid `l2ChainId` is provided to `deploy`.
    error InvalidChainId();

    /// @notice Thrown when a role's address is not valid.
    error InvalidRoleAddress(string role);

    /// @notice Thrown when a deployment fails.
    error DeploymentFailed(string reason);

    /// @notice Temporary error since the deploy method is not yet implemented.
    error NotImplemented();

    // -------- Methods --------

    /// @notice OPSM is intended to be proxied when used in production. Since we are initially
    /// focused on an OPSM version that unblocks interop, we are not proxying OPSM for simplicity.
    /// Later, we will `_disableInitializers` in the constructor and replace any constructor logic
    /// with an `initialize` function, which will be a breaking change to the OPSM interface.
    constructor(address _releaseManager, string memory _latestVersion) {
        releaseManager = _releaseManager;
        latestVersion = _latestVersion;
    }

    /// @notice Callable by the OPSM owner to release a set of implementation contracts for a given
    /// release version.
    /// @param _version The release version to set implementations for, of the format `op-contracts/vX.Y.Z`.
    /// @param _isLatest Whether the release version is the latest released version. This is
    /// significant because the latest version is used to deploy chains in the `deploy` function.
    /// @param _setters The set of implementations to set for the release version.
    function setRelease(string memory _version, bool _isLatest, ImplementationSetter[] calldata _setters) external {
        if (msg.sender != releaseManager) revert Unauthorized();

        if (_isLatest) latestVersion = _version;

        for (uint256 i = 0; i < _setters.length; i++) {
            ImplementationSetter calldata setter = _setters[i];
            Implementation storage impl = implementations[version][setter.name];
            if (impl.logic != address(0)) revert AlreadyReleased();

            impl.initializer = setter.info.initializer;
            impl.logic = setter.info.logic;
        }
    }

    function deploy(DeployInput calldata _input) external view returns (DeployOutput memory output_) {
        assertValidInputs(_input);

        // Silence compiler warnings.
        _input;
        output_;

        revert NotImplemented();
    }

    /// @notice Verifies that all inputs are valid and reverts if any are invalid.
    /// Typically the proxy admin owner is expected to have code, but this is not enforced here.
    function assertValidInputs(DeployInput calldata _input) internal view {
        if (_input.l2ChainId == 0 || _input.l2ChainId == block.chainid) revert InvalidChainId();

        if (_input.roles.opChainProxyAdminOwner == address(0)) revert InvalidRoleAddress("opChainProxyAdminOwner");
        if (_input.roles.systemConfigOwner == address(0)) revert InvalidRoleAddress("systemConfigOwner");
        if (_input.roles.batcher == address(0)) revert InvalidRoleAddress("batcher");
        if (_input.roles.unsafeBlockSigner == address(0)) revert InvalidRoleAddress("unsafeBlockSigner");
        if (_input.roles.proposer == address(0)) revert InvalidRoleAddress("proposer");
        if (_input.roles.challenger == address(0)) revert InvalidRoleAddress("challenger");
    }

    /// @notice Maps an L2 chain ID to an L1 batch inbox address as defined by the standard
    /// configuration's convention. This convention is `versionByte || keccak256(bytes32(chainId))[:19]`,
    /// where || denotes concatenation`, versionByte is 0x00, and chainId is a uint256.
    /// https://specs.optimism.io/protocol/configurability.html#consensus-parameters
    function chainIdToBatchInboxAddress(uint256 _l2ChainId) internal pure returns (address) {
        bytes1 versionByte = 0x00;
        bytes32 hashedChainId = keccak256(bytes.concat(bytes32(_l2ChainId)));
        bytes19 first19Bytes = bytes19(hashedChainId);
        return address(uint160(bytes20(bytes.concat(versionByte, first19Bytes))));
    }
}
