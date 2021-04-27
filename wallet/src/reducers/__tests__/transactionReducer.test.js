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

import transactionReducer from '../transactionReducer';

describe('transactionReducer', () => {
  it('should return the initial state', () => {
    const newState = transactionReducer(undefined, { type: '@@INIT' });
    expect(newState).toEqual({});
  });

  it('should handle transaction fetch success', () => {
    const action = {
      type: 'TRANSACTION/GETALL/SUCCESS',
      payload: [ { txhash: '0x1', metadata: 'toto' } ]
    };
    const newState = transactionReducer(undefined, action);
    expect(newState).toEqual({
      '0x1': { txhash: '0x1', metadata: 'toto' }
    });
  });

  it('should handle transfer create success', () => {
    const action = {
      type: 'TRANSFER/CREATE/SUCCESS',
      payload: { txhash: '0x1', metadata: 'toto' }
    };
    const newState = transactionReducer(undefined, action);
    expect(newState).toEqual({
      '0x1': { txhash: '0x1', metadata: 'toto' }
    });
  });
});
