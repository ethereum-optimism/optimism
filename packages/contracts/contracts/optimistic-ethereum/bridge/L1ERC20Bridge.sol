pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;
import { ERC20 } from "./ERC20.sol";
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { MockL1ToL2MessagePasser } from "../ovm/test-helpers/MockL1ToL2MessagePasser.sol";
import { DataTypes } from "../utils/libraries/DataTypes.sol";

//contract L1ERC20Bridge is ContractResolver {
contract L1ERC20Bridge {

    address public l2ERC20BridgeAddress;
    address public l1ToL2MessagePasser;
    mapping (address => bytes32) public redeemedWithdrawals;

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
            "forwardDeposit(address,uint)",
            _depositer,
            _amount
        );
        // Tell L2 to mint corresponding coins
        MockL1ToL2MessagePasser(l1ToL2MessagePasser).passMessageToL2(messageData);
    }

    function redeemWithdrawal(
        DataTypes.Withdrawal memory _withdrawal
    ) public returns(bool) {
        address withdrawTo = _withdrawal.withdrawTo;
        uint amount =_withdrawal.amount;
        address l1ERC20Address = _withdrawal.l1ERC20Address;

        // If the withrawal is permitted, this is what should've been sent.
        bytes memory withdrawalData = abi.encode(
            _withdrawal.withdrawTo,
            _withdrawal.amount,
            _withdrawal.l1ERC20Address,
            _withdrawal.nonce
        );

        bytes32 withdrawalHash = keccak256(withdrawalData);

        require(
            redeemedWithdrawals[l1ERC20Address].length ==0,
            "Withdrawal has already been redeemed."
        );

        redeemedWithdrawals[l1ERC20Address] = withdrawalHash;

         /*
         * Verify correct contract authenticated this withdrawal
         *
        ROLLUP_CONTRACT.verifyL2ToL1Message(
            withdrawalData,
            l2BridgeContracts[_L1ERC20Address],
		 _witness
        );
        */
        // send to the withdrawer
        ERC20(l1ERC20Address).transfer(withdrawTo, amount);

        // Returns true so that the L1 Message Receiver knows whether the call was successfully made
        return true;
    }
}