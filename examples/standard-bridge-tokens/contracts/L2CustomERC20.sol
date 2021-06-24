// SPDX-License-Identifier: MIT
pragma solidity >=0.5.16 <0.8.0;

import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
//import { IL2StandardERC20 } from "@eth-optimism/contracts/libraries/standards/IL2StandardERC20.sol";
import './IL2StandardERC20.sol';

contract L2CustomERC20 is IL2StandardERC20, ERC20 {
    address public override l1Token;
    address public l2Bridge;

    /**
     * @param _l1Token Address of the corresponding L1 token.
     * @param _name ERC20 name.
     * @param _symbol ERC20 symbol.
     */
    constructor(
        address _l2Bridge,
        address _l1Token,
        string memory _name,
        string memory _symbol
    )
        ERC20(_name, _symbol) {
        l1Token = _l1Token;
        l2Bridge = _l2Bridge;
    }

    modifier onlyL2Bridge {
        require(msg.sender == l2Bridge, "Only L2 Bridge can mint and burn");
        _;
    }

    function supportsInterface(bytes4 _interfaceId) public override pure returns (bool) {
        // 0x01ffc9a7 = bytes4(keccak256("supportsInterface(bytes4)")) (ERC165)
        // 0x1d1d8b63 = bytes4(keccak256("l1Token()")) ^ bytes4(keccak256("mint(address,uint256)")) ^ bytes4(keccak256("burn(address,uint256)"))
        return _interfaceId == 0x01ffc9a7 || _interfaceId == 0x1d1d8b63;
    }

    function mint(address _to, uint256 _amount) public override onlyL2Bridge {
        _mint(_to, _amount);

        emit Mint(_to, _amount);
    }

    function burn(address _from, uint256 _amount) public override onlyL2Bridge {
        _burn(_from, _amount);

        emit Burn(_from, _amount);
    }
}
