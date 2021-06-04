// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1StandardBridge } from "../../../iOVM/bridge/tokens/iOVM_L1StandardBridge.sol";
import { iOVM_L1ERC20Bridge } from "../../../iOVM/bridge/tokens/iOVM_L1ERC20Bridge.sol";
import { iOVM_L2ERC20Bridge } from "../../../iOVM/bridge/tokens/iOVM_L2ERC20Bridge.sol";
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/* Library Imports */
import { Lib_AddressResolver } from "../../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_AddressManager } from "../../../libraries/resolver/Lib_AddressManager.sol";
import { OVM_CrossDomainEnabled } from "../../../libraries/bridge/OVM_CrossDomainEnabled.sol";
import { Lib_PredeployAddresses } from "../../../libraries/constants/Lib_PredeployAddresses.sol";
import { SafeMath } from "@openzeppelin/contracts/math/SafeMath.sol";
import { SafeERC20 } from "@openzeppelin/contracts/token/ERC20/SafeERC20.sol";

/**
 * @title OVM_L1StandardBridge
 * @dev The L1 ETH and ERC20 Bridge is a contract which stores deposited L1 funds and standard tokens that are in use on L2.
 * It synchronizes a corresponding L2 Bridge, informing it of deposits, and listening to it for newly finalized withdrawals.
 *
 * Compiler used: solc
 * Runtime target: EVM
 */
contract OVM_L1StandardBridge is iOVM_L1StandardBridge, OVM_CrossDomainEnabled, Lib_AddressResolver {
    using SafeMath for uint;
    using SafeERC20 for IERC20;

    /*************
     * Constants *
     ************/

    address internal constant L1_ETH_ADDRESS = 0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE;

    /********************************
     * External Contract References *
     ********************************/

    address public l2TokenBridge;

    address public ovmEth;

    // Maps L1 token to L2 token to balance of the L1 token deposited
    mapping(address => mapping (address => uint256)) public deposits;

    /***************
     * Constructor *
     ***************/

    // This contract lives behind a proxy, so the constructor parameters will go unused.
    constructor()
        OVM_CrossDomainEnabled(address(0))
        Lib_AddressResolver(address(0))
    {}

    /******************
     * Initialization *
     ******************/

    /**
     * @param _libAddressManager Address manager for this OE deployment.
     * @param _l1messenger L1 Messenger address being used for cross-chain communications.
     * @param _ovmEth L2 OVM_ETH implementation of iOVM_DepositedToken.
     */
    function initialize(
        address _libAddressManager,
        address _l1messenger,
        address _l2TokenBridge,
        address _ovmEth
    )
        public
    {
        require(libAddressManager == Lib_AddressManager(0), "Contract has already been initialized.");
        libAddressManager = Lib_AddressManager(_libAddressManager);
        messenger = _l1messenger;
        l2TokenBridge = _l2TokenBridge;
        ovmEth = _ovmEth;
        messenger = resolve("Proxy__OVM_L1CrossDomainMessenger");
    }

    /**************
     * Depositing *
     **************/
    /**
     * @dev This function can be called with no data
     * to deposit an amount of ETH to the caller's balance on L2.
     * Since the receive function doesn't take data, a conservative
     * default of 1.2 Million gas is forwarded to L2.
     */
    receive()
        external
        payable
    {
        _initiateETHDeposit(msg.sender, msg.sender, 1_200_000, bytes(""));
    }

    /**
     * @inheritdoc iOVM_L1StandardBridge
     */
    function depositETH(
        uint32 _l2Gas,
        bytes calldata _data
    )
        external
        override
        payable
    {
        _initiateETHDeposit(
            msg.sender,
            msg.sender,
            _l2Gas,
            _data
        );
    }

    /**
     * @inheritdoc iOVM_L1StandardBridge
     */
    function depositETHTo(
        address _to,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external
        override
        payable
    {
        _initiateETHDeposit(
            msg.sender,
            _to,
            _l2Gas,
            _data
        );
    }

    /**
     * @dev Performs the logic for deposits by storing the ETH and informing the L2 ETH Gateway of the deposit.
     * @param _from Account to pull the deposit from on L1.
     * @param _to Account to give the deposit to on L2.
     * @param _l2Gas Gas limit required to complete the deposit on L2.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function _initiateETHDeposit(
        address _from,
        address _to,
        uint32 _l2Gas,
        bytes memory _data
    )
        internal
    {
        // Construct calldata for finalizeDeposit call
        bytes memory message =
            abi.encodeWithSelector(
                iOVM_L2ERC20Bridge.finalizeDeposit.selector,
                L1_ETH_ADDRESS,
                Lib_PredeployAddresses.OVM_ETH,
                _from,
                _to,
                msg.value,
                _data
            );

        // Send calldata into L2
        sendCrossDomainMessage(
            l2TokenBridge,
            _l2Gas,
            message
        );

        emit ETHDepositInitiated(_from, _to, msg.value, _data);
    }

    /**
     * @inheritdoc iOVM_L1ERC20Bridge
     */
    function depositERC20(
        address _l1Token,
        address _l2Token,
        uint256 _amount,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external
        override
        virtual
    {
        _initiateERC20Deposit(_l1Token, _l2Token, msg.sender, msg.sender, _amount, _l2Gas, _data);
    }

     /**
     * @inheritdoc iOVM_L1ERC20Bridge
     */
    function depositERC20To(
        address _l1Token,
        address _l2Token,
        address _to,
        uint256 _amount,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external
        override
        virtual
    {
        _initiateERC20Deposit(_l1Token, _l2Token, msg.sender, _to, _amount, _l2Gas, _data);
    }

    /**
     * @dev Performs the logic for deposits by informing the L2 Deposited Token
     * contract of the deposit and calling a handler to lock the L1 funds. (e.g. transferFrom)
     *
     * @param _l1Token Address of the L1 ERC20 we are depositing
     * @param _l2Token Address of the L1 respective L2 ERC20
     * @param _from Account to pull the deposit from on L1
     * @param _to Account to give the deposit to on L2
     * @param _amount Amount of the ERC20 to deposit.
     * @param _l2Gas Gas limit required to complete the deposit on L2.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function _initiateERC20Deposit(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        uint32 _l2Gas,
        bytes calldata _data
    )
        internal
    {
        // When a deposit is initiated on L1, the L1 Bridge transfers the funds to itself for future withdrawals.
        IERC20(_l1Token).safeTransferFrom(
            _from,
            address(this),
            _amount
        );

        // Construct calldata for _l2Token.finalizeDeposit(_to, _amount)
        bytes memory message = abi.encodeWithSelector(
            iOVM_L2ERC20Bridge.finalizeDeposit.selector,
            _l1Token,
            _l2Token,
            _from,
            _to,
            _amount,
            _data
        );

        // Send calldata into L2
        sendCrossDomainMessage(
            l2TokenBridge,
            _l2Gas,
            message
        );

        deposits[_l1Token][_l2Token] = deposits[_l1Token][_l2Token].add(_amount);

        // We omit _data here because events only support bytes32 types.
        emit ERC20DepositInitiated(_l1Token, _l2Token, _from, _to, _amount, _data);
    }

    /*************************
     * Cross-chain Functions *
     *************************/

     /**
     * @inheritdoc iOVM_L1StandardBridge
     */
    function finalizeETHWithdrawal(
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    )
        external
        override
        onlyFromCrossDomainAccount(l2TokenBridge)
    {
        (bool success, ) = _to.call{value: _amount}(new bytes(0));
        require(success, 'TransferHelper::safeTransferETH: ETH transfer failed');

        emit ETHWithdrawalFinalized(_from, _to, _amount, _data);
    }

    /**
     * @inheritdoc iOVM_L1ERC20Bridge
     */
    function finalizeERC20Withdrawal(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    )
        external
        override
        onlyFromCrossDomainAccount(l2TokenBridge)
    {
        deposits[_l1Token][_l2Token] = deposits[_l1Token][_l2Token].sub(_amount);

        // When a withdrawal is finalized on L1, the L1 Bridge transfers the funds to the withdrawer.
        IERC20(_l1Token).safeTransfer(_to, _amount);

        emit ERC20WithdrawalFinalized(_l1Token, _l2Token, _from, _to, _amount, _data);
    }

    /*****************************
     * Temporary - Migrating ETH *
     *****************************/

    /**
     * @dev Migrates entire ETH balance to another gateway.
     * @param _to Gateway Proxy address to migrate ETH to.
     */
    function migrateEth(address payable _to) external {
        address owner = Lib_AddressManager(libAddressManager).owner();
        require(msg.sender == owner, "Only the owner can migrate ETH");
        uint256 balance = address(this).balance;
        OVM_L1StandardBridge(_to).donateETH{value:balance}();
    }

    /**
     * @dev Adds ETH balance to the account. This is meant to allow for ETH
     * to be migrated from an old gateway to a new gateway.
     */
    function donateETH() external payable {}
}
