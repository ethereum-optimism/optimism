import React from 'react';
import { connect } from 'react-redux';
import { isEqual } from 'lodash';
import { logAmount, powAmount } from 'util/amountConvert';
import { BigNumber } from 'ethers';

import { openAlert, openError, openModal } from 'actions/uiAction';
import { getFarmInfo, updateStakeToken, updateWithdrawToken } from 'actions/farmAction';

import Button from 'components/button/Button';

import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import RemoveIcon from '@material-ui/icons/Remove';

import networkService from 'services/networkService';

import { getCoinImage } from 'util/coinImage';

import { Box, Typography, Fade, Grid } from '@material-ui/core';
import * as S from "./ListFarm.styles"

class ListFarm extends React.Component {

  constructor(props) {

    super(props)

    const {
      poolInfo,
      userInfo,
      L1orL2Pool,
      balance,
    } = this.props;

    this.state = {
      balance,
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

    const { poolInfo, userInfo, balance } = this.props;

    if (!isEqual(prevState.poolInfo, poolInfo)) {
      this.setState({ poolInfo });
    }

    if (!isEqual(prevState.userInfo, userInfo)) {
      this.setState({ userInfo });
    }

    if (!isEqual(prevState.balance, balance)) {
      this.setState({ balance });
    }

  }

  handleStakeToken() {

    const { poolInfo, L1orL2Pool, balance } = this.state

    this.props.dispatch(updateStakeToken({
      symbol: poolInfo.symbol,
      currency: L1orL2Pool === 'L1LP' ? poolInfo.l1TokenAddress : poolInfo.l2TokenAddress,
      LPAddress: L1orL2Pool === 'L1LP' ? networkService.L1LPAddress : networkService.L2LPAddress,
      L1orL2Pool,
      balance,
      decimals: poolInfo.decimals
    }))

    this.props.dispatch(openModal('farmDepositModal'))
  }

  handleWithdrawToken() {

    const { poolInfo, L1orL2Pool, balance } = this.state;

    this.props.dispatch(updateWithdrawToken({
      symbol: poolInfo.symbol,
      currency: L1orL2Pool === 'L1LP' ? poolInfo.l1TokenAddress : poolInfo.l2TokenAddress,
      LPAddress: L1orL2Pool === 'L1LP' ? networkService.L1LPAddress : networkService.L2LPAddress,
      L1orL2Pool,
      balance,
      decimals: poolInfo.decimals
    }))

    this.props.dispatch(openModal('farmWithdrawModal'))
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
      this.props.dispatch(openAlert(`${logAmount(userReward, poolInfo.decimals, 2)} ${poolInfo.symbol} was added to your account`))
      this.props.dispatch(getFarmInfo())
      this.setState({ loading: false })
    } else {
      this.props.dispatch(openError("Failed to get reward"))
      this.setState({ loading: false })
    }

  }

  render() {

    const {
      poolInfo, userInfo,
      dropDownBox,
      loading, L1orL2Pool
    } = this.state;

    const { isMobile } = this.props

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
    const decimals = poolInfo.decimals
    const logo = getCoinImage(symbol)

    //console.log('poolinfo',poolInfo)

    return (
      <S.Wrapper dropDownBox={dropDownBox}>
        <Grid container spacing={2} direction="row" justifyContent="center" alignItems="center" >

          <S.GridItemTag item xs={4} md={1.7}>
              <img src={logo} alt="logo" width={30} />
              <Typography variant="overline">{name}</Typography>
          </S.GridItemTag>

          <S.GridItemTag item xs={4} md={1.7}>
            {isMobile ? (
              <Typography variant="overline" sx={{opacity: 0.7}}>Earned</Typography>
            ) : (null)}
            <Typography variant="body1">
              {userReward ?
                `${logAmount(userReward, decimals, 2)}` : `0`
              }
            </Typography>
          </S.GridItemTag>

          <S.GridItemTag item xs={4} md={1.7}>
            {isMobile ? (
              <Typography variant="overline" sx={{opacity: 0.7}}>Your Stake</Typography>
            ) : (null)}
            <Typography variant="body1">
              {userInfo.amount ?
                `${logAmount(userInfo.amount, decimals, 2)}` : `0`
              }
            </Typography>
          </S.GridItemTag>

          <S.GridItemTag item xs={4} md={1.7}>
            {isMobile ? (
              <Typography variant="overline" sx={{opacity: 0.7}}>APR</Typography>
            ) : (null)}
            <Typography variant="body1">
              {userInfo.amount ?
                `${logAmount(poolInfo.APR, 0, 2)}` : `0`
              }
            </Typography>
          </S.GridItemTag>

          <S.GridItemTag item xs={4} md={1.7}>
            {isMobile ? (
              <Typography variant="overline" sx={{opacity: 0.7}}>Liquidity</Typography>
            ) : (null)}
            <Typography variant="body1">
              {poolInfo.userDepositAmount ?
                `${logAmount(poolInfo.userDepositAmount, decimals, 2)}` : `0`
              }
            </Typography>
          </S.GridItemTag>

          <S.GridItemTag item xs={4} md={1.7}>
            {isMobile ? (
              <Typography variant="overline" sx={{opacity: 0.7}}>Balance</Typography>
            ) : (null)}
            <Typography variant="body1">
              {poolInfo.tokenBalance ?
                `${logAmount(poolInfo.tokenBalance, decimals, 2)}` : `0`
              }
            </Typography>
          </S.GridItemTag>

          <S.GridItemTag item xs={12} md={1.7}>
            <Box
              disabled={disabled}
              onClick={()=>{this.setState({ dropDownBox: !dropDownBox, dropDownBoxInit: false })}}
              sx={{display: 'flex', cursor: 'pointer', color: "#0ebf9a", transform: dropDownBox ? "rotate(-180deg)" : ""}}
            >
              <ExpandMoreIcon />
            </Box>
          </S.GridItemTag>
        </Grid>

        {/*********************************************/
        /**************  Drop Down Box ****************/
        /**********************************************/
        }
        {dropDownBox ? (
          <Fade in={dropDownBox}>
            <S.DropdownContent>
              <S.DropdownWrapper>
                <Typography sx={{flex: 1}} variant="body2" component="div">{`${name}`} Earned</Typography>
                <Typography sx={{flex: 1}} variant="body2" component="div" color="secondary">{logAmount(userReward, decimals, 2)}</Typography>
                <Button
                  variant="contained"
                  fullWidth
                  disabled={logAmount(userReward, decimals) === '0' || disabled}
                  onClick={()=>{this.handleHarvest()}}
                  loading={loading}
                  sx={{flex: 1}}
                >
                  Harvest
                </Button>
              </S.DropdownWrapper>

              <S.DropdownWrapper>
                {logAmount(userInfo.amount, decimals) === '0' ?
                  <>
                    <Typography sx={{flex: 1}} variant="body2" component="div">Your Stake {`${name}`}</Typography>
                    <Button
                      variant="contained"
                      onClick={() => {this.handleStakeToken()}}
                      disabled={disabled}
                      fullWidth
                      sx={{flex: 1}}
                    >
                      Stake
                    </Button>
                  </> :
                  <>
                    <Typography variant="body2" component="div">{`${name}`} Staked</Typography>
                    <Typography variant="body2" component="div" color="secondary">{logAmount(userInfo.amount, decimals)}</Typography>
                    <Box sx={{display: "flex", alignItems: "center", gap: "5px"}}>
                      <Button
                        variant="outlined"
                        color="neutral"
                        onClick={() => {!disabled && this.handleWithdrawToken()}}
                      >
                        <RemoveIcon/>
                      </Button>
                      <Button variant="contained" onClick={() => {!disabled && this.handleStakeToken()}}>
                        Stake More
                      </Button>
                    </Box>
                  </>
                }
              </S.DropdownWrapper>
            </S.DropdownContent>
          </Fade>
        ) : null }

      </S.Wrapper>
    )
  }
}

const mapStateToProps = state => ({
})

export default connect(mapStateToProps)(ListFarm)
