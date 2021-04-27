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

import setupReducer from '../setupReducer';

describe('setupReducer', () => {
  it('should return the initial state', () => {
    const initialState = { walletMethod: 'WalletConnect' };
    const newState = setupReducer(initialState, { type: 'ACTION/NOT/EXIST' });
    expect(newState).toEqual(initialState);
  });

  it('should handle setting the wallet method', () => {
    const initialState = { walletMethod: null };
    const action = {
      type: 'SETUP/WALLET_METHOD/SET',
      payload: 'WalletConnect'
    };
    const newState = setupReducer(initialState, action);
    expect(newState).toEqual({ ...initialState, walletMethod: 'WalletConnect' });
  });
});
