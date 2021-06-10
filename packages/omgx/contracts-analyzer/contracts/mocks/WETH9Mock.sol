// SPDX-License-Identifier: MIT

pragma solidity 0.6.12;

// CHANGE_OMGX
// contract WETH9Mock {
// 	string public name = "Wrapped Ether";
// 	string public symbol = "WETH";
// 	uint8 public decimals = 18;

// 	event Approval(address indexed src, address indexed guy, uint256 wad);
// 	event Transfer(address indexed src, address indexed dst, uint256 wad);
// 	event Deposit(address indexed dst, uint256 wad);
// 	event Withdrawal(address indexed src, uint256 wad);

// 	mapping(address => uint256) public balanceOf;
// 	mapping(address => mapping(address => uint256)) public allowance;

// 	function deposit() public payable {
// 		balanceOf[msg.sender] += msg.value;
// 		emit Deposit(msg.sender, msg.value);
// 	}

// 	function withdraw(uint256 wad) public {
// 		require(balanceOf[msg.sender] >= wad, "WETH9: Error");
// 		balanceOf[msg.sender] -= wad;
// 		msg.sender.transfer(wad);
// 		emit Withdrawal(msg.sender, wad);
// 	}

// 	function totalSupply() public view returns (uint256) {
// 		return address(this).balance;
// 	}

// 	function approve(address guy, uint256 wad) public returns (bool) {
// 		allowance[msg.sender][guy] = wad;
// 		emit Approval(msg.sender, guy, wad);
// 		return true;
// 	}

// 	function transfer(address dst, uint256 wad) public returns (bool) {
// 		return transferFrom(msg.sender, dst, wad);
// 	}

// 	function transferFrom(
// 		address src,
// 		address dst,
// 		uint256 wad
// 	) public returns (bool) {
// 		require(balanceOf[src] >= wad, "WETH9: Error");

// 		if (src != msg.sender && allowance[src][msg.sender] != uint256(-1)) {
// 			require(allowance[src][msg.sender] >= wad, "WETH9: Error");
// 			allowance[src][msg.sender] -= wad;
// 		}

// 		balanceOf[src] -= wad;
// 		balanceOf[dst] += wad;

// 		emit Transfer(src, dst, wad);

// 		return true;
// 	}
// }