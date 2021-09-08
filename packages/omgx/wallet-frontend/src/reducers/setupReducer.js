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

//localStorage.removeItem("masterConfig")
//localStorage.removeItem("netLayer")

let masterConfigCache = localStorage.getItem("masterConfig")

if (masterConfigCache) {
  masterConfigCache = JSON.parse(masterConfigCache)
}

let netLayerCache = localStorage.getItem("netLayer")

if (netLayerCache) {
  netLayerCache = JSON.parse(netLayerCache)
}

const initialState = {
  walletMethod: null,
  masterConfig: masterConfigCache ? masterConfigCache : 'mainnet',
  blockexplorerURL: '',
  etherscan: '',
  minter: false,
  netLayer: netLayerCache ? netLayerCache : 'L1'
};

function setupReducer (state = initialState, action) {
  switch (action.type) {
    case 'SETUP/WALLET_METHOD/SET':
      return { 
        ...state, 
        walletMethod: action.payload 
      }
    case 'SETUP/NETWORK/SET':
      localStorage.setItem("masterConfig", JSON.stringify(action.payload))
      return { 
      	...state, 
        masterConfig: action.payload
      }
    case 'SETUP/LAYER/SET':
      localStorage.setItem("netLayer", JSON.stringify(action.payload))
      return { 
        ...state, 
        netLayer: action.payload
      }
    case 'SETUP/NFT/MINTER':
      return { 
        ...state, 
        minter: action.payload
      }
    default:
      return state;
  }
}

export default setupReducer;