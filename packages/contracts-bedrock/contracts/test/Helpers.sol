//SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ERC20 } from "@rari-capital/solmate/src/tokens/ERC20.sol";
import { ERC721 } from "@rari-capital/solmate/src/tokens/ERC721.sol";

contract TestERC20 is ERC20 {
    constructor() ERC20("TEST", "TST", 18) {}

    function mint(address to, uint256 value) public {
        _mint(to, value);
    }
}

contract TestERC721 is ERC721 {
    constructor() ERC721("TEST", "TST") {}

    function mint(address to, uint256 tokenId) public {
        _mint(to, tokenId);
    }

    function tokenURI(uint256) public pure virtual override returns (string memory) {}
}

contract CallRecorder {
    struct CallInfo {
        address sender;
        bytes data;
        uint256 gas;
        uint256 value;
    }

    CallInfo public lastCall;

    function record() public payable {
        lastCall.sender = msg.sender;
        lastCall.data = msg.data;
        lastCall.gas = gasleft();
        lastCall.value = msg.value;
    }
}

contract Reverter {
    function doRevert() public pure {
        revert("Reverter reverted");
    }
}

contract SimpleStorage {
    mapping(bytes32 => bytes32) public db;

    function set(bytes32 _key, bytes32 _value) public payable {
        db[_key] = _value;
    }

    function get(bytes32 _key) public view returns (bytes32) {
        return db[_key];
    }
}
