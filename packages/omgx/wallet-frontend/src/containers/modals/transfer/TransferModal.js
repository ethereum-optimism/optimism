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

import React, { useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';

import { transfer } from 'actions/networkAction';

import { closeModal, openAlert } from 'actions/uiAction';
import { selectLoading } from 'selectors/loadingSelector';

import Button from 'components/button/Button';
import Modal from 'components/modal/Modal';

import { logAmount } from 'util/amountConvert'
import networkService from 'services/networkService';

import * as styles from './TransferModal.module.scss';
import Input from 'components/input/Input';

function TransferModal ({ open, token }) {

  const dispatch = useDispatch()

  const [ value, setValue ] = useState('')
  const [ recipient, setRecipient ] = useState('')

  const loading = useSelector(selectLoading([ 'TRANSFER/CREATE' ]));

  async function submit () {
    if (
      value > 0 &&
      token.address &&
      recipient
    ) {
      try {
        const transferResponse = await dispatch(transfer(recipient, value, token.address));
        if (transferResponse) {
          dispatch(openAlert('Transaction submitted'));
          handleClose();
        }
      } catch (err) {
        //guess not really?
      }
    }
  }

  function handleClose () {
    setValue('')
    setRecipient('')
    dispatch(closeModal('transferModal'))
  }

  const disabledTransfer = value <= 0 ||
    !token.address ||
    !recipient

  function renderTransferScreen () {

    if(typeof(token) === 'undefined') return

    return (
      <>
        <h2>Transfer</h2>
        
        <div className={styles.address}>
          {`From address: ${networkService.account}`}
        </div>

        <Input
          label='To Address'
          placeholder='Hash or ENS name'
          paste
          value={recipient}
          onChange={i => setRecipient(i.target.value)}
        />

        <Input
          placeholder={`Amount to transfer`}
          value={value}
          type="number"
          onChange={(i) => {setValue(i.target.value)}}
          unit={token.symbol}
          maxValue={logAmount(token.balance, token.decimals)}
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
            onClick={()=>{submit({useLedgerSign: false})}}
            type='primary'
            loading={loading}
            tooltip='Your transfer is still pending. Please wait for confirmation.'
            disabled={disabledTransfer}
            triggerTime={new Date()}
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
