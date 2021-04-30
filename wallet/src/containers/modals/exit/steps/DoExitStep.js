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

import { exitOptimism, depositL2LP } from 'actions/networkAction';
import { openAlert } from 'actions/uiAction';
import { selectLoading } from 'selectors/loadingSelector';

import InputSelect from 'components/inputselect/InputSelect';
import Button from 'components/button/Button';

import { logAmount } from 'util/amountConvert';

import networkService from 'services/networkService';

import * as styles from '../ExitModal.module.scss';


function DoExitStep ({
  handleClose,
  fast
}) {
  const dispatch = useDispatch();

  const [ currency, setCurrency ] = useState('');
  const [ value, setValue ] = useState('');
  const [ LPBalance, setLPBalance ] = useState(0);
  const [ feeRate, setFeeRate ] = useState(0);

  const balances = useSelector(selectChildchainBalance, isEqual);

  useEffect(() => {
    if (balances.length && !currency) {
      setCurrency(balances[0].currency);
    }
    if (fast && currency) {
      networkService.L1LPBalance(currency).then((LPBalance)=>{
        setLPBalance(LPBalance)
      })
      networkService.getL2LPFeeRatio().then((feeRate)=>{
        setFeeRate(feeRate)
      })
    }
  }, [ balances, currency, fast ]);

  const selectOptions = balances.map(i => ({
    title: i.symbol,
    value: i.currency,
    subTitle: `Balance: ${logAmount(i.amount, i.decimals)}`
  }));

  const currencySymbols = balances.reduce((acc, cur) => {
    acc[cur.currency] = cur.symbol;
    return acc;
  }, {})

  const submitLoading = useSelector(selectLoading([ 'EXIT/CREATE' ]));

  async function doExit () {
    const networkStatus = await dispatch(networkService.checkNetwork('L2'));
    if (!networkStatus) return 
    let res;
    if (fast === false) {
      res = await dispatch(exitOptimism(currency, value));
    } else {
      res = await dispatch(depositL2LP(currency, value));
    }
    if (res) {
      if (fast === false) {
        dispatch(openAlert(`${currencySymbols[currency]} was exited to L1`));
      } else {
        dispatch(openAlert(`${currencySymbols[currency]} was deposited to liquidity pool. You got ${(Number(value) * 0.97).toFixed(2)} ${currencySymbols[currency]} on L1`));
      }
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
      
      {fast && currency ? (
        <>
          <h3>
            The L1 liquidity pool has {LPBalance} {currencySymbols[currency]}.
          </h3>
          <h3>
            The convenience fee is {feeRate}%. {value && `You are going to receive ${(Number(value) * 0.97).toFixed(2)} ${currencySymbols[currency]} on L1.`}
          </h3>
        </>
      ):<></>}

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
