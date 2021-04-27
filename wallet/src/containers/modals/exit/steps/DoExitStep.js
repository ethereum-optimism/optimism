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
import { useDispatch, useSelector } from 'react-redux';
import { isEqual } from 'lodash';

import { selectChildchainBalance } from 'selectors/balanceSelector';

import { exitOptimism } from 'actions/networkAction';
import { openAlert } from 'actions/uiAction';
import { selectLoading } from 'selectors/loadingSelector';

import InputSelect from 'components/inputselect/InputSelect';
import Button from 'components/button/Button';

import { logAmount } from 'util/amountConvert';

import networkService from 'services/networkService';

import * as styles from '../ExitModal.module.scss';


function DoExitStep ({
  handleClose,
}) {
  const dispatch = useDispatch();

  const [ currency, setCurrency ] = useState('');
  const [ value, setValue ] = useState('');

  const balances = useSelector(selectChildchainBalance, isEqual);

  useEffect(() => {
    if (balances.length && !currency) {
      setCurrency(balances[0].currency);
    }
  }, [ balances, currency ]);

  const selectOptions = balances.map(i => ({
    title: i.symbol,
    value: i.currency,
    subTitle: `Balance: ${logAmount(i.amount, i.decimals)}`
  }));

  const submitLoading = useSelector(selectLoading([ 'EXIT/CREATE' ]));

  async function doExit () {
    const networkStatus = await dispatch(networkService.checkNetwork('L2'));
    if (!networkStatus) return 
    const res = await dispatch(exitOptimism(currency, value));
    if (res) {
      dispatch(openAlert('Exit finished.'));
      handleClose();
    }
  }


  function getMaxTransferValue () {

    const transferingBalanceObject = balances.find(i => i.currency === currency);
    if (!transferingBalanceObject) {
      return;
    }
    return logAmount(transferingBalanceObject.amount, transferingBalanceObject.decimals);
  }

  return (
    <>
      <h2>Start Standard Exit</h2>

      <InputSelect
        label='Amount to exit'
        placeholder={0}
        value={value}
        onChange={i => {
          setValue(i.target.value);
        }}
        selectOptions={selectOptions}
        onSelect={i => {
          setCurrency(i.target.value);
        }}
        selectValue={currency}
        maxValue={getMaxTransferValue()}
      />

      <div className={styles.buttons}>
        <Button
          onClick={handleClose}
          className={styles.button}
          type='outline'
          style={{ flex: 0 }}
        >
          CANCEL
        </Button>
        <Button
          onClick={doExit}
          type='primary'
          style={{ flex: 0 }}
          loading={submitLoading}
          className={styles.button}
          tooltip='Your exit transaction is still pending. Please wait for confirmation.'
        >
          EXIT
        </Button>
      </div>
    </>
  );
}

export default React.memo(DoExitStep);
