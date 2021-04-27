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

import { WebWalletError } from 'services/errorService';

export function createAction (key, asyncAction) {
  return async function (dispatch) {
    dispatch({ type: `${key}/REQUEST` });
    try {
      const response = await asyncAction();
      dispatch({ type: `${key}/SUCCESS`, payload: response });
      return true;
    } catch (error) {
      // cancel request loading state
      dispatch({ type: `${key}/ERROR` });

      const _error = error instanceof WebWalletError
        ? error
        : new WebWalletError({
          originalError: error,
          customErrorMessage: 'Something went wrong',
          reportToSentry: true,
          reportToUi: true
        });

      // perform necessary reports
      _error.report(dispatch);
      return false;
    }
  };
}
