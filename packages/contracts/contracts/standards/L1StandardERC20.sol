// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "./IL1StandardERC20.sol";

contract L1StandardERC20 is IL1StandardERC20, ERC20 {
    address public l2Token;
    address public l1Bridge;

    /**
     * @param _l1Bridge Address of the L1 standard bridge.
     * @param _l2Token Address of the corresponding L2 token.
     * @param _name ERC20 name.
     * @param _symbol ERC20 symbol.
     */
    constructor(
        address _l1Bridge,
        address _l2Token,
        string memory _name,
        string memory _symbol
    ) ERC20(_name, _symbol) {
        l2Token = _l2Token;
        l1Bridge = _l1Bridge;
    }

    modifier onlyL2Bridge() {
        require(msg.sender == l1Bridge, "Only L2 Bridge can mint and burn");
        _;
    }

    // slither-disable-next-line external-function
    function supportsInterface(bytes4 _interfaceId) public pure returns (bool) {
        bytes4 firstSupportedInterface = bytes4(keccak256("supportsInterface(bytes4)")); // ERC165
        bytes4 secondSupportedInterface = IL1StandardERC20.l2Token.selector ^
            IL1StandardERC20.mint.selector ^
            IL1StandardERC20.burn.selector;
        return _interfaceId == firstSupportedInterface || _interfaceId == secondSupportedInterface;
    }

    // slither-disable-next-line external-function
    function mint(address _to, uint256 _amount) public virtual onlyL2Bridge {
        _mint(_to, _amount);

        emit Mint(_to, _amount);
    }

    // slither-disable-next-line external-function
    function burn(address _from, uint256 _amount) public virtual onlyL2Bridge {
        _burn(_from, _amount);

        emit Burn(_from, _amount);
    }
}
