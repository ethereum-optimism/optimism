pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;
import { ERC20 } from "./ERC20.sol";
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { MockL1ToL2MessagePasser } from "../ovm/test-helpers/MockL1ToL2MessagePasser.sol";

//contract L1ERC20Bridge is ContractResolver {
contract L1ERC20Bridge {

    //mapping (address => address) public correspondingDepositedERC20;
    uint depositNonce = 0;
    address l2ERC20BridgeAddress;
    address l1ToL2MessagePasser;

    constructor(
        address _l2ERC20BridgeAddress,
        address _l1ToL2MessagePasser
    ) public {
        l2ERC20BridgeAddress = _l2ERC20BridgeAddress;
        l1ToL2MessagePasser = _l1ToL2MessagePasser;
    }

    // constructor(
    //     address _addressResolver
    // ) public
    // ContractResolver(_addressResolver){
    //     ROLLUP_CONTRACT = resolveContract("ROLLUP_CONTRACT");
    // }


    function initializeDeposit(
        address _L1ERC20Address,
        address _depositer,
        uint _amount
    ) public {
        // Transfer deposit funds to this contract
        ERC20(_L1ERC20Address).transferFrom(
            _depositer,
            address(this),
            _amount
        );

        bytes memory messageData = abi.encodeWithSignature(
            "processDeposit(address,uint,uint)",
            _depositer,
            _amount,
            depositNonce
        );

                // Tell L2 to mint corresponding coins
        MockL1ToL2MessagePasser(l1ToL2MessagePasser).passMessageToL2(messageData);

        // ROLLUP_CONTRACT.sendL1ToL2Message(
        //     messageData,
        //     _L2ERC20BridgeAddress
        // );

    }

}