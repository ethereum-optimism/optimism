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

import React from 'react';
import { CircularProgress } from '@material-ui/core';

import { hashTypedDataMessage, hashTypedDataDomain } from '@omisego/omg-js-util';

import Button from 'components/button/Button';

import ledger from 'images/ledger_connect.png';
import eth from 'images/eth.svg';
import connect from 'images/connect.svg';
import lock from 'images/lock.svg';

import * as styles from './LedgerPrompt.module.scss';

function LedgerPrompt ({
  loading,
  submit,
  handleClose,
  typedData
}) {
  return (
    <>
      {!loading && (
        <>
          <div className={styles.header}>
            <div className={styles.logoContainer}>
              <img src={ledger} className={styles.logo} alt='ledger_logo' />
            </div>
            <div className={styles.title}>Ledger Sign</div>
          </div>
          <div className={styles.description}>Please make sure your Ledger meets the following conditions:</div>
          <div className={styles.steps}>
            <div className={styles.step}>
              <div className={styles.iconWrapper}>
                <img src={connect} alt='connect' />
              </div>
              <div className={styles.text}>Connected</div>
            </div>
            <div className={styles.step}>
              <div className={styles.iconWrapper}>
                <img src={lock} alt='lock' />
              </div>
              <div className={styles.text}>Unlocked</div>
            </div>
            <div className={styles.step}>
              <div className={styles.iconWrapper}>
                <img src={eth} alt='eth' />
              </div>
              <div className={styles.text}>The Ethereum application is open and running a version greater than 1.4.0</div>
            </div>
          </div>
          <div className={styles.buttons}>
            <Button
              onClick={handleClose}
              type='outline'
              className={styles.button}
            >
              CANCEL
            </Button>
            <Button
              className={styles.button}
              onClick={() => submit({ useLedgerSign: true })}
              type='primary'
              loading={loading}
            >
              SIGN
            </Button>
          </div>
        </>
      )}

      {loading && (
        <>
          <div className={styles.header}>
            <div className={styles.logoContainer}>
              <img src={ledger} className={styles.logo} alt='ledger_logo' />
              <div className={styles.spinner}>
                <CircularProgress className={styles.spinnerGraphic} size={15} color='inherit' />
              </div>
            </div>
            <div className={styles.title}>Processing...</div>
          </div>
          <div className={styles.description}>
            Please continue signing the transaction on your Ledger device.<br />
            Check that the domain and message hash displayed on the device match the following:
          </div>
          {typedData && (
            <div className={styles.steps}>
              <div className={[ styles.step, styles.code ].join(' ')}>
                Domain Hash: {hashTypedDataDomain(typedData).toUpperCase()}
              </div>
              <div className={[ styles.step, styles.code ].join(' ')}>
                Message Hash: {hashTypedDataMessage(typedData).toUpperCase()}
              </div>
            </div>
          )}
        </>
      )}
    </>
  );
}

export default React.memo(LedgerPrompt);
