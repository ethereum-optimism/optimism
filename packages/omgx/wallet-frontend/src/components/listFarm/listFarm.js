import React from 'react';
import { connect } from 'react-redux';
import { isEqual } from 'lodash';
import { logAmount, powAmount } from 'util/amountConvert';
import { BigNumber } from 'ethers';

import { openAlert, openError, openModal } from 'actions/uiAction';
import { getFarmInfo, updateStakeToken, updateWithdrawToken } from 'actions/farmAction';

import Button from 'components/button/Button';

import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import AddIcon from '@material-ui/icons/Add';
import RemoveIcon from '@material-ui/icons/Remove';

import networkService from 'services/networkService';

import * as styles from './listFarm.module.scss';

class ListFarm extends React.Component {
  
  constructor(props) {
    
    super(props);
    
    const { 
      logo,
      poolInfo, 
      userInfo, 
      L1orL2Pool,
      balance, 
      decimals 
    } = this.props;

    this.state = {
      logo,
      balance,
      decimals,
      L1orL2Pool,
      // data
      poolInfo, 
      userInfo,
      //drop down box
      dropDownBox: false,
      dropDownBoxInit: true,
      // loading
      loading: false,
    }
  }
  
  componentDidUpdate(prevState) {

    const { poolInfo, userInfo, balance, decimals } = this.props;

    if (!isEqual(prevState.poolInfo, poolInfo)) {
      this.setState({ poolInfo });
    }

    if (!isEqual(prevState.userInfo, userInfo)) {
      this.setState({ userInfo });
    }

    if (!isEqual(prevState.balance, balance)) {
      this.setState({ balance });
    }

    if (!isEqual(prevState.decimals, decimals)) {
      this.setState({ decimals });
    }

  }

  handleStakeToken() {
    
    const { poolInfo, L1orL2Pool, balance, decimals } = this.state
    
    this.props.dispatch(updateStakeToken({
      symbol: poolInfo.symbol,
      currency: L1orL2Pool === 'L1LP' ? poolInfo.l1TokenAddress : poolInfo.l2TokenAddress,
      LPAddress: L1orL2Pool === 'L1LP' ? networkService.L1LPAddress : networkService.L2LPAddress,
      L1orL2Pool,
      balance,
      decimals
    }))
    
    this.props.dispatch(openModal('farmDepositModal'))
  }

  handleWithdrawToken() {
    
    const { poolInfo, L1orL2Pool, balance, decimals } = this.state;
    
    this.props.dispatch(updateWithdrawToken({
      symbol: poolInfo.symbol,
      currency: L1orL2Pool === 'L1LP' ? poolInfo.l1TokenAddress : poolInfo.l2TokenAddress,
      LPAddress: L1orL2Pool === 'L1LP' ? networkService.L1LPAddress : networkService.L2LPAddress,
      L1orL2Pool,
      balance,
      decimals
    }))
    
    this.props.dispatch(openModal('farmWithdrawModal'));
  }

  async handleHarvest() {
    
    const { poolInfo, userInfo } = this.state;

    this.setState({ loading: true })

    const userReward = BigNumber.from(userInfo.pendingReward).add(
      BigNumber.from(userInfo.amount)
      .mul(BigNumber.from(poolInfo.accUserRewardPerShare))
      .div(BigNumber.from(powAmount(1, 12)))
      .sub(BigNumber.from(userInfo.rewardDebt))
    ).toString()

    let getRewardTX = null;

    if(networkService.L1orL2 === 'L1') {
      getRewardTX = await networkService.getRewardL1(
        poolInfo.l1TokenAddress,
        userReward
      )
    } else if (networkService.L1orL2 === 'L2') {
      getRewardTX = await networkService.getRewardL2(
        poolInfo.l2TokenAddress,
        userReward
      )
    } else {
      console.log("handleHarvest(): Chain not set")
    }

    if (getRewardTX) {
      this.props.dispatch(openAlert(`${logAmount(userReward, 18, 2)} ${poolInfo.symbol} was added to your account`));
      this.props.dispatch(getFarmInfo());
      this.setState({ loading: false });
    } else {
      this.props.dispatch(openError("Failed to get reward"));
      this.setState({ loading: false });
    }

  }

  render() {

    const { 
      logo,
      poolInfo, userInfo,
      dropDownBox, dropDownBoxInit,
      loading, L1orL2Pool
    } = this.state;

    let userReward = 0;

    if (Object.keys(userInfo).length && Object.keys(poolInfo).length) {
      userReward = BigNumber.from(userInfo.pendingReward).add(
        BigNumber.from(userInfo.amount)
        .mul(BigNumber.from(poolInfo.accUserRewardPerShare))
        .div(BigNumber.from(powAmount(1, 12)))
        .sub(BigNumber.from(userInfo.rewardDebt))
      ).toString()
    }

    // L1orL2Pool: L1LP || L2LP
    // networkService.L1OrL2 L1: || L2
    const disabled = !L1orL2Pool.includes(networkService.L1orL2)
    const symbol = poolInfo.symbol
    const name = poolInfo.name

    return (
      <div className={styles.ListFarm}>
        <div 
          className={styles.topContainer} 
          style={disabled ? {pointerEvents: 'none'} : {}}
          onClick={()=>{this.setState({ dropDownBox: !dropDownBox, dropDownBoxInit: false })}}
        >
          <div className={styles.Table1}>
            <img className={styles.Image} src={logo} alt="logo"/>
            <div className={styles.BasicText}>{name}</div>
          </div>
          <div className={styles.Table2}>
            <div className={styles.BasicText}>Earned</div>
            <div className={styles.BasicLightText}>
              {userReward ? 
                `${logAmount(userReward, 18, 2)} ${symbol}` : `0 ${symbol}`
              }
            </div>
          </div>
          <div className={styles.Table3}>
            <div className={styles.BasicText}>Share</div>
            <div className={styles.BasicLightText}>
              {userInfo.amount ? 
                `${logAmount(userInfo.amount, 18, 2)} ${symbol}` : `0 ${symbol}`
              }
            </div>
          </div>
          <div className={styles.Table4}>
            <div className={styles.BasicText}>APR</div>
            <div className={styles.BasicLightText}>
              {`${poolInfo.APR ? poolInfo.APR.toFixed(2) : 0}%`}
            </div>
          </div>
          <div className={styles.Table5}>
            <div className={styles.BasicText}>Liquidity</div>
            <div className={styles.BasicLightText}>
              {poolInfo.userDepositAmount ? 
                `${logAmount(poolInfo.userDepositAmount, 18, 2)} ${symbol}` : `0 ${symbol}`
              }
            </div>
          </div>
          {disabled &&
            <div className={styles.Table6}>
              <div className={styles.LinkTextOff}>Staking</div>
              <ExpandMoreIcon className={styles.LinkButtonOff} />
            </div>
          }
          {!disabled &&
            <div className={styles.Table6}>
              <div className={styles.LinkText}>Staking</div>
              <ExpandMoreIcon className={styles.LinkButton} />
            </div>
          }
        </div>

        {/*********************************************/
        /**************  Drop Down Box ****************/
        /**********************************************/
        }
        <div 
          className={dropDownBox ? 
            styles.dropDownContainer: dropDownBoxInit ? styles.dropDownInit : styles.closeDropDown}
        >
          <div className={styles.boxContainer}>
            <div className={styles.BasicText}>{`${name}`} Earned</div>
            <div className={styles.boxRowContainer}>
              <div className={styles.LargeBlueText}>{logAmount(userReward, 18, 2)}</div>
              <Button
                type='primary'
                size='small'
                className={styles.smallButton}
                disabled={logAmount(userReward, 18) === '0' || disabled}
                onClick={()=>{this.handleHarvest()}}
                loading={loading}
              >
                Harvest
              </Button>
            </div>
          </div>
          
          <div className={styles.boxContainer}>
            {logAmount(userInfo.amount, 18) === '0' ? 
              <>
                <div className={styles.BasicText}>Stake {`${name}`}</div>
                <div className={styles.boxRowContainer}>
                  <Button
                    type='primary'
                    className={styles.largeButton}
                    onClick={() => {this.handleStakeToken()}}
                    disabled={disabled}
                  >
                    Stake
                  </Button>
                </div>
              </>:
              <>
                <div className={styles.BasicText}>{`${name}`} Staked</div>
                <div className={styles.boxRowContainer}>
                  <div className={styles.LargeBlueText}>{logAmount(userInfo.amount, 18)}</div>
                  <div className={styles.AdjustButtonsContainer}>
                    <div 
                      className={disabled ? styles.AdjustButtonContainerDisabled : styles.AdjustButtonContainer}
                      onClick={() => {!disabled && this.handleStakeToken()}}
                    >
                      <AddIcon className={styles.AdjustButton} />
                    </div>
                    <div 
                      className={disabled ? styles.AdjustButtonContainerDisabled : styles.AdjustButtonContainer}
                      onClick={() => {!disabled && this.handleWithdrawToken()}}
                    >
                      <RemoveIcon className={styles.AdjustButton} />
                    </div>
                  </div>
                </div>
              </>
            }
          </div>

        </div>

      </div>
    )
  }
}

const mapStateToProps = state => ({ 
  login: state.login,
  sell: state.sell,
  sellTask: state.sellTask,
  buy: state.buy,
});

export default connect(mapStateToProps)(ListFarm);