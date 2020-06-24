pragma solidity ^0.5.1;

// Repurposed from https://github.com/OpenZeppelin/openzeppelin-contracts/blob/master/contracts/token/ERC20/ERC20.sol
contract TestToken {
  mapping (address => uint256) private _balances;
  uint256 private _totalSupply;
  bytes32 _meaninglessHash;

  event Transfer(
    address indexed from,
    address indexed to,
    uint256 indexed amount,
    uint256 block
  );

  constructor(
    uint256 totalSupply
  ) public {
    _totalSupply = totalSupply;
    _balances[msg.sender] = _totalSupply;
  }

  function totalSupply() public view returns (uint256) {
    return _totalSupply;
  }

  function balanceOf(address account) public view returns (uint256) {
    return _balances[account];
  }

  function transfer(address sender, address recipient, uint256 amount) public {
    require(sender != address(0), "ERC20: transfer from the zero address");
    require(recipient != address(0), "ERC20: transfer to the zero address");
    require(_balances[sender] >= amount, "Insufficient sender balance");

    _balances[sender] -= amount;
    _balances[recipient] += amount;
    emit Transfer(sender, recipient, amount, block.timestamp);
  }

  function updateMeaninglessHash(bytes memory input) public {
    _meaninglessHash = keccak256(input);
  }
}