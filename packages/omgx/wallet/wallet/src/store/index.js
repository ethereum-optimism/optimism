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

import reduxThunk from 'redux-thunk';
import * as Sentry from '@sentry/react';
import { composeWithDevTools } from 'redux-devtools-extension';
import { createStore, applyMiddleware } from 'redux';
import reducers from 'reducers';

const initialState = {};

const sentryReduxEnhancer = Sentry.createReduxEnhancer({
  configureScopeWithState: (scope, state) => {
    scope.setTag('wallet-method', state.setup.walletMethod);
  },
  stateTransformer: state => {
    return {
      status: state.status,
      ui: state.ui,
      setup: state.setup
    };
  }
});

const store = createStore(
  reducers,
  initialState,
  composeWithDevTools (
    applyMiddleware(reduxThunk),
    sentryReduxEnhancer
  )
);

export default store;
