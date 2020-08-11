pragma solidity ^0.5.0;
import { ERC20 } from "./ERC20.sol";
import { L2ERC20Bridge } from "./L2ERC20Bridge.sol";

contract DepositedERC20 is ERC20 {

    L2ERC20Bridge public l2ERC20Bridge;

    //Todo: change jingle wingle
    constructor () public ERC20(10, "Jingle Wingle", 8, "JING") {
        l2ERC20Bridge = L2ERC20Bridge(msg.sender);
    }

    /*
    * Public functions
    */
    function processDeposit(
        address _depositer,
        uint _amount
        //uint _depositNonce
    ) public {
        //Only the creator of this contract can authenticate deposits
        require(msg.sender == address(l2ERC20Bridge), "Get outta here. L2 factory bridge address ONLY.");
        _mint(_depositer, _amount);
    }

    function initializeWithdrawal(
        address _withdrawTo,
        uint _amount
    ) public {
        _burn(msg.sender, _amount);
        l2ERC20Bridge.forwardWithdrawal(_withdrawTo, _amount);
    }

    function returnMS() public view returns(address){
        return msg.sender;
    }
}