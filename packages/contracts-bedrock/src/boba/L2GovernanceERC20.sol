// SPDX-License-Identifier: MIT
pragma solidity >0.7.5;

import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { ERC20Permit } from "@openzeppelin/contracts/token/ERC20/extensions/draft-ERC20Permit.sol";
// prettier-ignore
import { ERC20Votes } from "@openzeppelin/contracts/token/ERC20/extensions/regenesis/ERC20VotesRegenesis.sol";
// prettier-ignore
import { ERC20VotesComp } from "@openzeppelin/contracts/token/ERC20/extensions/regenesis/ERC20VotesCompRegenesis.sol";
import { ILegacyMintableERC20 } from "../universal/OptimismMintableERC20.sol";

contract L2GovernanceERC20 is ILegacyMintableERC20, ERC20, ERC20Permit, ERC20Votes, ERC20VotesComp {
    // slither-disable-start immutable-states
    address public l1Token;
    address public l2Bridge;
    // slither-disable-end immutable-states
    // slither-disable-next-line too-many-digits
    uint224 public constant maxSupply = 500000000e18; // 500 million BOBA
    uint8 private immutable _decimals;

    event Mint(address indexed _account, uint256 _amount);
    event Burn(address indexed _account, uint256 _amount);

    /**
     * @param _l2Bridge Address of the L2 standard bridge.
     * @param _l1Token Address of the corresponding L1 token.
     * @param _name ERC20 name.
     * @param _symbol ERC20 symbol.
     */
    constructor(
        address _l2Bridge,
        address _l1Token,
        string memory _name,
        string memory _symbol,
        uint8 decimals_
    )
        ERC20(_name, _symbol)
        ERC20Permit(_name)
    {
        l1Token = _l1Token;
        l2Bridge = _l2Bridge;
        _decimals = decimals_;
    }

    modifier onlyL2Bridge() {
        require(msg.sender == l2Bridge, "Only L2 Bridge can mint and burn");
        _;
    }

    function decimals() public view virtual override returns (uint8) {
        return _decimals;
    }

    function supportsInterface(bytes4 _interfaceId) public pure returns (bool) {
        bytes4 firstSupportedInterface = bytes4(keccak256("supportsInterface(bytes4)")); // ERC165
        bytes4 secondSupportedInterface = ILegacyMintableERC20.l1Token.selector ^ ILegacyMintableERC20.mint.selector
            ^ ILegacyMintableERC20.burn.selector;
        return _interfaceId == firstSupportedInterface || _interfaceId == secondSupportedInterface;
    }

    function mint(address _to, uint256 _amount) public virtual onlyL2Bridge {
        _mint(_to, _amount);

        emit Mint(_to, _amount);
    }

    function burn(address _from, uint256 _amount) public virtual onlyL2Bridge {
        _burn(_from, _amount);

        emit Burn(_from, _amount);
    }

    // Overrides required by Solidity
    function _mint(address _to, uint256 _amount) internal override(ERC20, ERC20Votes) {
        super._mint(_to, _amount);
    }

    function _burn(address _account, uint256 _amount) internal override(ERC20, ERC20Votes) {
        super._burn(_account, _amount);
    }

    function _afterTokenTransfer(address from, address to, uint256 amount) internal override(ERC20, ERC20Votes) {
        super._afterTokenTransfer(from, to, amount);
    }

    function _maxSupply() internal pure override(ERC20Votes, ERC20VotesComp) returns (uint224) {
        return maxSupply;
    }
}
