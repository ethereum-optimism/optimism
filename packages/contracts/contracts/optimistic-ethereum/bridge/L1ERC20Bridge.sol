pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;
import { ERC20 } from "./ERC20.sol";
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { MockL1ToL2MessagePasser } from "../ovm/test-helpers/MockL1ToL2MessagePasser.sol";

//contract L1ERC20Bridge is ContractResolver {
contract L1ERC20Bridge {

    address public l2ERC20BridgeAddress;
    address public l1ToL2MessagePasser;
    //uint public depositNonce;

    constructor(
        address _l1ToL2MessagePasser
    ) public {
        l1ToL2MessagePasser = _l1ToL2MessagePasser;
    }

    function setCorrespondingL2BridgeAddress(
        address _l2ERC20BridgeAddress
    ) public {
        // Make sure the address has not been set yet, so this can only be done once
        require(
            l2ERC20BridgeAddress==address(0),
            "This address has already been set."
        );
        l2ERC20BridgeAddress = _l2ERC20BridgeAddress;
    }

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
            _amount
        );
        // Tell L2 to mint corresponding coins
        MockL1ToL2MessagePasser(l1ToL2MessagePasser).passMessageToL2(messageData);
    }
}