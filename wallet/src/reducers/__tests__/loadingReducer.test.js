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

import loadingReducer from '../loadingReducer';

describe('loadingReducer', () => {
  it('should return the initial state', () => {
    const newState = loadingReducer(undefined, { type: '@@INIT' });
    expect(newState).toEqual({});
  });

  it('should handle request correctly', () => {
    const action = { type: 'TEST/GET/REQUEST' };
    const newState = loadingReducer(undefined, action);
    expect(newState).toEqual({ 'TEST/GET': true });
  });

  it('should cancel loading state on non request correctly', () => {
    const action = { type: 'TEST/GET/SUCCESS' };
    const newState = loadingReducer(undefined, action);
    expect(newState).toEqual({ 'TEST/GET': false });

    const action2 = { type: 'TEST/GET/ERROR' };
    const newState2 = loadingReducer(undefined, action2);
    expect(newState2).toEqual({ 'TEST/GET': false });
  });

  it('should pass through other action types', () => {
    const action = { type: 'TEST/MODAL/OPEN' };
    const newState = loadingReducer(undefined, action);
    expect(newState).toEqual({});
  });
});
