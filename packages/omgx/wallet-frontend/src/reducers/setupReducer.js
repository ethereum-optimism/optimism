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

let networkNameCache = localStorage.getItem("networkName");

if (networkNameCache) {
  networkNameCache = JSON.parse(networkNameCache);
}

const initialState = {
  walletMethod: null,
  networkName: networkNameCache ? networkNameCache : 'rinkeby',
  blockexplorerURL: '',
  etherscan: '',
  minter: false
};

function setupReducer (state = initialState, action) {
  switch (action.type) {
    case 'SETUP/WALLET_METHOD/SET':
      return { 
        ...state, 
        walletMethod: action.payload 
      }
    case 'SETUP/NETWORK/SET':
      localStorage.setItem("networkName", JSON.stringify(action.payload));
      return { 
      	...state, 
        networkName: action.payload,
      	// networkName: action.payload.network.name,
        // blockexplorerURL: action.payload.network.blockexplorer,
        // etherscan: action.payload.network.etherscan,
      }
    case 'SETUP/LAYER/SET':
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