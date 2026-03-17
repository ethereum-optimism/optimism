// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { OptimismMintableERC721 } from "src/L2/OptimismMintableERC721.sol";
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @notice Legacy mapping storage layout for OptimismMintableERC721Factory.
contract OptimismMintableERC721FactoryLegacyMapping {
    /// @notice Tracks addresses created by this factory.
    mapping(address => bool) public isOptimismMintableERC721;
}

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000017
/// @title OptimismMintableERC721Factory
/// @notice Factory contract for creating OptimismMintableERC721 contracts.
contract OptimismMintableERC721Factory is
    ProxyAdminOwnedBase,
    ISemver,
    OptimismMintableERC721FactoryLegacyMapping,
    Initializable
{
    /// @notice Address of the ERC721 bridge on this network.
    /// @custom:network-specific
    address public bridge;

    /// @notice Chain ID for the remote network.
    /// @custom:network-specific
    uint256 public remoteChainID;

    /// @notice Reserve extra slots in the storage layout for future upgrades.
    ///         A gap size of 46 was chosen here, so that the first slot used in a child contract
    ///         would be a multiple of 50.
    uint256[46] private __gap;

    /// @notice Emitted whenever a new OptimismMintableERC721 contract is created.
    /// @param localToken  Address of the token on the this domain.
    /// @param remoteToken Address of the token on the remote domain.
    /// @param deployer    Address of the initiator of the deployment
    event OptimismMintableERC721Created(address indexed localToken, address indexed remoteToken, address deployer);

    /// @notice Semantic version.
    /// @custom:semver 1.5.1
    string public constant version = "1.5.1";

    /// @notice Constructs the OptimismMintableERC721Factory contract.
    constructor() {
        _disableInitializers();
    }

    /// @notice Initializes the contract.
    /// @param _bridge Address of the ERC721 bridge on this network.
    /// @param _remoteChainID Chain ID for the remote network.
    function initialize(address _bridge, uint256 _remoteChainID) external initializer {
        _assertOnlyProxyAdminOrProxyAdminOwner();
        bridge = _bridge;
        remoteChainID = _remoteChainID;
    }

    /// @notice Getter function for the address of the ERC721 bridge on this network.
    ///         Public getter is legacy and will be removed in the future. Use `bridge` instead.
    /// @return Address of the ERC721 bridge on this network.
    /// @custom:legacy
    function BRIDGE() external view returns (address) {
        return bridge;
    }

    /// @notice Getter function for the chain ID of the remote network.
    ///         Public getter is legacy and will be removed in the future. Use `remoteChainID` instead.
    /// @return Chain ID for the remote network.
    /// @custom:legacy
    function REMOTE_CHAIN_ID() external view returns (uint256) {
        return remoteChainID;
    }

    /// @notice Creates an instance of the standard ERC721.
    /// @param _remoteToken Address of the corresponding token on the other domain.
    /// @param _name        ERC721 name.
    /// @param _symbol      ERC721 symbol.
    function createOptimismMintableERC721(
        address _remoteToken,
        string memory _name,
        string memory _symbol
    )
        external
        returns (address)
    {
        require(_remoteToken != address(0), "OptimismMintableERC721Factory: L1 token address cannot be address(0)");

        bytes32 salt = keccak256(abi.encode(_remoteToken, _name, _symbol));
        address localToken =
            address(new OptimismMintableERC721{ salt: salt }(bridge, remoteChainID, _remoteToken, _name, _symbol));

        isOptimismMintableERC721[localToken] = true;
        emit OptimismMintableERC721Created(localToken, _remoteToken, msg.sender);

        return localToken;
    }
}
