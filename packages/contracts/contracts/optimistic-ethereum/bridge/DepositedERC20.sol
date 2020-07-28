pragma solidity ^0.5.0;
import { ERC20 } from "./ERC20.sol";
import { L2ERC20Bridge } from "./L2ERC20Bridge.sol";

contract DepositedERC20 is ERC20 {


    L2ERC20Bridge public l2ERC20Bridge;
    uint public _withdrawalNonce = 0; //like a reference ID for the wdrwl
    constructor () public ERC20(10, "Jingle Wingle", 8, "JING") {
        // store the factory address which is creating us
        l2ERC20Bridge = L2ERC20Bridge(msg.sender);
    }

    /*
    * Public functions
    */
    function processDeposit(
        address _depositer,
        uint _amount
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
        //l2ERC20Bridge.forwardWithdrawal(_withdrawTo, _amount);
    }


}