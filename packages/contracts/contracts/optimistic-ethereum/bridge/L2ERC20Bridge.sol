pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;
import { ERC20 } from "./ERC20.sol";
import { DataTypes } from "../utils/DataTypes.sol";
import { L2ToL1MessagePasser } from "../ovm/precompiles/L2ToL1MessagePasser.sol";
import { DepositedERC20 } from "./DepositedERC20.sol";


contract L2ERC20Bridge {
    address l1ERC20Bridge;
    uint public withdrawalNonce = 0;
    mapping (address => address) public correspondingL1ERC20;
    mapping (address => address) public correspondingL2ERC20;

    constructor (address _l1ERC20Bridge) public {
        l1ERC20Bridge = _l1ERC20Bridge;
    }

    function deployNewDepositedERC20(
        address _withdrawTo,
        uint _amount
    ) public {

    }

    function forwardWithdrawalToL1(
        address _withdrawTo,
        uint _amount
    ) public {
        // DataType.Withdrawal withdrawal = withdrawal {
        //     amount: _amount
        //     withdrawTo: _withdrawTO
        //     withdra
        // }
        //abi.encode(_amount);
    }
}