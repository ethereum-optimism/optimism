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

import { createAction } from 'actions/createAction';
import store from 'store';

jest.mock('store');
jest.mock('services/networkService');

function fakeAsyncRequestSuccess () {
  return Promise.resolve('toto-success');
}

function fakeAsyncRequestFailure () {
  return Promise.reject(Error('toto-failed'));
}

describe('createAction', () => {
  beforeEach(() => {
    store.clearActions();
    jest.clearAllMocks();
  });

  it('should return false to caller on async failure', async () => {
    const res = await store.dispatch(
      createAction('TEST/GET', () => fakeAsyncRequestFailure())
    );
    expect(res).toBe(false);
  });

  it('should return true to caller on async success', async () => {
    const res = await store.dispatch(
      createAction('TEST/GET', () => fakeAsyncRequestSuccess())
    );
    expect(res).toBe(true);
  });

  it('should dispatch request/success on successful async call', async () => {
    const expectedActions = [
      { type: 'TEST/GET/REQUEST' },
      { type: 'TEST/GET/SUCCESS', payload: 'toto-success' }
    ];
    await store.dispatch(
      createAction('TEST/GET', () => fakeAsyncRequestSuccess())
    );
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch request/error/uiError on failure of async call', async () => {
    const expectedActions = [
      { type: 'TEST/GET/REQUEST' },
      { type: 'TEST/GET/ERROR' },
      { type: 'UI/ERROR/UPDATE', payload: 'Something went wrong' }
    ];
    await store.dispatch(
      createAction('TEST/GET', () => fakeAsyncRequestFailure())
    );
    expect(store.getActions()).toEqual(expectedActions);
  });
});
