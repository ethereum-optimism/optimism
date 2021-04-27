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

import * as actions from 'actions/uiAction';
import store from 'store';

jest.mock('store');

describe('uiActions', () => {
  beforeEach(() => {
    store.clearActions();
    jest.clearAllMocks();
  });

  it('should dispatch correct actions on openModal', async () => {
    const expectedActions = [ { type: 'UI/MODAL/OPEN', payload: 'toto' } ];
    await store.dispatch(actions.openModal('toto'));
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on closeModal', async () => {
    const expectedActions = [ { type: 'UI/MODAL/CLOSE', payload: 'toto' } ];
    await store.dispatch(actions.closeModal('toto'));
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on openAlert', async () => {
    const expectedActions = [ { type: 'UI/ALERT/UPDATE', payload: 'toto' } ];
    await store.dispatch(actions.openAlert('toto'));
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on closeAlert', async () => {
    const expectedActions = [ { type: 'UI/ALERT/UPDATE', payload: null } ];
    await store.dispatch(actions.closeAlert());
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on openError', async () => {
    const expectedActions = [ { type: 'UI/ERROR/UPDATE', payload: 'toto' } ];
    await store.dispatch(actions.openError('toto'));
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on closeError', async () => {
    const expectedActions = [ { type: 'UI/ERROR/UPDATE', payload: null } ];
    await store.dispatch(actions.closeError());
    expect(store.getActions()).toEqual(expectedActions);
  });
});
