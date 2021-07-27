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

require('dotenv').config()

const NETWORKS = {
  local: {
    addressUrl:       `http://${window.location.hostname}:8080/addresses.json`,
    addressOMGXUrl:   `http://${window.location.hostname}:8078/addresses.json`,
    OMGX_WATCHER_URL: null,//Does not exist on local
    L1: {
      name: "Local L1",
      chainId: 31337,
      rpcUrl: `http://${window.location.hostname}:9545`,
      blockExplorer: null,//does not exist on local
      transaction: null, //TODO: ❓ @jan please provide the valid url here for local
    },
    L2: {
      name: "Local L2",
      chainId: 28,
      rpcUrl: `http://${window.location.hostname}:8545`,
      blockExplorer: null,//does not exist on local
      transaction: null, //TODO: ❓ @jan please provide the valid url here for local
    },
  },
  rinkeby: {
    addressUrl:       `https://rinkeby.omgx.network:8080/addresses.json`,
    addressOMGXUrl:   `https://rinkeby.omgx.network:8078/addresses.json`,
    OMGX_WATCHER_URL: `https://api-watcher.rinkeby.omgx.network/`,
    L1: {
      name: "Rinkeby L1",
      chainId: 4,
      rpcUrl: `https://rinkeby.infura.io/v3/${process.env.REACT_APP_INFURA_ID}`,
      blockExplorer: `https://api-rinkeby.etherscan.io/api?module=account&action=txlist&startblock=0&endblock=99999999&sort=asc&apikey=${process.env.REACT_APP_ETHERSCAN_API}`,
      transaction: `https://rinkeby.etherscan.io/tx/`,
    },
    L2: {
      name: "Rinkeby L2",
      chainId: 28,
      rpcUrl: `https://rinkeby.omgx.network`,
      blockExplorer: `https://blockexplorer.rinkeby.omgx.network`,
      transaction: `https://blockexplorer.rinkeby.omgx.network/tx/`,
    }
  },
/*
  mainnet: {
    addressUrl:       `https://mainnet.omgx.network:8080/addresses.json`,
    addressOMGXUrl:   `https://mainnet.omgx.network:8078/addresses.json`,
    OMGX_WATCHER_URL: `https://api-watcher.mainnet.omgx.network/`,
    L1: {
      name: "Mainnet L1",
      chainId: 4,
      rpcUrl: `https://mainnet.infura.io/v3/${process.env.REACT_APP_INFURA_ID}`,
      blockExplorer: `https://api-mainnet.etherscan.io/api?module=account&action=txlist&startblock=0&endblock=99999999&sort=asc&apikey=${process.env.REACT_APP_ETHERSCAN_API}`,
    },
    L2: {
      name: "Mainnet L2",
      chainId: 288,
      rpcUrl: `https://mainnet.omgx.network`,
      blockExplorer: `https://blockexplorer.mainnet.omgx.network`,
    }
  }
*/
}

const BaseServices = {
  WALLET_SERVICE:   `https://api-service.rinkeby.omgx.network/`,
  //relevant to local?
  SELLER_OPTIMISM_API_URL: `https://pm7f0dp9ud.execute-api.us-west-1.amazonaws.com/prod/`,
  //relevant to local?
  BUYER_OPTIMISM_API_URL: `https://n245h0ka3i.execute-api.us-west-1.amazonaws.com/prod/`,
  //relevant to local?
  SERVICE_OPTIMISM_API_URL: `https://zlba6djrv6.execute-api.us-west-1.amazonaws.com/prod/`,
  //relevant to local?
  WEBSOCKET_API_URL: `wss://d1cj5xnal2.execute-api.us-west-1.amazonaws.com/prod`,
  //Coing gecko url
  COIN_GECKO_URL: `https://api.coingecko.com/api/v3/`,
}

export function getAllNetworks () {
  return NETWORKS;
}

export function getBaseServices () {
  return BaseServices;
}