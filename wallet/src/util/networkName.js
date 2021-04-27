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

//import config from 'util/config';

const shortNetworkMapRev = {
  'Mainnet':   'main', 
  'Rinkeby':   'rinkeby',
  'RinkebyLR': 'rinkeby',
};

const linksWatcher = {
  'Mainnet':   'https://watcher-info.mainnet.v1.omg.network',
  'Rinkeby':   'https://watcher-info.rinkeby.v1.omg.network',
  'RinkebyLR': 'https://development-watcher-info-rinkeby-lr.omg.network',
};

const linksBlockexplorer = {
  'Mainnet':   'https://blockexplorer.mainnet.v1.omg.network',
  'Rinkeby':   'https://blockexplorer.rinkeby.v1.omg.network',
  'RinkebyLR': 'https://development-blockexplorer-rinkeby-lr.omg.network',
};

const plasmaAddress = {
  'Mainnet':   '0x0d4c1222f5e839a911e2053860e45f18921d72ac',
  'Rinkeby':   '0xb43f53394d86deab35bc2d8356d6522ced6429b5',
  'RinkebyLR': '0x9d9ad8a9baa52a10a6958dfe31ac504f6d62427d',
};

const etherscan = {
  'Mainnet':   'https://etherscan.io',
  'Rinkeby':   'https://rinkeby.etherscan.io',
  'RinkebyLR': 'https://rinkeby.etherscan.io',
};

export function getAllNetworks () {

  const rawPossibilities = 'Mainnet|Rinkeby|RinkebyLR'

  if (!rawPossibilities) {
    return [];
  }

  const options = rawPossibilities.split('|');
  const networks = [];

  options.forEach(option => {
    networks.push({
      name: option,
      shortName: shortNetworkMapRev[option],
      watcher: linksWatcher[option],
      blockexplorer: linksBlockexplorer[option],
      plasmaAddress: plasmaAddress[option],
      etherscan: etherscan[option],
    });
  });

  return networks;
}
