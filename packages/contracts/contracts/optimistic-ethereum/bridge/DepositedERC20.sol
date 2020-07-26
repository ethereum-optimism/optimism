pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;
import { ERC20 } from "./ERC20.sol";

contract DepositedERC20 is ERC20 {


    address public l2ERC20BridgeAddress;
    uint public withdrawalNonce = 0; //like a reference ID for the wdrwl
    constructor () public ERC20(10, "Jingle Wingle", 8, "JING") {
        // store the factory address which is creating us
        l2ERC20BridgeAddress = msg.sender;
    }

    /*
    * Public functions
    */
    function processDeposit(
            address _depositer,
            uint _amount
        ) public {
            //Only the creator of this contract can authenticate deposits
            require(msg.sender == l2ERC20BridgeAddress, "Get outta here. L2 factory bridge address ONLY.");
            _mint(_depositer, _amount); // inherited mint function
        }


}