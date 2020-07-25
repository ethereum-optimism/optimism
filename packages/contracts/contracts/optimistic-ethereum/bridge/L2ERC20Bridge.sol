pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;
import { ERC20 } from "../../test-helpers/ERC20.sol";

contract L2ERC20Bridge is ERC20 {


    address public l2BridgeFactoryAddress;
    uint public withdrawalNonce = 0; //like a reference ID for the wdrwl
    constructor () public ERC20(10, "Jingle Wingle", 8, "JING") {
        // store the factory address which is creating us
        l2BridgeFactoryAddress = msg.sender;
    }




}