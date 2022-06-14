// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import {
    Lib_PredeployAddresses
} from "@eth-optimism/contracts/libraries/constants/Lib_PredeployAddresses.sol";
import { StandardBridge } from "../universal/StandardBridge.sol";

/**
 * @title L1StandardBridge
 * @dev The L1 ETH and ERC20 Bridge is a contract which stores deposited L1 funds and standard
 * tokens that are in use on L2. It synchronizes a corresponding L2 Bridge, informing it of deposits
 * and listening to it for newly finalized withdrawals.
 * Note that this contract is not intended to support all variations of ERC20 tokens; this
 * includes, but is not limited to tokens with transfer fees, rebasing tokens, and
 * tokens with blocklists.
 */
contract L1StandardBridge is StandardBridge {
    /**********
     * Events *
     **********/

    event ETHDepositInitiated(
        address indexed _from,
        address indexed _to,
        uint256 _amount,
        bytes _extraData
    );

    event ETHWithdrawalFinalized(
        address indexed _from,
        address indexed _to,
        uint256 _amount,
        bytes _extraData
    );

    event ERC20DepositInitiated(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _extraData
    );

    event ERC20WithdrawalFinalized(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _extraData
    );

    /********************
     * Public Functions *
     ********************/

    /**
     * @dev initialize the L1StandardBridge with the address of the
     *      messenger in the same domain
     */
    function initialize(address payable _messenger) public {
        _initialize(_messenger, payable(Lib_PredeployAddresses.L2_STANDARD_BRIDGE));
    }

    /**
     * @dev Get the address of the corresponding L2 bridge contract.
     *      This is a legacy getter, provided for backwards compatibility.
     * @return Address of the corresponding L2 bridge contract.
     */
    function l2TokenBridge() external returns (address) {
        return address(otherBridge);
    }

    /**
     * @dev Deposit an amount of the ETH to the caller's balance on L2.
     * @param _minGasLimit limit required to complete the deposit on L2.
     * @param _extraData Optional data to forward to L2. This data is provided solely as a
     *        convenience for external contracts which may validate that the data is included in the
     *        CrossDomainMessenger's sentMessages mapping.
     */
    function depositETH(uint32 _minGasLimit, bytes calldata _extraData) external payable onlyEOA {
        _initiateETHDeposit(msg.sender, msg.sender, _minGasLimit, _extraData);
    }

    /**
     * @dev Deposit an amount of ETH to a recipient's balance on L2. Note that if ETH is sent to a
     *      contract on L2 and the call fails, then that ETH will be locked in the L2StandardBridge.
     * @param _to L2 address to credit the deposit to.
     * @param _minGasLimit Gas limit required to complete the deposit on L2.
     * @param _extraData Optional data to forward to L2. This data is provided solely as a
     *        convenience for external contracts which may validate that the data is included in the
     *        CrossDomainMessenger's sentMessages mapping.
     */
    function depositETHTo(
        address _to,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) external payable {
        _initiateETHDeposit(msg.sender, _to, _minGasLimit, _extraData);
    }

    /**
     * @dev deposit an amount of the ERC20 to the caller's balance on L2.
     * @param _l1Token Address of the L1 ERC20 we are depositing.
     * @param _l2Token Address of the L2 token we are depositing to.
     * @param _amount Amount of the ERC20 to deposit.
     * @param _minGasLimit limit required to complete the deposit on L2.
     * @param _extraData Optional data to forward to L2.  This data is not forwarded to the
     *        token contract and is provided solely as a convenience for external contracts
     *        which may validate that the data is included in the CrossDomainMessenger's
     *        sentMessages mapping.
     */
    function depositERC20(
        address _l1Token,
        address _l2Token,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) external virtual onlyEOA {
        _initiateERC20Deposit(
            _l1Token,
            _l2Token,
            msg.sender,
            msg.sender,
            _amount,
            _minGasLimit,
            _extraData
        );
    }

    /**
     * @dev deposit an amount of ERC20 to a recipient's balance on L2.
     * @param _l1Token Address of the L1 ERC20 we are depositing.
     * @param _l2Token Address of the L2 token we are depositing to.
     * @param _to L2 address to credit the deposit to.
     * @param _amount Amount of the ERC20 to deposit.
     * @param _minGasLimit Gas limit required to complete the deposit on L2.
     * @param _extraData Optional data to forward to L2.  This data is not forwarded to the
     *        token contract and is provided solely as a convenience for external contracts
     *        which may validate that the data is included in the CrossDomainMessenger's
     *        sentMessages mapping.
     */
    function depositERC20To(
        address _l1Token,
        address _l2Token,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) external virtual {
        _initiateERC20Deposit(
            _l1Token,
            _l2Token,
            msg.sender,
            _to,
            _amount,
            _minGasLimit,
            _extraData
        );
    }

    function finalizeETHWithdrawal(
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    ) external payable onlyOtherBridge {
        emit ETHWithdrawalFinalized(_from, _to, _amount, _data);
        finalizeBridgeETH(_from, _to, _amount, _data);
    }

    /**
     * @dev Complete a withdrawal from L2 to L1, and credit funds to the recipient's balance of the
     * L1 ERC20 token.
     * This call will fail if the initialized withdrawal from L2 has not been finalized.
     *
     * @param _l1Token Address of L1 token to finalizeWithdrawal for.
     * @param _l2Token Address of L2 token where withdrawal was initiated.
     * @param _from L2 address initiating the transfer.
     * @param _to L1 address to credit the withdrawal to.
     * @param _amount Amount of the ERC20 to deposit.
     * @param _extraData Data provided by the sender on L2. This data is not forwarded to the
     *   token contract. It is provided solely as a convenience for external contracts which
     *   may validate that the data is included in the CrossDomainMessenger's sentMessages
     *   mapping.
     */
    function finalizeERC20Withdrawal(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    ) external onlyOtherBridge {
        emit ERC20WithdrawalFinalized(_l1Token, _l2Token, _from, _to, _amount, _extraData);
        finalizeBridgeERC20(_l1Token, _l2Token, _from, _to, _amount, _extraData);
    }

    /**********************
     * Internal Functions *
     **********************/

    function _initiateETHDeposit(
        address _from,
        address _to,
        uint32 _minGasLimit,
        bytes memory _extraData
    ) internal {
        emit ETHDepositInitiated(_from, _to, msg.value, _extraData);
        _initiateBridgeETH(_from, _to, msg.value, _minGasLimit, _extraData);
    }

    function _initiateERC20Deposit(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) internal {
        emit ERC20DepositInitiated(_l1Token, _l2Token, _from, _to, _amount, _extraData);
        _initiateBridgeERC20(_l1Token, _l2Token, _from, _to, _amount, _minGasLimit, _extraData);
    }
}
