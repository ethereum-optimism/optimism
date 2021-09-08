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
const env = process.env.REACT_APP_ENV;

let NETWORKS;

NETWORKS = {
  rinkeby: {
    addressUrl:       `https://rinkeby.boba.network:8080/addresses.json`,
    addressOMGXUrl:   `https://rinkeby.boba.network:8078/addresses.json`,
    OMGX_WATCHER_URL: `https://api-watcher.rinkeby.boba.network/`,
    MM_Label:         `Rinkeby Test Network`,
    L1: {
      name: "Rinkeby",
      chainId: 4,
      rpcUrl: `https://rinkeby.infura.io/v3/${process.env.REACT_APP_INFURA_ID}`,
      blockExplorer: `https://api-rinkeby.etherscan.io/api?module=account&action=txlist&startblock=0&endblock=99999999&sort=asc&apikey=${process.env.REACT_APP_ETHERSCAN_API}`,
      transaction: `https://rinkeby.etherscan.io/tx/`,
    },
    L2: {
      name: "BOBA Rinkeby L2",
      chainId: 28,
      rpcUrl: `https://rinkeby.boba.network`,
      blockExplorer: `https://blockexplorer.rinkeby.boba.network/?network=Rinkeby`,
      transaction: `https://blockexplorer.rinkeby.boba.network/tx/`,
    }
  },
  mainnet: {
    addressUrl:       `https://mainnet.boba.network:8080/addresses.json`,
    addressOMGXUrl:   `https://mainnet.boba.network:8078/addresses.json`,
    OMGX_WATCHER_URL: `https://api-watcher.mainnet.boba.network/`,
    MM_Label:         `Ethereum Mainnet`,
    L1: {
      name: "Mainnet",
      chainId: 1,
      rpcUrl: `https://mainnet.infura.io/v3/${process.env.REACT_APP_INFURA_ID}`,
      blockExplorer: `https://api.etherscan.io/api?module=account&action=txlist&startblock=0&endblock=99999999&sort=asc&apikey=${process.env.REACT_APP_ETHERSCAN_API}`,
      transaction: ` https://etherscan.io/tx/`,
    },
    L2: {
      name: "BOBA L2",
      chainId: 288,
      rpcUrl: `https://mainnet.boba.network`,
      blockExplorer: `https://blockexplorer.rinkeby.boba.network/?network=Mainnet`,
      transaction: `https://blockexplorer.rinkeby.boba.network/tx/`,
    }
  }
}

if (env === 'dev') {
  NETWORKS = {
    local: {
      addressUrl:       `http://${window.location.hostname}:8080/addresses.json`,
      addressOMGXUrl:   `http://${window.location.hostname}:8078/addresses.json`,
      OMGX_WATCHER_URL: null, //Does not exist on local
      MM_Label:         `Local Network`,
      L1: {
        name: "Local L1",
        chainId: 31337,
        rpcUrl: `http://${window.location.hostname}:9545`,
        blockExplorer: null, //does not exist on local
      },
      L2: {
        name: "Local L2",
        chainId: 31338,
        rpcUrl: `http://${window.location.hostname}:8545`,
        blockExplorer: null, //does not exist on local
      },
    },
    rinkeby_integration: {
      addressUrl:       `https://rinkeby-integration.boba.network:8081/addresses.json`,
      addressOMGXUrl:   `https://rinkeby-integration.boba.network:8081/omgx-addr.json`,
      OMGX_WATCHER_URL: `https://api-watcher.rinkeby-integration.boba.network/`,
      MM_Label:         `Rinkeby Int Test Network`,
      L1: {
        name: "Rinkeby Int",
        chainId: 4,
        rpcUrl: `https://rinkeby.infura.io/v3/${process.env.REACT_APP_INFURA_ID}`,
        blockExplorer: `https://api-rinkeby.etherscan.io/api?module=account&action=txlist&startblock=0&endblock=99999999&sort=asc&apikey=${process.env.REACT_APP_ETHERSCAN_API}`,
        transaction: `https://rinkeby.etherscan.io/tx/`,
      },
      L2: {
        name: "BOBA Rinkeby Int L2",
        chainId: 29,
        rpcUrl: `https://rinkeby-integration.boba.network`,
        blockExplorer: `https://blockexplorer.rinkeby.boba.network/?network=Rinkeby%20Test`,
        transaction: `https://blockexplorer.rinkeby.boba.network/tx/`,
      }
    },
    ...NETWORKS,
  }
}

const BaseServices = {
  WALLET_SERVICE:   `https://api-service.boba.network/`,
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
  //ETH gas station
  ETH_GAS_STATION_URL: `https://ethgasstation.info/`,
}

export function getAllNetworks () {
  return NETWORKS;
}

export function getBaseServices () {
  return BaseServices;
}
