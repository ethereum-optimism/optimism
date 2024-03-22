// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/introspection/IERC165.sol";

import { IOptimismMintableERC20 } from "src/universal/IOptimismMintableERC20.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @custom:predeploy 0x4200000000000000000000000000000000000042
/// @title GovernanceToken
/// @notice The Optimism token used in governance and supporting voting and delegation. Implements
///         EIP 2612 allowing signed approvals. Contract is "owned" by a `MintManager` instance with
///         permission to the `mint` function only, for the purposes of enforcing the token
///         inflation schedule.
contract GovernanceToken is ERC20Burnable, ERC20Votes, Ownable, IOptimismMintableERC20 {
    /// @notice Constructs the GovernanceToken contract.
    constructor() ERC20("Optimism", "OP") ERC20Permit("Optimism") { }

    /// @notice Only allows the owner or the bridge to call a function. On OP Mainnet, only the
    ///         owner is allowed to call the function. On other chains, only the bridge is allowed.
    modifier onlyOwnerOrBridge() {
        if (block.chainid == 10) {
            require(msg.sender == owner(), "GovernanceToken: caller is not the owner");
        } else {
            require(msg.sender == bridge(), "GovernanceToken: caller is not the bridge");
        }
        _;
    }

    /// @inheritdoc IOptimismMintableERC20
    function remoteToken() public pure returns (address) {
        // TODO: Update with actual L1 token address.
        return 0x60FE6a0462bce7FbF1dCdD8B1Bb344E4D58B4C96;
    }

    /// @inheritdoc IOptimismMintableERC20
    function bridge() public pure returns (address) {
        return Predeploys.L2_STANDARD_BRIDGE;
    }

    /// @notice Allows the owner or the bridge to mint tokens.
    /// @param _account The account receiving minted tokens.
    /// @param _amount  The amount of tokens to mint.
    function mint(address _account, uint256 _amount) public onlyOwnerOrBridge {
        _mint(_account, _amount);
    }

    /// @notice Allows the owner or the bridge to burn tokens.
    /// @param _account The account that tokens will be burned from.
    /// @param _amount  The amount of tokens that will be burned.
    function burn(address _account, uint256 _amount) public onlyOwnerOrBridge {
        _burn(_account, _amount);
    }

    /// @notice ERC165 interface check function.
    /// @param _interfaceId Interface ID to check.
    /// @return Whether or not the interface is supported by this contract.
    function supportsInterface(bytes4 _interfaceId) external view virtual returns (bool) {
        bytes4 iface1 = type(IERC165).interfaceId;
        // Interface corresponding to IOptimismMintableERC20 (this contract).
        // Interface is not supported for OP Mainnet.
        bytes4 iface2 = type(IOptimismMintableERC20).interfaceId;
        return _interfaceId == iface1 || (block.chainid != 10 && _interfaceId == iface2);
    }

    /// @notice Callback called after a token transfer.
    /// @param from   The account sending tokens.
    /// @param to     The account receiving tokens.
    /// @param amount The amount of tokens being transfered.
    function _afterTokenTransfer(address from, address to, uint256 amount) internal override(ERC20, ERC20Votes) {
        super._afterTokenTransfer(from, to, amount);
    }

    /// @notice Internal mint function.
    /// @param to     The account receiving minted tokens.
    /// @param amount The amount of tokens to mint.
    function _mint(address to, uint256 amount) internal override(ERC20, ERC20Votes) {
        super._mint(to, amount);
    }

    /// @notice Internal burn function.
    /// @param account The account that tokens will be burned from.
    /// @param amount  The amount of tokens that will be burned.
    function _burn(address account, uint256 amount) internal override(ERC20, ERC20Votes) {
        super._burn(account, amount);
    }
}
