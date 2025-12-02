// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { SafeSend } from "src/universal/SafeSend.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { INativeAssetLiquidity } from "interfaces/L2/INativeAssetLiquidity.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @custom:proxied true
/// @custom:predeploy 0x420000000000000000000000000000000000002A
/// @title LiquidityController
/// @notice The LiquidityController contract is responsible for controlling the liquidity of the native asset on the L2
///         chain.
contract LiquidityController is ISemver, Initializable, OwnableUpgradeable {
    /// @notice Emitted when an address is authorized to mint/burn liquidity
    /// @param minter The address that was authorized
    event MinterAuthorized(address indexed minter);

    /// @notice Emitted when an address is deauthorized to mint/burn liquidity
    /// @param minter The address that was deauthorized
    event MinterDeauthorized(address indexed minter);

    /// @notice Emitted when liquidity is minted
    /// @param minter The address that minted the liquidity
    /// @param to The address that received the minted liquidity
    /// @param amount The amount of liquidity that was minted
    event LiquidityMinted(address indexed minter, address indexed to, uint256 amount);

    /// @notice Emitted when liquidity is burned
    /// @param minter The address that burned the liquidity
    /// @param amount The amount of liquidity that was burned
    event LiquidityBurned(address indexed minter, uint256 amount);

    /// @notice Error for when an address is unauthorized to perform liquidity control operations
    error LiquidityController_Unauthorized();

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Mapping of addresses authorized to control liquidity operations
    mapping(address => bool) public minters;

    /// @notice The name of the native asset
    string public gasPayingTokenName;

    /// @notice The symbol of the native asset
    string public gasPayingTokenSymbol;

    constructor() {
        _disableInitializers();
    }

    /// @notice Initializer.
    /// @param _owner The owner of the LiquidityController
    /// @param _gasPayingTokenName The name of the native asset
    /// @param _gasPayingTokenSymbol The symbol of the native asset
    function initialize(
        address _owner,
        string memory _gasPayingTokenName,
        string memory _gasPayingTokenSymbol
    )
        external
        initializer
    {
        __Ownable_init();
        transferOwnership(_owner);

        gasPayingTokenName = _gasPayingTokenName;
        gasPayingTokenSymbol = _gasPayingTokenSymbol;
    }

    /// @notice Authorizes an address to perform liquidity control operations
    /// @param _minter The address to authorize as a minter
    function authorizeMinter(address _minter) external onlyOwner {
        minters[_minter] = true;
        emit MinterAuthorized(_minter);
    }

    /// @notice Deauthorizes an address from performing liquidity control operations
    /// @param _minter The address to deauthorize as a minter
    function deauthorizeMinter(address _minter) external onlyOwner {
        delete minters[_minter];
        emit MinterDeauthorized(_minter);
    }

    /// @notice Mints native asset liquidity and sends it to a specified address
    /// @param _to The address to receive the minted native asset
    /// @param _amount The amount of native asset to mint and send
    function mint(address _to, uint256 _amount) external {
        if (!minters[msg.sender]) revert LiquidityController_Unauthorized();
        INativeAssetLiquidity(Predeploys.NATIVE_ASSET_LIQUIDITY).withdraw(_amount);

        // This is a forced native asset send to the recipient, the recipient should NOT expect to be called
        new SafeSend{ value: _amount }(payable(_to));

        emit LiquidityMinted(msg.sender, _to, _amount);
    }

    /// @notice Burns native asset liquidity by sending that native asset to the NativeAssetLiquidity contract
    function burn() external payable {
        if (!minters[msg.sender]) revert LiquidityController_Unauthorized();
        INativeAssetLiquidity(Predeploys.NATIVE_ASSET_LIQUIDITY).deposit{ value: msg.value }();

        emit LiquidityBurned(msg.sender, msg.value);
    }
}
