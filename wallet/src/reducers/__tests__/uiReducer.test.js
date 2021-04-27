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

import uiReducer from '../uiReducer';

const initialState = {
  depositModal: false,
  transferModal: false,
  exitModal: false,
  mergeModal: false,
  wrongNetworkModal: false,
  ledger: false,
  ledgerConnectModal: false,
  activeHistoryTab: 'Transactions',
  alert: null,
  error: null
};

describe('uiReducer', () => {
  it('should return the initial state', () => {
    const newState = uiReducer(undefined, { type: '@@INIT' });
    expect(newState).toEqual(initialState);
  });

  it('should handle modal open', () => {
    const action = {
      type: 'UI/MODAL/OPEN',
      payload: 'depositModal'
    };
    const newState = uiReducer(undefined, action);
    expect(newState).toEqual({ ...initialState, depositModal: true });
  });

  it('should handle modal close', () => {
    const action = {
      type: 'UI/MODAL/CLOSE',
      payload: 'depositModal'
    };
    const newState = uiReducer(undefined, action);
    expect(newState).toEqual(initialState);
  });

  it('should handle new alert', () => {
    const action = {
      type: 'UI/ALERT/UPDATE',
      payload: 'oops'
    };
    const newState = uiReducer(undefined, action);
    expect(newState).toEqual({ ...initialState, alert: 'oops' });
  });

  it('should handle new error', () => {
    const action = {
      type: 'UI/ERROR/UPDATE',
      payload: 'oops'
    };
    const newState = uiReducer(undefined, action);
    expect(newState).toEqual({ ...initialState, error: 'oops' });
  });
});
