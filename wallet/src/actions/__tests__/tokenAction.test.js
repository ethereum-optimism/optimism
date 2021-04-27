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

import { getToken } from 'actions/tokenAction';
import networkService from 'services/networkService';
import store from 'store';

jest.mock('services/networkService');
jest.mock('store');

describe('tokenAction', () => {
  beforeEach(() => {
    store.clearActions();
    jest.clearAllMocks();
  });

  it('should return token information using getToken', async () => {
    const tokenInfo = await getToken('0x123');
    expect(networkService.web3.eth.Contract).toHaveBeenCalled();
    expect(tokenInfo).toEqual({
      currency: '0x123',
      decimals: 18,
      name: 'OMG'
    });
  });

  it('should return early if token info already fetched', async () => {
    await getToken('0x0000000000000000000000000000000000000000');
    expect(networkService.web3.eth.Contract).not.toHaveBeenCalled();
    expect(store.getActions()).toEqual([]);
  });

  it('should dispatch token success using getToken', async () => {
    await getToken('0x123');
    const expectedActions = [ {
      type: 'TOKEN/GET/SUCCESS',
      payload: {
        currency: '0x123',
        decimals: 18,
        name: 'OMG'
      }
    } ];
    expect(store.getActions()).toEqual(expectedActions);
  });
});
