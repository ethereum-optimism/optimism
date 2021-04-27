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

import feeReducer from '../feeReducer';

describe('feeReducer', () => {
  it('should return the initial state', () => {
    const newState = feeReducer(undefined, { type: '@@INIT' });
    expect(newState).toEqual({});
  });

  it('should handle fee fetch success', () => {
    const action = {
      type: 'FEE/GET/SUCCESS',
      payload: [
        { currency: 'ETH', amount: 1 },
        { currency: 'OMG', amount: 10 }
      ]
    };
    const newState = feeReducer(undefined, action);
    expect(newState).toEqual({
      ETH: { currency: 'ETH', amount: 1 },
      OMG: { currency: 'OMG', amount: 10 }
    });
  });
});
