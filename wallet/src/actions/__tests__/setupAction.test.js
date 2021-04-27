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

import * as actions from 'actions/setupAction';
import store from 'store';

jest.mock('store');

describe('setupActions', () => {
  beforeEach(() => {
    store.clearActions();
    jest.clearAllMocks();
  });

  it('should dispatch correct actions on setWalletMethod', async () => {
    const expectedActions = [ { type: 'SETUP/WALLET_METHOD/SET', payload: 'WalletLink' } ];
    await store.dispatch(actions.setWalletMethod('WalletLink'));
    expect(store.getActions()).toEqual(expectedActions);
  });
});
