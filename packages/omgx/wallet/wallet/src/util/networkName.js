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

import { INFURA_ID } from "Settings";

const NETWORKS = {
  local: {
    L1: {
      name: "Local L1",
      chainId: 31337,
      rpcUrl: "http://" + window.location.hostname + ":9545",
      blockExplorer: "",
    },
    L2: {
      name: "Local L2",
      chainId: 420,
      rpcUrl: "http://" + window.location.hostname + ":8545",
      blockExplorer: "",
    },
  },
  rinkeby: {
    L1: {
      name: "Rinkeby L1",
      chainId: 4,
      rpcUrl: `https://rinkeby.infura.io/v3/${INFURA_ID}`,
      blockExplorer: `https://rinkeby.etherscan.io/${INFURA_ID}`,
    },
    L2: {
      name: "Rinkeby L2",
      chainId: 28,
      rpcUrl: "http://3.85.224.26:8545",
      blockExplorer: "",
    }
  }
}

export function getAllNetworks () {
  return NETWORKS;
}