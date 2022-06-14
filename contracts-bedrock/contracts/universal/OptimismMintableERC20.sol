// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "./SupportedInterfaces.sol";

/**
 * @title OptimismMintableERC20
 * This contract represents the remote representation
 * of an ERC20 token. It is linked to the address of
 * a token in another domain and tokens can be locked
 * in the StandardBridge which will mint tokens in the
 * other domain.
 */
contract OptimismMintableERC20 is ERC20 {
    event Mint(address indexed _account, uint256 _amount);
    event Burn(address indexed _account, uint256 _amount);

    /**
     * @notice The address of the token in the remote domain
     */
    address public remoteToken;

    /**
     * @notice The address of the bridge responsible for
     * minting. It is in the same domain.
     */
    address public bridge;

    /**
     * @param _bridge Address of the L2 standard bridge.
     * @param _remoteToken Address of the corresponding L1 token.
     * @param _name ERC20 name.
     * @param _symbol ERC20 symbol.
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
     * @notice Returns the corresponding L1 token address.
     * This is a legacy function and wraps the remoteToken value.
     */
    function l1Token() public view returns (address) {
        return remoteToken;
    }

    /**
     * @notice The address of the bridge contract
     * responsible for minting tokens. This is a legacy
     * getter function
     */
    function l2Bridge() public view returns (address) {
        return bridge;
    }

    /**
     * @notice A modifier that only allows the bridge to call
     */
    modifier onlyBridge() {
        require(msg.sender == bridge, "Only L2 Bridge can mint and burn");
        _;
    }

    /**
     * @notice ERC165
     */
    // slither-disable-next-line external-function
    function supportsInterface(bytes4 _interfaceId) public pure returns (bool) {
        bytes4 iface1 = type(IERC165).interfaceId;
        bytes4 iface2 = type(IL1Token).interfaceId;
        bytes4 iface3 = type(IRemoteToken).interfaceId;
        return _interfaceId == iface1 || _interfaceId == iface2 || _interfaceId == iface3;
    }

    /**
     * @notice The bridge can mint tokens
     */
    // slither-disable-next-line external-function
    function mint(address _to, uint256 _amount) public virtual onlyBridge {
        _mint(_to, _amount);

        emit Mint(_to, _amount);
    }

    /**
     * @notice The bridge can burn tokens
     */
    // slither-disable-next-line external-function
    function burn(address _from, uint256 _amount) public virtual onlyBridge {
        _burn(_from, _amount);

        emit Burn(_from, _amount);
    }
}
