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

const networkService = {
  OmgUtil: {
    ethErrorReason: jest.fn().mockImplementationOnce(() => Promise.resolve('toto'))
  },
  web3: {
    eth: {
      Contract: jest.fn(() => ({
        methods: {
          symbol: () => ({
            call: jest.fn(() => Promise.resolve('OMG'))
          }),
          decimals: () => ({
            call: jest.fn(() => Promise.resolve(18))
          })

        }
      }))
    }
  },
  checkStatus: jest.fn(() => Promise.resolve('toto')),
  getBalances: jest.fn(() => Promise.resolve('toto')),
  getAllTransactions: jest.fn(() => Promise.resolve('toto')),
  getDeposits: jest.fn(() => Promise.resolve('toto')),
  getExits: jest.fn(() => Promise.resolve('toto')),
  checkForExitQueue: jest.fn(() => Promise.resolve(true)),
  getExitQueue: jest.fn(() => Promise.resolve('toto')),
  addExitQueue: jest.fn(() => Promise.resolve('toto')),
  exitUtxo: jest.fn(() => Promise.resolve('toto')),
  depositEth: jest.fn(() => Promise.resolve('toto')),
  processExits: jest.fn(() => Promise.resolve('toto')),
  transfer: jest.fn(() => Promise.resolve('toto')),
  mergeUtxos: jest.fn(() => Promise.resolve('toto')),
  getGasPrice: jest.fn(() => Promise.resolve('toto')),
  fetchFees: jest.fn(() => Promise.resolve([ 1, 2, 3 ]))
};

export default networkService;
