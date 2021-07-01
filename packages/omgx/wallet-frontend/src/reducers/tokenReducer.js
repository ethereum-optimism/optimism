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

const ETH0x = '0x0000000000000000000000000000000000000000';
const oETH = '0x4200000000000000000000000000000000000006';

const initialState = {
  [ETH0x]: {
    currency: ETH0x,
    decimals: 18,
    symbol: 'ETH',
    name: 'Ethereum'
  },
  [oETH]: {
    currency: oETH,
    decimals: 18,
    symbol: 'oETH',
    name: 'Ethereum',
  }
};

function tokenReducer (state = initialState, action) {
  switch (action.type) {
    case 'TOKEN/GET/SUCCESS':
      return { 
        ...state, 
        [action.payload.currency]: action.payload,
      };
    case 'TOKEN/GET/FAILURE':
      return {
        ...state,
        [action.payload.currency]: action.payload,
      }
    default:
      return state;
  }
}

export default tokenReducer;
