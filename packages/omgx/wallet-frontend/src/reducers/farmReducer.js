/*
Copyright 2019-present OmiseGO Pte Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License. */

import networkService from 'services/networkService';

const initialState = {
  totalFeeRate: 0,
  userRewardFeeRate: 0,
  poolInfo: {
    L1LP: {
      [networkService.L1_ETH_Address]: {},
    },
    L2LP: {
      [networkService.L2_ETH_Address]: {},
    }
  },
  userInfo: {
    L1LP: {
      [networkService.L1_ETH_Address]: {},
    },
    L2LP: {
      [networkService.L2_ETH_Address]: {},
    }
  },
  stakeToken: {
    symbol: "ETH",
    currency: networkService.L1_ETH_Address,
    LPAddress: networkService.L1LPAddress,
    L1orL2Pool: 'L1LP'
  },
  withdrawToken: {
    symbol: "ETH",
    currency: networkService.L1_ETH_Address,
    LPAddress: networkService.L1LPAddress,
    L1orL2Pool: 'L1LP'
  }
};

function farmReducer (state = initialState, action) {
  switch (action.type) {
    case 'GET_FARMINFO':
      return state;
    case 'GET_FARMINFO_SUCCESS':
      return {
        ...state,
        poolInfo: {
          L1LP: action.payload.L1PoolInfo,
          L2LP: action.payload.L2PoolInfo,
        },
        userInfo: {
          L1LP: action.payload.L1UserInfo,
          L2LP: action.payload.L2UserInfo,
        }
      }
    case 'GET_FEE':
      return state;
    case 'GET_FEE_SUCCESS':
      return { 
        ...state, 
        userRewardFeeRate: action.payload.userRewardFeeRate,
        totalFeeRate: action.payload.totalFeeRate,
      }
    case 'UPDATE_STAKE_TOKEN':
      return {
        ...state,
        stakeToken: action.payload,
      }
    case 'UPDATE_WITHDRAW_TOKEN':
      return {
        ...state,
        withdrawToken: action.payload,
      }
    default:
      return state;
  }
}

export default farmReducer;
