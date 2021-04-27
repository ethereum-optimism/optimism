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
import { orderBy } from 'lodash';
import { useDispatch, useSelector } from 'react-redux';
import { Check } from '@material-ui/icons';

import { selectLoading } from 'selectors/loadingSelector';
import { selectLedger } from 'selectors/uiSelector';
import { mergeUtxos } from 'actions/networkAction';
import { closeModal, openAlert, setActiveHistoryTab } from 'actions/uiAction';

import Button from 'components/button/Button';
import Modal from 'components/modal/Modal';

import LedgerPrompt from 'containers/modals/ledger/LedgerPrompt';

import networkService from 'services/networkService';
import { logAmount } from 'util/amountConvert';

import * as styles from './MergeModal.module.scss';

function MergeModal ({ open }) {
  const dispatch = useDispatch();

  const [ selectedUTXOs, setSelectedUTXOs ] = useState([]);
  const [ searchUTXO, setSearchUTXO ] = useState('');
  const [ utxos, setUtxos ] = useState([]);
  const [ ledgerModal, setLedgerModal ] = useState(false);
  const [ typedData, setTypedData ] = useState({});

  const loading = useSelector(selectLoading([ 'TRANSFER/CREATE' ]));
  const ledgerConnect = useSelector(selectLedger);

  useEffect(() => {
    async function fetchUTXOS () {
      const _utxos = await networkService.getUtxos();
      const utxos = orderBy(_utxos, i => i.currency, 'desc');
      setUtxos(utxos);
    }
    if (open) {
      fetchUTXOS();
    }
  }, [ open ]);

  useEffect(() => {
    if (selectedUTXOs.length) {
      setSearchUTXO(selectedUTXOs[0].currency);
      const { typedData } = networkService.getMergeTypedData(selectedUTXOs);
      setTypedData(typedData);
    }
    if (!selectedUTXOs.length) {
      setSearchUTXO('');
    }
  }, [ selectedUTXOs ]);

  async function submit ({ useLedgerSign }) {
    if (selectedUTXOs.length > 1 && selectedUTXOs.length < 5) {
      const res = await dispatch(mergeUtxos(useLedgerSign, selectedUTXOs));
      if (res) {
        dispatch(setActiveHistoryTab('Transactions'));
        dispatch(openAlert('Merge submitted. You will be blocked from making further transactions until the merge is confirmed.'));
        handleClose();
      }
    }
  }

  function handleClose () {
    setSelectedUTXOs([]);
    setSearchUTXO('');
    setLedgerModal(false);
    dispatch(closeModal('mergeModal'));
  }

  function handleUtxoClick (utxo) {
    const isSelected = selectedUTXOs.some(i => i.utxo_pos === utxo.utxo_pos);
    if (isSelected) {
      setSelectedUTXOs(selectedUTXOs.filter(i => i.utxo_pos !== utxo.utxo_pos));
    }
    if (!isSelected && selectedUTXOs.length < 4) {
      setSelectedUTXOs([ ...selectedUTXOs, utxo ]);
    }
  }

  function renderMergeScreen () {
    const _utxos = utxos
      .filter(i => i.currency.includes(searchUTXO))
      .filter(i => i);

    return (
      <>
        <h2>Merge UTXOs</h2>
        <div className={styles.disclaimer}>Select the UTXOs you want to merge</div>

        <div className={styles.list}>
          {!utxos.length && (
            <div className={styles.disclaimer}>You do not have any UTXOs on the OMG Network.</div>
          )}
          {_utxos.map((i, index) => {
            const selected = selectedUTXOs.some(selected => selected.utxo_pos === i.utxo_pos);
            return (
              <div
                key={index}
                onClick={() => handleUtxoClick(i)}
                className={[
                  styles.utxo,
                  selected ? styles.selected : ''
                ].join(' ')}
              >
                <div className={styles.title}>
                  {i.tokenInfo.symbol}
                </div>

                <div className={styles.value}>
                  <div className={styles.amount}>
                    {logAmount(i.amount.toString(), i.tokenInfo.decimals)}
                  </div>

                  <div className={styles.check}>
                    {selected && <Check />}
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        <div className={styles.disclaimer}>You can select a maximum of 4 UTXOs to merge at once.</div>

        <div className={styles.buttons}>
          <Button
            onClick={handleClose}
            type='outline'
            className={styles.button}
          >
            CANCEL
          </Button>
          {ledgerConnect ? (
            <Button
              onClick={() => setLedgerModal(true)}
              type='primary'
              className={styles.button}
              loading={loading}
              tooltip='Your merge transaction is still pending. Please wait for confirmation.'
              disabled={selectedUTXOs.length <= 1 || selectedUTXOs.length > 4}
            >
            MERGE WITH LEDGER
            </Button>) : (
            <Button
              onClick={submit}
              type='primary'
              className={styles.button}
              loading={loading}
              tooltip='Your merge transaction is still pending. Please wait for confirmation.'
              disabled={selectedUTXOs.length <= 1 || selectedUTXOs.length > 4}
            >
            MERGE
            </Button>)}
        </div>
      </>
    );
  }

  return (
    <Modal open={open}>
      {!ledgerModal && renderMergeScreen()}
      {ledgerModal && (
        <LedgerPrompt
          loading={loading}
          submit={submit}
          handleClose={handleClose}
          typedData={typedData}
        />
      )}
    </Modal>
  );
}

export default React.memo(MergeModal);
