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

import gasReducer from '../gasReducer';

describe('gasReducer', () => {
  it('should return the initial state', () => {
    const newState = gasReducer(undefined, { type: '@@INIT' });
    expect(newState).toEqual({ slow: 0, normal: 0, fast: 0 });
  });

  it('should handle gas fetch success', () => {
    const action = {
      type: 'GAS/GET/SUCCESS',
      payload: { slow: 1, normal: 10, fast: 100 }
    };
    const newState = gasReducer(undefined, action);
    expect(newState).toEqual(action.payload);
  });
});
