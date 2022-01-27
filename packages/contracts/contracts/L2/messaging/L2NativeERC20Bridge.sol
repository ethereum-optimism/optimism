// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Interface Imports */
import { IL2NativeERC20Bridge } from "./IL2NativeERC20Bridge.sol";
import { IL1NativeERC20Bridge } from "../../L1/messaging/IL1NativeERC20Bridge.sol";
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/* Library Imports */
import { CrossDomainEnabled } from "../../libraries/bridge/CrossDomainEnabled.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";
import { SafeERC20 } from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

/**
 * @title L2NativeERC20Bridge
 * @dev
 *
 */
contract L2NativeERC20Bridge is IL2NativeERC20Bridge, CrossDomainEnabled {
    using SafeERC20 for IERC20;

    /********************************
     * External Contract References *
     ********************************/

    address public l1TokenBridge;

    // Maps L2 token to L1 token to balance of the L2 token deposited
    mapping(address => mapping(address => uint256)) public deposits;

    /***************
     * Constructor *
     ***************/

    // This contract lives behind a proxy, so the constructor parameters will go unused.
    constructor() CrossDomainEnabled(address(0)) {}

    /******************
     * Initialization *
     ******************/

    /**
     * @param _l2messenger L2 Messenger address being used for cross-chain communications.
     * @param _l1TokenBridge L1 standard bridge address.
     */
    // slither-disable-next-line external-function
    function initialize(address _l2messenger, address _l1TokenBridge) public {
        require(messenger == address(0), "Contract has already been initialized.");
        messenger = _l2messenger;
        l1TokenBridge = _l1TokenBridge;
    }

    /**************
     * Depositing *
     **************/

    /** @dev Modifier requiring sender to be EOA.  This check could be bypassed by a malicious
     *  contract via initcode, but it takes care of the user error we want to avoid.
     */
    modifier onlyEOA() {
        // Used to stop deposits from contracts (avoid accidentally lost tokens)
        require(!Address.isContract(msg.sender), "Account not EOA");
        _;
    }

    /**
     * @dev This function can be called with no data
     * to deposit an amount of ETH to the caller's balance on L2.
     * Since the receive function doesn't take data, a conservative
     * default amount is forwarded to L2.
     */
    // receive() external payable onlyEOA {
    //     _initiateETHDeposit(msg.sender, msg.sender, 200_000, bytes(""));
    // }

    /**
     * @inheritdoc IL2NativeERC20Bridge
     */
    function depositERC20(
        address _l2Token,
        address _l1Token,
        uint256 _amount,
        uint32 _l1Gas,
        bytes calldata _data
    ) external virtual onlyEOA {
        _initiateERC20Deposit(_l2Token, _l1Token, msg.sender, msg.sender, _amount, _l1Gas, _data);
    }

    /**
     * @inheritdoc IL2NativeERC20Bridge
     */
    function depositERC20To(
        address _l2Token,
        address _l1Token,
        address _to,
        uint256 _amount,
        uint32 _l1Gas,
        bytes calldata _data
    ) external virtual {
        _initiateERC20Deposit(_l2Token, _l1Token, msg.sender, _to, _amount, _l1Gas, _data);
    }

    /**
     * @dev Performs the logic for deposits by informing the L1 Deposited Token
     * contract of the deposit and calling a handler to lock the L2 funds. (e.g. transferFrom)
     * @param _l2Token Address of the L2 ERC20 we are depositing
     * @param _l1Token Address of the L2 respective L1 ERC20
     * @param _from Account to pull the deposit from on L2
     * @param _to Account to give the deposit to on L1
     * @param _amount Amount of the ERC20 to deposit.
     * @param _l1Gas Gas limit required to complete the deposit on L1.
     * @param _data Optional data to forward to L1. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function _initiateERC20Deposit(
        address _l2Token,
        address _l1Token,
        address _from,
        address _to,
        uint256 _amount,
        uint32 _l1Gas,
        bytes calldata _data
    ) internal {
        // When a deposit is initiated on L2, the L2 Bridge transfers the funds to itself for future
        // withdrawals. safeTransferFrom also checks if the contract has code, so this will fail if
        // _from is an EOA or address(0).
        // slither-disable-next-line reentrancy-events, reentrancy-benign
        IERC20(_l2Token).safeTransferFrom(_from, address(this), _amount);

        // Construct calldata for _l1Token.finalizeDeposit(_to, _amount)
        bytes memory message = abi.encodeWithSelector(
            IL1NativeERC20Bridge.finalizeDeposit.selector,
            _l2Token,
            _l1Token,
            _from,
            _to,
            _amount,
            _data
        );

        // Send calldata into L1
        // slither-disable-next-line reentrancy-events, reentrancy-benign
        sendCrossDomainMessage(l1TokenBridge, _l1Gas, message);

        // slither-disable-next-line reentrancy-benign
        deposits[_l2Token][_l1Token] = deposits[_l2Token][_l1Token] + _amount;

        // slither-disable-next-line reentrancy-events
        emit NativeERC20DepositInitiated(_l2Token, _l1Token, _from, _to, _amount, _data);
    }

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * @inheritdoc IL2NativeERC20Bridge
     */
    function finalizeERC20Withdrawal(
        address _l2Token,
        address _l1Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    ) external onlyFromCrossDomainAccount(l1TokenBridge) {
        deposits[_l2Token][_l1Token] = deposits[_l2Token][_l1Token] - _amount;

        // When a withdrawal is finalized on L2, the L2 Bridge transfers the funds to the withdrawer
        // slither-disable-next-line reentrancy-events
        IERC20(_l2Token).safeTransfer(_to, _amount);

        // slither-disable-next-line reentrancy-events
        emit NativeERC20DepositInitiated(_l2Token, _l1Token, _from, _to, _amount, _data);
    }
}
