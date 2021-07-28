/*
  Varna - A Privacy-Preserving Marketplace
  Varna uses Fully Homomorphic Encryption to make markets fair. 
  Copyright (C) 2021 Enya Inc. Palo Alto, CA

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

import networkService from 'services/networkService';

const getFarmInfoBegin = () => ({
  type: 'GET_FARMINFO',
})

const getFarmInfoSuccess = (L1PoolInfo, L1UserInfo, L2PoolInfo, L2UserInfo) => ({
  type: 'GET_FARMINFO_SUCCESS',
  payload: { L1PoolInfo, L1UserInfo, L2PoolInfo, L2UserInfo }
})

const getFeeBegin = () => ({
  type: 'GET_USERINFO',
})

const getFeeSuccess = (totalFeeRate, userRewardFeeRate) => ({
  type: 'GET_FEE_SUCCESS',
  payload: { totalFeeRate, userRewardFeeRate },
})

export const getFarmInfo = () => async (dispatch) => {
  dispatch(getFarmInfoBegin());
  const [L1LPInfo, L2LPInfo] = await Promise.all([
    networkService.getL1LPInfo(),
    networkService.getL2LPInfo(),
  ])
  dispatch(getFarmInfoSuccess(
    L1LPInfo.poolInfo,
    L1LPInfo.userInfo,
    L2LPInfo.poolInfo,
    L2LPInfo.userInfo,
  ))
}


export const getFee = () => async (dispatch) => {
  dispatch(getFeeBegin());
  const [totalFeeRate, userRewardFeeRate] = await Promise.all([
    networkService.getTotalFeeRate(),
    networkService.getUserRewardFeeRate(),
  ])
  dispatch(getFeeSuccess(totalFeeRate, userRewardFeeRate));
}

export const updateStakeToken = (stakeToken) => ({
  type: 'UPDATE_STAKE_TOKEN',
  payload: stakeToken,
})

export const updateWithdrawToken = (withdrawToken) => ({
  type: 'UPDATE_WITHDRAW_TOKEN',
  payload: withdrawToken,
})