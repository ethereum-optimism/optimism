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
import BN from 'bignumber.js';

import { transfer } from 'actions/networkAction';

import { selectChildchainBalance } from 'selectors/balanceSelector';
import { selectLoading } from 'selectors/loadingSelector';
import { closeModal, openAlert } from 'actions/uiAction';

import Button from 'components/button/Button';
import Modal from 'components/modal/Modal';
import Input from 'components/input/Input';
import InputSelect from 'components/inputselect/InputSelect';

import networkService from 'services/networkService';
import { logAmount } from 'util/amountConvert';

import * as styles from './TransferModal.module.scss';

function TransferModal ({ open }) {
  const dispatch = useDispatch();

  const [ currency, setCurrency ] = useState('');
  const [ value, setValue ] = useState('');
  const [ recipient, setRecipient ] = useState('');

  const balances = useSelector(selectChildchainBalance, isEqual);

  const loading = useSelector(selectLoading([ 'TRANSFER/CREATE' ]));

  useEffect(() => {
    if (balances.length && !currency) {
      setCurrency(balances[0].currency);
    }
  }, [ balances, currency, open ]);

  const selectOptions = balances.map(i => ({
    title: i.symbol,
    value: i.currency,
    subTitle: `Balance: ${logAmount(i.amount, i.decimals)}`
  }));

  async function submit () {
    if (
      value > 0 &&
      currency &&
      recipient
    ) {
      try {
        const transferResponse = await dispatch(transfer(recipient, value, currency));
        if (transferResponse) {
          dispatch(openAlert('Transaction was submitted'));
          handleClose();
        }
      } catch (err) {
        //
      }
    }
  }

  function handleClose () {
    setCurrency('');
    setValue('');
    setRecipient('');
    dispatch(closeModal('transferModal'));
  }

  const disabledTransfer = value <= 0 ||
    !currency ||
    !recipient ||
    new BN(value).gt(new BN(getMaxTransferValue()));

  function getMaxTransferValue () {

    const transferingBalanceObject = balances.find(i => i.currency === currency);
    if (!transferingBalanceObject) {
      return;
    }
    return logAmount(transferingBalanceObject.amount, transferingBalanceObject.decimals);
  }

  function renderTransferScreen () {
    return (
      <>
        <h2>Transfer</h2>
        
        <div className={styles.address}>
          {`From address : ${networkService.account}`}
        </div>

        <Input
          label='To Address'
          placeholder='Hash or ENS name'
          paste
          value={recipient}
          onChange={i => setRecipient(i.target.value)}
        />

        <InputSelect
          label='Amount to transfer'
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
            type='secondary'
            className={styles.button}
          >
            CANCEL
          </Button>

          <Button
            className={styles.button}
            onClick={() => {
              submit({ useLedgerSign: false });
            }}
            type='primary'
            loading={loading}
            tooltip='Your transfer transaction is still pending. Please wait for confirmation.'
            disabled={disabledTransfer}
          >
            TRANSFER
          </Button>
        </div>
      </>
    );
  }

  return (
    <Modal open={open}>
      {renderTransferScreen()}
    </Modal>
  );
}

export default React.memo(TransferModal);
