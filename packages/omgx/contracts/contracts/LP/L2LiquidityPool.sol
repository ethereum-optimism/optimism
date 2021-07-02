// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;

import "./interfaces/iL1LiquidityPool.sol";

/* Library Imports */
import "@eth-optimism/contracts/contracts/optimistic-ethereum/libraries/bridge/OVM_CrossDomainEnabled.sol";

/* External Imports */
import '@openzeppelin/contracts/math/SafeMath.sol';
import '@openzeppelin/contracts/token/ERC20/SafeERC20.sol';
import '@openzeppelin/contracts/access/Ownable.sol';

/**
 * @dev An L2 LiquidityPool implementation
 */

contract L2LiquidityPool is OVM_CrossDomainEnabled, Ownable {
    using SafeERC20 for IERC20;
    using SafeMath for uint256;

    /**************
     *   Struct   *
     **************/
    // Info of each user.
    struct UserInfo {
        uint256 amount; // How many LP tokens the user has provided.
        uint256 rewardDebt; // Reward debt. See explanation below.
        uint256 pendingReward; // Pending reward
        //
        // We do some fancy math here. Basically, any point in time, the amount of rewards
        // entitled to a user but is pending to be distributed is:
        //
        //   Update Reward Per Share:
        //   accUserRewardPerShare = accUserRewardPerShare + (accUserReward - lastAccUserReward) / userDepositAmount
        //
        //  LP Provider:
        //      Deposit:
        //          Case 1 (new user):
        //              Update Reward Per Share();
        //              Calculate user.rewardDebt = amount * accUserRewardPerShare;
        //          Case 2 (user who has already deposited add more funds):
        //              Update Reward Per Share();
        //              Calculate user.pendingReward = amount * accUserRewardPerShare - user.rewardDebt;
        //              Calculate user.rewardDebt = (amount + new_amount) * accUserRewardPerShare;
        //
        //      Withdraw
        //          Update Reward Per Share();
        //          Calculate user.pendingReward = amount * accUserRewardPerShare - user.rewardDebt;
        //          Calculate user.rewardDebt = (amount - withdraw_amount) * accUserRewardPerShare;
    }
    // Info of each pool.
    struct PoolInfo {
        address l1TokenAddress; // Address of token contract.
        address l2TokenAddress; // Address of toekn contract.

        // balance
        uint256 userDepositAmount; // user deposit amount;

        // user rewards
        uint256 lastAccUserReward; // Last accumulated user reward
        uint256 accUserReward; // Accumulated user reward.
        uint256 accUserRewardPerShare; // Accumulated user rewards per share, times 1e12. See below.

        // owner rewards
        uint256 accOwnerReward; // Accumulated owner reward.

        // start time
        uint256 startTime;
    }

    /*************
     * Variables *
     *************/

    // mapping L1 and L2 token address to poolInfo
    mapping(address => PoolInfo) public poolInfo;
    // Info of each user that stakes tokens.
    mapping(address => mapping(address => UserInfo)) public userInfo;

    address L1LiquidityPoolAddress;
    uint256 public totalFeeRate;
    uint256 public userRewardFeeRate;
    uint256 public ownerRewardFeeRate;
    // Default gas value which can be overridden if more complex logic runs on L1.
    uint32 internal constant DEFAULT_FINALIZE_WITHDRAWAL_L1_GAS = 100000;

    /********************
     *       Event      *
     ********************/

    event AddLiquidity(
        address sender,
        uint256 amount,
        address tokenAddress
    );

    event OwnerRecoverFee(
        address sender,
        address receiver,
        uint256 amount,
        address tokenAddress
    );

    event ClientDepositL2(
        address sender,
        uint256 receivedAmount,
        address tokenAddress
    );

    event ClientPayL2(
        address sender,
        uint256 amount,
        uint256 userRewardFee,
        uint256 ownerRewardFee,
        uint256 totalFee,
        address tokenAddress
    );

    event WithdrawLiquidity(
        address sender,
        address receiver,
        uint256 amount,
        address tokenAddress
    );

    event WithdrawReward(
        address sender,
        address receiver,
        uint256 amount,
        address tokenAddress
    );

    /********************************
     * Constructor & Initialization *
     ********************************/

    /**
     * @param _l2CrossDomainMessenger L1 Messenger address being used for cross-chain communications.
     */
    constructor (
        address _l2CrossDomainMessenger
    )
        OVM_CrossDomainEnabled(_l2CrossDomainMessenger)
    {}

    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyInitialized() {
        require(address(L1LiquidityPoolAddress) != address(0), "Contract has not yet been initialized");
        _;
    }

    /********************
     * Public Functions *
     ********************/

    /**
     * @dev Initialize this contract.
     *
     * @param _userRewardFeeRate fee rate that users get
     * @param _ownerRewardFeeRate fee rate that contract owner gets
     * @param _L1LiquidityPoolAddress Address of the corresponding L1 LP deployed to the main chain
     */
    function init(
        uint256 _userRewardFeeRate,
        uint256 _ownerRewardFeeRate,
        address _L1LiquidityPoolAddress
    )
        public
        onlyOwner()
    {
        totalFeeRate = _userRewardFeeRate + _ownerRewardFeeRate;
        userRewardFeeRate = _userRewardFeeRate;
        ownerRewardFeeRate = _ownerRewardFeeRate;
        L1LiquidityPoolAddress = _L1LiquidityPoolAddress;
    }

    /***
     * @dev Add the new token pair to the pool
     * DO NOT add the same LP token more than once. Rewards will be messed up if you do.
     *
     * @param _l1TokenAddress
     * @param _l2TokenAddress
     *
     */
    function registerPool (
        address _l1TokenAddress,
        address _l2TokenAddress
    )
        public
        onlyOwner()
    {
        // use with caution, can register only once
        PoolInfo storage pool = poolInfo[_l2TokenAddress];
        // l2 token address equal to zero, then pair is not registered.
        require(pool.l2TokenAddress == address(0), "Token Address Already Registerd");
        poolInfo[_l2TokenAddress] =
            PoolInfo({
                l1TokenAddress: _l1TokenAddress,
                l2TokenAddress: _l2TokenAddress,
                userDepositAmount: 0,
                lastAccUserReward: 0,
                accUserReward: 0,
                accUserRewardPerShare: 0,
                accOwnerReward: 0,
                startTime: block.timestamp
            });
    }

    /**
     * @dev Overridable getter for the L1 gas limit of settling the deposit, in the case it may be
     * dynamic, and the above public constant does not suffice.
     *
     */
    function getFinalizeDepositL1Gas()
        public
        view
        virtual
        returns(
            uint32
        )
    {
        return DEFAULT_FINALIZE_WITHDRAWAL_L1_GAS;
    }

    /**
     * Update the user reward per share
     * @param _tokenAddress Address of the target token.
     */
    function updateUserRewardPerShare(
        address _tokenAddress
    )
        public
    {
        PoolInfo storage pool = poolInfo[_tokenAddress];
        if (pool.lastAccUserReward < pool.accUserReward) {
            uint256 accUserRewardDiff = (pool.accUserReward.sub(pool.lastAccUserReward));
            if (pool.userDepositAmount != 0) {
                pool.accUserRewardPerShare = pool.accUserRewardPerShare.add(
                    accUserRewardDiff.mul(1e12).div(pool.userDepositAmount)
                );
            }
            pool.lastAccUserReward = pool.accUserReward;
        }
    }

    /**
     * Liquididity providers add liquidity
     * @param _amount liquidity amount that users want to deposit.
     * @param _tokenAddress address of the liquidity token.
     */
     function addLiquidity(
        uint256 _amount,
        address _tokenAddress
    )
        external
    {
        PoolInfo storage pool = poolInfo[_tokenAddress];
        UserInfo storage user = userInfo[_tokenAddress][msg.sender];

        require(pool.l2TokenAddress != address(0), "Token Address Not Registered");

        // Update accUserRewardPerShare
        updateUserRewardPerShare(_tokenAddress);

        // if the user has already deposited token, we move the rewards to
        // pendingReward and update the reward debet.
        if (user.amount > 0) {
            user.pendingReward = user.pendingReward.add(
                user.amount.mul(pool.accUserRewardPerShare).div(1e12).sub(user.rewardDebt)
            );
            user.rewardDebt = (user.amount.add(_amount)).mul(pool.accUserRewardPerShare).div(1e12);
        } else {
            user.rewardDebt = _amount.mul(pool.accUserRewardPerShare).div(1e12);
        }

        // transfer funds
        IERC20(_tokenAddress).safeTransferFrom(msg.sender, address(this), _amount);

        // update amounts
        user.amount = user.amount.add(_amount);
        pool.userDepositAmount = pool.userDepositAmount.add(_amount);

        emit AddLiquidity(
            msg.sender,
            _amount,
            _tokenAddress
        );
    }

    /**
     * Client deposit ERC20 from their account to this contract, which then releases funds on the L1 side
     * @param _amount amount that client wants to transfer.
     * @param _tokenAddress L2 token address
     */
    function clientDepositL2(
        uint256 _amount,
        address _tokenAddress
    )
        external
    {
        PoolInfo storage pool = poolInfo[_tokenAddress];

        require(pool.l2TokenAddress != address(0), "Token Address Not Registered");

        IERC20(_tokenAddress).safeTransferFrom(msg.sender, address(this), _amount);

        // Construct calldata for L1LiquidityPool.depositToFinalize(_to, receivedAmount)
        bytes memory data = abi.encodeWithSelector(
            iL1LiquidityPool.clientPayL1.selector,
            msg.sender,
            _amount,
            pool.l1TokenAddress
        );

        // Send calldata into L1
        sendCrossDomainMessage(
            address(L1LiquidityPoolAddress),
            getFinalizeDepositL1Gas(),
            data
        );

        emit ClientDepositL2(
            msg.sender,
            _amount,
            _tokenAddress
        );

    }

    /**
     * Users withdraw token from LP
     * @param _amount amount to withdraw
     * @param _tokenAddress L2 token address
     * @param _to receiver to get the funds
     */
    function withdrawLiquidity(
        uint256 _amount,
        address _tokenAddress,
        address payable _to
    )
        external
    {
        PoolInfo storage pool = poolInfo[_tokenAddress];
        UserInfo storage user = userInfo[_tokenAddress][msg.sender];

        require(pool.l2TokenAddress != address(0), "Token Address Not Registered");
        require(user.amount >= _amount, "Withdraw Error");

        // Update accUserRewardPerShare
        updateUserRewardPerShare(_tokenAddress);

        // calculate all the rewards and set it as pending rewards
        user.pendingReward = user.pendingReward.add(
            user.amount.mul(pool.accUserRewardPerShare).div(1e12).sub(user.rewardDebt)
        );
        // Update the user data
        user.amount = user.amount.sub(_amount);
        // update reward debt
        user.rewardDebt = user.amount.mul(pool.accUserRewardPerShare).div(1e12);
        // update total user deposit amount
        pool.userDepositAmount = pool.userDepositAmount.sub(_amount);

        IERC20(_tokenAddress).safeTransfer(_to, _amount);

        emit WithdrawLiquidity(
            msg.sender,
            _to,
            _amount,
            _tokenAddress
        );
    }

    /**
     * owner recovers fee from ERC20
     * @param _amount amount to transfer to the other account.
     * @param _tokenAddress L2 token address
     * @param _to receiver to get the fee.
     */
    function ownerRecoverFee(
        uint256 _amount,
        address _tokenAddress,
        address _to
    )
        external
        onlyOwner()
    {
        PoolInfo storage pool = poolInfo[_tokenAddress];

        require(pool.l2TokenAddress != address(0), "Token Address Not Register");
        require(pool.accOwnerReward >= _amount, "Owner Reward Withdraw Error");

        IERC20(_tokenAddress).safeTransfer(_to, _amount);

        pool.accOwnerReward = pool.accOwnerReward.sub(_amount);

        emit OwnerRecoverFee(
            msg.sender,
            _to,
            _amount,
            _tokenAddress
        );
    }

    /**
     * withdraw reward from ERC20
     * @param _amount reward amount that liquidity providers want to withdraw
     * @param _tokenAddress L2 token address
     * @param _to receiver to get the reward
     */
    function withdrawReward(
        uint256 _amount,
        address _tokenAddress,
        address _to
    )
        external
        onlyOwner()
    {
        PoolInfo storage pool = poolInfo[_tokenAddress];
        UserInfo storage user = userInfo[_tokenAddress][msg.sender];

        require(pool.l2TokenAddress != address(0), "Token Address Not Registered");

        uint256 pendingReward = user.pendingReward.add(
            user.amount.mul(pool.accUserRewardPerShare).div(1e12).sub(user.rewardDebt)
        );

        require(pendingReward >= _amount, "Withdraw Reward Error");

        user.pendingReward = pendingReward.sub(_amount);
        user.rewardDebt = user.amount.mul(pool.accUserRewardPerShare).div(1e12);

        IERC20(_tokenAddress).safeTransfer(_to, _amount);

        emit WithdrawReward(
            msg.sender,
            _to,
            _amount,
            _tokenAddress
        );
    }

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * Move funds from L1 to L2, and pay out from the right liquidity pool
     * @param _to receiver to get the funds
     * @param _amount amount to to be transferred.
     * @param _tokenAddress L2 token address
     */
    function clientPayL2(
        address _to,
        uint256 _amount,
        address _tokenAddress
    )
        external
        onlyInitialized()
        onlyFromCrossDomainAccount(address(L1LiquidityPoolAddress))
    {
        PoolInfo storage pool = poolInfo[_tokenAddress];

        uint256 userRewardFee = (_amount.mul(userRewardFeeRate)).div(1000);
        uint256 ownerRewardFee = (_amount.mul(ownerRewardFeeRate)).div(1000);
        uint256 totalFee = userRewardFee.add(ownerRewardFee);
        uint256 receivedAmount = _amount.sub(totalFee);

        pool.accUserReward = pool.accUserReward.add(userRewardFee);
        pool.accOwnerReward = pool.accOwnerReward.add(ownerRewardFee);

        IERC20(_tokenAddress).safeTransfer(_to, receivedAmount);

        emit ClientPayL2(
          _to,
          receivedAmount,
          userRewardFee,
          ownerRewardFee,
          totalFee,
          _tokenAddress
        );
    }

}
