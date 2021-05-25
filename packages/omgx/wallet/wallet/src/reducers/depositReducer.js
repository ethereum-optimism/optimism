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

import { keyBy } from 'lodash';

const initialState = {
  eth: {},
  erc20: {}
};

function depositReducer (state = initialState, action) {
  switch (action.type) {
    case 'DEPOSIT/CREATE/SUCCESS':
      const isEth = action.payload.isEth;
      if (isEth) {
        return {
          ...state,
          eth: {
            ...state.eth,
            [action.payload.transactionHash]: action.payload
          }
        };
      }
      return {
        ...state,
        erc20: {
          ...state.erc20,
          [action.payload.transactionHash]: action.payload
        }
      };
    case 'DEPOSIT/CHECKALL/SUCCESS':
    case 'DEPOSIT/GETALL/SUCCESS':
      const { eth, erc20 } = action.payload;
      return {
        ...state,
        eth: {
          ...state.eth,
          ...keyBy(eth, 'transactionHash')
        },
        erc20: {
          ...state.erc20,
          ...keyBy(erc20, 'transactionHash')
        }
      };
    default:
      return state;
  }
}

export default depositReducer;
