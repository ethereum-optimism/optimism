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

import balanceReducer from '../balanceReducer';

describe('balanceReducer', () => {
  it('should return the initial state', () => {
    const newState = balanceReducer(undefined, { type: '@@INIT' });
    expect(newState).toEqual({ rootchain: [], childchain: [] });
  });

  it('should handle balance success', () => {
    const action = {
      type: 'BALANCE/GET/SUCCESS',
      payload: { rootchain: [ 'toto' ], childchain: [ 'toto' ] }
    };
    const newState = balanceReducer(undefined, action);
    expect(newState).toEqual(action.payload);
  });
});
