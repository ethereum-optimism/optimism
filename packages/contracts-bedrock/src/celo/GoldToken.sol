// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../lib/openzeppelin-contracts/contracts/access/Ownable.sol";
import "../../lib/openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";

import "./UsingRegistry.sol";
import "./CalledByVm.sol";
import "./Initializable.sol";
import "./interfaces/ICeloToken.sol";
import "./common/interfaces/ICeloVersionedContract.sol";

contract GoldToken is Initializable, CalledByVm, UsingRegistry, IERC20, ICeloToken, ICeloVersionedContract {
    // Address of the TRANSFER precompiled contract.
    // solhint-disable state-visibility
    address constant TRANSFER = address(0xff - 2);
    string constant NAME = "Celo native asset";
    string constant SYMBOL = "CELO";
    uint8 constant DECIMALS = 18;
    uint256 internal totalSupply_;
    // solhint-enable state-visibility

    mapping(address => mapping(address => uint256)) internal allowed;

    // Burn address is 0xdEaD because truffle is having buggy behaviour with the zero address
    address constant BURN_ADDRESS = address(0x000000000000000000000000000000000000dEaD);

    event TransferComment(string comment);

    /**
     * @notice Sets initialized == true on implementation contracts
     * @param test Set to true to skip implementation initialization
     */
    constructor(bool test) Initializable(test) { }

    /**
     * @notice Returns the storage, major, minor, and patch version of the contract.
     * @return Storage version of the contract.
     * @return Major version of the contract.
     * @return Minor version of the contract.
     * @return Patch version of the contract.
     */
    function getVersionNumber() external pure returns (uint256, uint256, uint256, uint256) {
        return (1, 1, 2, 0);
    }

    /**
     * @notice Used in place of the constructor to allow the contract to be upgradable via proxy.
     * @param registryAddress Address of the Registry contract.
     */
    function initialize(address registryAddress) external initializer {
        totalSupply_ = 0;
        _transferOwnership(msg.sender);
        setRegistry(registryAddress);
    }

    /**
     * @notice Transfers CELO from one address to another.
     * @param to The address to transfer CELO to.
     * @param value The amount of CELO to transfer.
     * @return True if the transaction succeeds.
     */
    // solhint-disable-next-line no-simple-event-func-name
    function transfer(address to, uint256 value) external returns (bool) {
        return _transferWithCheck(to, value);
    }

    /**
     * @notice Transfers CELO from one address to another with a comment.
     * @param to The address to transfer CELO to.
     * @param value The amount of CELO to transfer.
     * @param comment The transfer comment
     * @return True if the transaction succeeds.
     */
    function transferWithComment(address to, uint256 value, string calldata comment) external returns (bool) {
        bool succeeded = _transferWithCheck(to, value);
        emit TransferComment(comment);
        return succeeded;
    }

    /**
     * @notice This function allows a user to burn a specific amount of tokens.
     *  Burning is implemented by sending tokens to the burn address.
     * @param value: The amount of CELO to burn.
     * @return True if burn was successful.
     */
    function burn(uint256 value) external returns (bool) {
        // not using transferWithCheck as the burn address can potentially be the zero address
        return _transfer(BURN_ADDRESS, value);
    }

    /**
     * @notice Approve a user to transfer CELO on behalf of another user.
     * @param spender The address which is being approved to spend CELO.
     * @param value The amount of CELO approved to the spender.
     * @return True if the transaction succeeds.
     */
    function approve(address spender, uint256 value) external returns (bool) {
        require(spender != address(0), "cannot set allowance for 0");
        allowed[msg.sender][spender] = value;
        emit Approval(msg.sender, spender, value);
        return true;
    }

    /**
     * @notice Increases the allowance of another user.
     * @param spender The address which is being approved to spend CELO.
     * @param value The increment of the amount of CELO approved to the spender.
     * @return True if the transaction succeeds.
     */
    function increaseAllowance(address spender, uint256 value) external returns (bool) {
        require(spender != address(0), "cannot set allowance for 0");
        uint256 oldValue = allowed[msg.sender][spender];
        uint256 newValue = oldValue + value;
        allowed[msg.sender][spender] = newValue;
        emit Approval(msg.sender, spender, newValue);
        return true;
    }

    /**
     * @notice Decreases the allowance of another user.
     * @param spender The address which is being approved to spend CELO.
     * @param value The decrement of the amount of CELO approved to the spender.
     * @return True if the transaction succeeds.
     */
    function decreaseAllowance(address spender, uint256 value) external returns (bool) {
        uint256 oldValue = allowed[msg.sender][spender];
        uint256 newValue = oldValue - value;
        allowed[msg.sender][spender] = newValue;
        emit Approval(msg.sender, spender, newValue);
        return true;
    }

    /**
     * @notice Transfers CELO from one address to another on behalf of a user.
     * @param from The address to transfer CELO from.
     * @param to The address to transfer CELO to.
     * @param value The amount of CELO to transfer.
     * @return True if the transaction succeeds.
     */
    function transferFrom(address from, address to, uint256 value) external returns (bool) {
        require(to != address(0), "transfer attempted to reserved address 0x0");
        require(value <= balanceOf(from), "transfer value exceeded balance of sender");
        require(value <= allowed[from][msg.sender], "transfer value exceeded sender's allowance for spender");

        bool success;
        (success,) = TRANSFER.call{ value: 0, gas: gasleft() }(abi.encode(from, to, value));
        require(success, "CELO transfer failed");

        allowed[from][msg.sender] = allowed[from][msg.sender] - value;
        emit Transfer(from, to, value);
        return true;
    }

    /**
     * @notice Mints new CELO and gives it to 'to'.
     * @param to The account for which to mint tokens.
     * @param value The amount of CELO to mint.
     */
    function mint(address to, uint256 value) external onlyVm returns (bool) {
        if (value == 0) {
            return true;
        }

        require(to != address(0), "mint attempted to reserved address 0x0");
        totalSupply_ = totalSupply_ + value;

        bool success;
        (success,) = TRANSFER.call{ value: 0, gas: gasleft() }(abi.encode(address(0), to, value));
        require(success, "CELO transfer failed");

        emit Transfer(address(0), to, value);
        return true;
    }

    /**
     * @return The name of the CELO token.
     */
    function name() external pure returns (string memory) {
        return NAME;
    }

    /**
     * @return The symbol of the CELO token.
     */
    function symbol() external pure returns (string memory) {
        return SYMBOL;
    }

    /**
     * @return The number of decimal places to which CELO is divisible.
     */
    function decimals() external pure returns (uint8) {
        return DECIMALS;
    }

    /**
     * @return The total amount of CELO in existence, including what the burn address holds.
     */
    function totalSupply() external view returns (uint256) {
        return totalSupply_;
    }

    /**
     * @return The total amount of CELO in existence, not including what the burn address holds.
     */
    function circulatingSupply() external view returns (uint256) {
        return totalSupply_ - getBurnedAmount() - balanceOf(address(0));
    }

    /**
     * @notice Gets the amount of owner's CELO allowed to be spent by spender.
     * @param owner The owner of the CELO.
     * @param spender The spender of the CELO.
     * @return The amount of CELO owner is allowing spender to spend.
     */
    function allowance(address owner, address spender) external view returns (uint256) {
        return allowed[owner][spender];
    }

    /**
     * @notice Increases the variable for total amount of CELO in existence.
     * @param amount The amount to increase counter by
     */
    function increaseSupply(uint256 amount) external onlyVm {
        totalSupply_ = totalSupply_ + amount;
    }

    /**
     * @notice Gets the amount of CELO that has been burned.
     * @return The total amount of Celo that has been sent to the burn address.
     */
    function getBurnedAmount() public view returns (uint256) {
        return balanceOf(BURN_ADDRESS);
    }

    /**
     * @notice Gets the balance of the specified address.
     * @param owner The address to query the balance of.
     * @return The balance of the specified address.
     */
    function balanceOf(address owner) public view returns (uint256) {
        return owner.balance;
    }

    /**
     * @notice internal CELO transfer from one address to another.
     * @param to The address to transfer CELO to.
     * @param value The amount of CELO to transfer.
     * @return True if the transaction succeeds.
     */
    function _transfer(address to, uint256 value) internal returns (bool) {
        require(value <= balanceOf(msg.sender), "transfer value exceeded balance of sender");

        bool success;
        (success,) = TRANSFER.call{ value: 0, gas: gasleft() }(abi.encode(msg.sender, to, value));
        require(success, "CELO transfer failed");
        emit Transfer(msg.sender, to, value);
        return true;
    }

    /**
     * @notice Internal CELO transfer from one address to another.
     * @param to The address to transfer CELO to. Zero address will revert.
     * @param value The amount of CELO to transfer.
     * @return True if the transaction succeeds.
     */
    function _transferWithCheck(address to, uint256 value) internal returns (bool) {
        require(to != address(0), "transfer attempted to reserved address 0x0");
        return _transfer(to, value);
    }
}
