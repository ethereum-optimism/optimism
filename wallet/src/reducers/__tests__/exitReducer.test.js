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

import exitReducer from '../exitReducer';

describe('exitReducer', () => {
  it('should return the initial state', () => {
    const newState = exitReducer(undefined, { type: '@@INIT' });
    expect(newState).toEqual({ pending: {}, exited: {} });
  });

  it('should handle exit fetch success', () => {
    const action = {
      type: 'EXIT/GETALL/SUCCESS',
      payload: { pending: [ 'toto' ], exited: [ 'toto' ] }
    };
    const newState = exitReducer(undefined, action);
    expect(newState).toEqual(action.payload);
  });
});
