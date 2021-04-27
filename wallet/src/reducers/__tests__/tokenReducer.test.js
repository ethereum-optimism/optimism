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

import tokenReducer from '../tokenReducer';

describe('tokenReducer', () => {
  it('should return the initial state', () => {
    const newState = tokenReducer(undefined, { type: '@@INIT' });
    expect(newState).toEqual({
      '0x0000000000000000000000000000000000000000': {
        currency: '0x0000000000000000000000000000000000000000',
        decimals: 18,
        name: 'ETH'
      }
    });
  });

  it('should handle token fetch success', () => {
    const action = {
      type: 'TOKEN/GET/SUCCESS',
      payload: { currency: '0xomg', decimals: 18, name: 'OMG' }
    };
    const newState = tokenReducer(undefined, action);
    expect(newState).toEqual({
      '0x0000000000000000000000000000000000000000': { currency: '0x0000000000000000000000000000000000000000', decimals: 18, name: 'ETH' },
      '0xomg': { currency: '0xomg', decimals: 18, name: 'OMG' }
    });
  });
});
