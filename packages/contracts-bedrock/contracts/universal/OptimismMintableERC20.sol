// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "./SupportedInterfaces.sol";

/**
 * @title OptimismMintableERC20
 * @notice OptimismMintableERC20 is a standard extension of the base ERC20 token contract designed
 *         to allow the StandardBridge contracts to mint and burn tokens. This makes it possible to
 *         use an OptimismMintablERC20 as the L2 representation of an L1 token, or vice-versa.
 *         Designed to be backwards compatible with the older StandardL2ERC20 token which was only
 *         meant for use on L2.
 */
contract OptimismMintableERC20 is ERC20 {
    /**
     * @notice Emitted whenever tokens are minted for an account.
     *
     * @param account Address of the account tokens are being minted for.
     * @param amount  Amount of tokens minted.
     */
    event Mint(address indexed account, uint256 amount);

    /**
     * @notice Emitted whenever tokens are burned from an account.
     *
     * @param account Address of the account tokens are being burned from.
     * @param amount  Amount of tokens burned.
     */
    event Burn(address indexed account, uint256 amount);

    /**
     * @notice Address of the corresponding version of this token on the remote chain.
     */
    address public remoteToken;

    /**
     * @notice Address of the StandardBridge on this network.
     */
    address public bridge;

    /**
     * @param _bridge      Address of the L2 standard bridge.
     * @param _remoteToken Address of the corresponding L1 token.
     * @param _name        ERC20 name.
     * @param _symbol      ERC20 symbol.
     */
    constructor(
        address _bridge,
        address _remoteToken,
        string memory _name,
        string memory _symbol
    ) ERC20(_name, _symbol) {
        remoteToken = _remoteToken;
        bridge = _bridge;
    }

    /**
     * @custom:legacy
     * @notice Legacy getter for the remote token. Use remoteToken going forward.
     */
    function l1Token() public view returns (address) {
        return remoteToken;
    }

    /**
     * @custom:legacy
     * @notice Legacy getter for the bridge. Use bridge going forward.
     */
    function l2Bridge() public view returns (address) {
        return bridge;
    }

    /**
     * @notice A modifier that only allows the bridge to call
     */
    modifier onlyBridge() {
        require(msg.sender == bridge, "OptimismMintableERC20: only bridge can mint and burn");
        _;
    }

    /**
     * @notice ERC165 interface check function.
     *
     * @param _interfaceId Interface ID to check.
     *
     * @return Whether or not the interface is supported by this contract.
     */
    function supportsInterface(bytes4 _interfaceId) external pure returns (bool) {
        bytes4 iface1 = type(IERC165).interfaceId;
        bytes4 iface2 = type(IL1Token).interfaceId;
        bytes4 iface3 = type(IRemoteToken).interfaceId;
        return _interfaceId == iface1 || _interfaceId == iface2 || _interfaceId == iface3;
    }

    /**
     * @notice Allows the StandardBridge on this network to mint tokens.
     *
     * @param _to     Address to mint tokens to.
     * @param _amount Amount of tokens to mint.
     */
    function mint(address _to, uint256 _amount) external virtual onlyBridge {
        _mint(_to, _amount);
        emit Mint(_to, _amount);
    }

    /**
     * @notice Allows the StandardBridge on this network to burn tokens.
     *
     * @param _from   Address to burn tokens from.
     * @param _amount Amount of tokens to burn.
     */
    function burn(address _from, uint256 _amount) external virtual onlyBridge {
        _burn(_from, _amount);
        emit Burn(_from, _amount);
    }
}
