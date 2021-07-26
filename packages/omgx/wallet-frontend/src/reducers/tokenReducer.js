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

const L1ETH = '0x0000000000000000000000000000000000000000'
const L2ETH = '0x4200000000000000000000000000000000000006'

const initialState = {
  [L1ETH]: {
    currency:  L1ETH,
    addressL1: L1ETH,
    addressL2: L2ETH,
    symbolL1:  'ETH',
    symbolL2:  'oETH',
    decimals:  18,
    name:      'Ethereum',
    redalert:  false,
  },
}

function tokenReducer(state = initialState, action) {
  switch (action.type) {
    case 'TOKEN/GET/SUCCESS':
      return {
        ...state,
        [action.payload.currency]: action.payload,
      }
    case 'TOKEN/GET/FAILURE':
      return {
        ...state,
        [action.payload.currency]: action.payload,
      }
    default:
      return state
  }
}

export default tokenReducer
