/* eslint-disable quotes */
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

import React, { useState, useEffect } from 'react';
import { useDispatch } from 'react-redux';
import { CircularProgress } from '@material-ui/core';
import { closeModal, ledgerConnect } from 'actions/uiAction';

import Button from 'components/button/Button';
import Modal from 'components/modal/Modal';

import ledger from 'images/ledger_connect.png';
import boxarrow from 'images/boxarrow.svg';
import eth from 'images/eth.svg';
import key from 'images/key.svg';
import lock from 'images/lock.svg';

import { openError } from 'actions/uiAction';
import networkService from 'services/networkService';

import * as styles from './LedgerConnect.module.scss';

const steps = {
  usingLedger: 'USING_LEDGER',
  selectAddress: 'SELECT_ADDRESS'
};

function LedgerConnect ({ submit, open }) {
  const dispatch = useDispatch();
  const [ loading, setLoading ] = useState(false);
  const [ getConnectedAddressLoading, setGetConnectedAddressLoading ] = useState(false);
  const [ contractDataError, setContractDataError ] = useState(false);

  const [ step, setStep ] = useState(steps.usingLedger);

  const [ selectedAddress, setAddress ] = useState('');
  const [ selectedPath, setPath ] = useState('');

  useEffect(() => {
    async function fetchConnectedAddress () {
      setGetConnectedAddressLoading(true);
      try {
        const { path, address } = await networkService.getConnectedLedgerAddress();
        setAddress(address);
        setPath(path);
      } catch (error) {
        dispatch(openError('Configured Web3 account not one of the first 10 derivation paths on your Ledger. Please make sure your Web3 provider is pointing to the Ledger.'));
        setStep(steps.usingLedger);
      } finally {
        setGetConnectedAddressLoading(false);
      }
    }

    if (step === steps.selectAddress) {
      fetchConnectedAddress();
    }
  }, [ dispatch, step ]);

  function handleClose () {
    dispatch(closeModal('ledgerConnectModal'));
  }

  async function handleAddressConfirm () {
    dispatch(ledgerConnect(selectedPath));
    dispatch(closeModal('ledgerConnectModal'));
  }

  async function handleYes () {
    setLoading(true);
    setContractDataError(false);
    const ledgerConfig = await networkService.getLedgerConfiguration();
    setLoading(false);

    if (!ledgerConfig.connected) {
      return dispatch(openError('Could not connect to the Ledger. Please check that your Ledger is unlocked and the Ethereum application is open.'));
    }

    // check eth app is greater than or equal to 1.4.0
    const version = ledgerConfig.version.split('.').map(Number);
    if (
      version[0] < 1 ||
      (version[0] === 1 && version[1] < 4)
    ) {
      return dispatch(openError(`Ethereum application version ${ledgerConfig.version} is unsupported. Please install version 1.4.0 or greater on your device.`));
    }

    if (!ledgerConfig.dataEnabled) {
      setContractDataError(true);
      return dispatch(openError('Contract Data is not configured correctly. Please follow the steps outlined to allow Contract Data.'));
    }

    return setStep(steps.selectAddress);
  }

  return (
    <Modal open={open}>
      <div className={styles.header}>
        <div className={styles.logoContainer}>
          <img src={ledger} className={styles.logo} alt='ledger_logo' />
          {getConnectedAddressLoading && (
            <div className={styles.spinner}>
              <CircularProgress className={styles.spinnerGraphic} size={15} color='inherit' />
            </div>
          )}
        </div>
      </div>

      {step === steps.usingLedger && (
        <>
          <div className={styles.title}>
            {contractDataError ? 'Allow Contract Data' : 'Are you connecting with Ledger?'}
          </div>

          {contractDataError && (
            <>
              <div className={styles.description}>
                Please check the following steps to make sure Contract Data is allowed in your Ledger&apos;s Ethereum application settings.
              </div>
              <div className={styles.steps}>
                <div className={styles.step}>
                  <div className={styles.iconWrapper}>
                    <img src={lock} alt='lock' />
                  </div>
                  <div className={styles.text}>1. Connect and unlock your Ledger device.</div>
                </div>
                <div className={styles.step}>
                  <div className={styles.iconWrapper}>
                    <img src={eth} alt='eth' />
                  </div>
                  <div className={styles.text}>2. Open the Ethereum application.</div>
                </div>
                <div className={styles.step}>
                  <div className={styles.iconWrapper}>
                    <img src={boxarrow} alt='boxarrow' />
                  </div>
                  <div className={styles.text}>3. Press the right button to navigate to Settings. Then press both buttons to validate.</div>
                </div>
                <div className={styles.step}>
                  <div className={styles.iconWrapper}>
                    <img src={key} alt='key' />
                  </div>
                  <div className={styles.text}>4. In the Contract data settings, press both buttons to allow contract data in transactions. The device displays Allowed.</div>
                </div>
              </div>
            </>
          )}

          <div className={styles.buttons}>
            <Button onClick={handleClose} type='outline' className={styles.button}>
              {contractDataError ? 'CANCEL' : 'NO'}
            </Button>
            <Button
              className={styles.button}
              onClick={handleYes}
              type='primary'
              loading={loading}
            >
              {contractDataError ? 'CONTINUE' : 'YES'}
            </Button>
          </div>
        </>
      )}

      {step === steps.selectAddress && (
        <div className={styles.selectAddressContainer}>
          <div className={styles.title}>Confirm Address</div>

          {getConnectedAddressLoading && (
            <div className={styles.steps}>
              <div className={styles.step}>
                <div className={styles.iconWrapper}>
                  <img src={eth} alt='eth' />
                </div>
                <div className={styles.text}>
                  Verifying Web3 connected to Ledger...
                </div>
              </div>
            </div>
          )}

          {!getConnectedAddressLoading && (
            <>
              <div className={styles.description}>Confirm the Ledger address you will use with this wallet.</div>

              <div className={styles.steps}>
                <div className={styles.step}>
                  <div className={styles.iconWrapper}>
                    <img src={eth} alt='eth' />
                  </div>
                  <div className={styles.text}>
                    {selectedAddress}
                  </div>
                </div>
              </div>

              <div className={styles.buttons}>
                <Button onClick={() => setStep(steps.usingLedger)} type='outline' className={styles.button}>
                  CANCEL
                </Button>
                <Button
                  className={styles.button}
                  onClick={handleAddressConfirm}
                  type='primary'
                >
                  CONFIRM
                </Button>
              </div>
            </>
          )}
        </div>
      )}
    </Modal>
  );
}

export default React.memo(LedgerConnect);
