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

import * as Sentry from '@sentry/react';
import config from 'util/config';

if (config.sentry) {
  Sentry.init({ dsn: config.sentry });
}

const errorCache = [];
const noLogErrors = [
  'user denied',
  'user rejected',
  'user canceled',
  'user cancelled'
];

export class WebWalletError extends Error {
  constructor ({
    originalError,
    customErrorMessage,
    reportToSentry,
    reportToUi
  }) {
    super(originalError.message);
    this._originalError = originalError;
    this._customErrorMessage = customErrorMessage;
    this._reportToSentry = reportToSentry;
    this._reportToUi = reportToUi;
  }

  // report (dispatchMethod) {
  report () {
    const metamaskHeaderNotFoundCode = -3200;
    if (
      noLogErrors.find(i => this._originalError.message && this._originalError.message.includes(i)) ||
      this._originalError.code === metamaskHeaderNotFoundCode
    ) {
      return;
    }

    if (this._reportToSentry && config.sentry) {
      if (!errorCache.includes(this._originalError.message)) {
        errorCache.push(this._originalError.message);
        try {
          Sentry.captureException(this._originalError);
        } catch (error) {
          //
        }
      }
    }

    if (this._reportToUi) {
      // dispatchMethod(
      //   // openError(this._customErrorMessage || this._originalError.message || 'Something went wrong.')
      // );
    }
  }
}
