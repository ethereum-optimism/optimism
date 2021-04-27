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
import numbro from 'numbro';

import { selectByzantine } from 'selectors/statusSelector';
import { processExits, fetchExits } from 'actions/networkAction';

import GasPicker from 'components/gaspicker/GasPicker';
import Button from 'components/button/Button';
import Modal from 'components/modal/Modal';

import * as styles from './ProcessExitsModal.module.scss';

function ProcessExitsModal ({ exitData, open, toggle }) {
  const dispatch = useDispatch();

  const byzantineChain = useSelector(selectByzantine);
  const [ loading, setLoading ] = useState(false);
  const [ gasPrice, setGasPrice ] = useState();
  const [ selectedSpeed, setSelectedSpeed ] = useState('normal');

  async function submit () {
    setLoading(true);
    const res = await dispatch(processExits(exitData.queuePosition, exitData.currency, gasPrice));
    if (res) {
      await dispatch(fetchExits());
      setLoading(false);
      return handleClose();
    }
    return setLoading(false);
  }

  function handleClose () {
    setSelectedSpeed('normal');
    toggle();
  }

  return (
    <Modal open={open} onClose={handleClose}>
      <h2>Process Exit</h2>

      <div className={styles.note}>
        <span>This exit is currently</span>
        <span className={styles.position}>{exitData ? numbro(exitData.queuePosition).format({ output: 'ordinal' }) : ''}</span>
        <span>{`in the queue for this token. You will need to process ${exitData.queuePosition} ${exitData.queuePosition === 1 ? 'exit' : 'exits'} to release your funds.`}</span>
      </div>

      <GasPicker
        selectedSpeed={selectedSpeed}
        setSelectedSpeed={setSelectedSpeed}
        setGasPrice={setGasPrice}
      />

      <div className={styles.buttons}>
        <Button
          onClick={handleClose}
          type='outline'
          style={{ flex: 0 }}
        >
          CANCEL
        </Button>
        <Button
          onClick={submit}
          type='primary'
          style={{ flex: 0 }}
          loading={loading}
          tooltip='Your process exits transaction is still pending. Please wait for confirmation.'
          disabled={
            exitData.queueLength < 1 ||
            byzantineChain
          }
        >
          PROCESS
        </Button>
      </div>
    </Modal>
  );
}

export default ProcessExitsModal;
