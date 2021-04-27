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
import { isEqual, orderBy } from 'lodash';
import BN from 'bignumber.js';
import { Check } from '@material-ui/icons';

import { selectChildchainBalance } from 'selectors/balanceSelector';
import { selectLoading } from 'selectors/loadingSelector';
import { selectFees } from 'selectors/feeSelector';
import { selectLedger } from 'selectors/uiSelector';
import { transfer, getTransferTypedData } from 'actions/networkAction';
import { getToken } from 'actions/tokenAction';
import { closeModal, openAlert, setActiveHistoryTab } from 'actions/uiAction';

import LedgerPrompt from 'containers/modals/ledger/LedgerPrompt';

import Button from 'components/button/Button';
import Modal from 'components/modal/Modal';
import Input from 'components/input/Input';
import InputSelect from 'components/inputselect/InputSelect';
import Select from 'components/select/Select';

import networkService from 'services/networkService';
import { logAmount, powAmount } from 'util/amountConvert';

import * as styles from './TransferModal.module.scss';

function SwapModal ({ open }) {
  const dispatch = useDispatch();

  const [ currency, setCurrency ] = useState('');
  const [ value, setValue ] = useState('');
  const [ feeToken, setFeeToken ] = useState('');
  const [ recipient, setRecipient ] = useState('');
  const [ metadata, setMetadata ] = useState('');
  const [ usableFees, setUsableFees ] = useState([]);
  const [ ledgerModal, setLedgerModal ] = useState(false);
  const [ typedData, setTypedData ] = useState({});

  const [ utxoPicker, setUtxoPicker ] = useState(false);
  const [ utxos, setUtxos ] = useState([]);
  const [ selectedUtxos, setSelectedUtxos ] = useState([]);
  const [ selectedFeeUtxos, setSelectedFeeUtxos ] = useState([]);

  const balances = useSelector(selectChildchainBalance, isEqual);
  const fees = useSelector(selectFees, isEqual);
  const ledgerConnect = useSelector(selectLedger);

  const feesLoading = useSelector(selectLoading([ 'FEE/GET' ]));
  const loading = useSelector(selectLoading([ 'TRANSFER/CREATE' ]));

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
    if (Object.keys(fees).length) {
      const usableFees = balances.filter(balance => {
        const feeObject = fees[balance.currency];
        if (feeObject) {
          if (new BN(balance.amount).gte(new BN(feeObject.amount))) {
            return true;
          }
        }
        return false;
      }).map(i => {
        const feeObject = fees[i.currency];
        const feeAmount = new BN(feeObject.amount).div(new BN(feeObject.subunit_to_unit));
        return {
          title: i.symbol,
          value: i.currency,
          subTitle: `Fee Amount: ${feeAmount.toFixed()}`
        };
      });
      setUsableFees(usableFees);
    }
  }, [ balances, fees, open ]);

  useEffect(() => {
    if (balances.length && !currency) {
      setCurrency(balances[0].currency);
    }
  }, [ balances, currency, open ]);

  useEffect(() => {
    if (usableFees.length && !feeToken) {
      setFeeToken(usableFees[0].value);
    }
  }, [ usableFees, feeToken ]);

  const selectOptions = balances.map(i => ({
    title: i.symbol,
    value: i.currency,
    subTitle: `Balance: ${logAmount(i.amount, i.decimals)}`
  }));

  async function submit ({ useLedgerSign }) {
    if (
      value > 0 &&
      currency &&
      feeToken &&
      recipient
    ) {
      try {
        const valueTokenInfo = await getToken(currency);
        const { txBody, typedData } = await dispatch(getTransferTypedData({
          utxos: [ ...selectedUtxos, ...selectedFeeUtxos ],
          recipient,
          value: powAmount(value, valueTokenInfo.decimals),
          currency,
          feeToken,
          metadata
        }));
        setTypedData(typedData);
        const res = await dispatch(transfer({
          useLedgerSign,
          txBody,
          typedData
        }));
        if (res) {
          dispatch(setActiveHistoryTab('Transactions'));
          dispatch(openAlert('Transfer submitted. You will be blocked from making further transactions until the transfer is confirmed.'));
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
    setFeeToken('');
    setRecipient('');
    setSelectedUtxos([]);
    setUtxos([]);
    setUtxoPicker(false);
    setMetadata('');
    setLedgerModal(false);
    dispatch(closeModal('transferModal'));
  }

  const disabledTransfer = value <= 0 ||
    !currency ||
    !feeToken ||
    !recipient ||
    new BN(value).gt(new BN(getMaxTransferValue()));

  function getMaxTransferValue () {
    const transferingBalanceObject = balances.find(i => i.currency === currency);
    if (!transferingBalanceObject) {
      return;
    }
    if (currency === feeToken) {
      const availableAmount = new BN(transferingBalanceObject.amount).minus(new BN(fees[feeToken].amount));
      return logAmount(availableAmount, transferingBalanceObject.decimals);
    }
    return logAmount(transferingBalanceObject.amount, transferingBalanceObject.decimals);
  }

  function handleUtxoClick (utxo) {
    const isSelected = selectedUtxos.some(i => i.utxo_pos === utxo.utxo_pos);
    if (isSelected) {
      return setSelectedUtxos(selectedUtxos.filter(i => i.utxo_pos !== utxo.utxo_pos));
    }
    if ((selectedUtxos.length + selectedFeeUtxos.length) < 4) {
      return setSelectedUtxos([ ...selectedUtxos, utxo ]);
    }
  }

  function handleFeeUtxoClick (utxo) {
    const isSelected = selectedFeeUtxos.some(i => i.utxo_pos === utxo.utxo_pos);
    if (isSelected) {
      return setSelectedFeeUtxos(selectedFeeUtxos.filter(i => i.utxo_pos !== utxo.utxo_pos));
    }
    if ((selectedUtxos.length + selectedFeeUtxos.length) < 4) {
      return setSelectedFeeUtxos([ ...selectedFeeUtxos, utxo ]);
    }
  }

  function handleUtxoPickerBack () {
    setSelectedUtxos([]);
    setSelectedFeeUtxos([]);
    setUtxoPicker(false);
  }

  function renderUtxoPicker () {
    const currencyUtxos = utxos
      .filter(i => i.currency === currency)
      .filter(i => !!i);

    const feeUtxos = utxos
      .filter(i => i.currency === feeToken)
      .filter(i => !!i);

    const selectedCurrencyAmount = selectedUtxos.reduce((acc, cur) => {
      return acc.plus(new BN(cur.amount.toString()));
    }, new BN(0));

    const selectedFeeAmount = selectedFeeUtxos.reduce((acc, cur) => {
      return acc.plus(new BN(cur.amount.toString()));
    }, new BN(0));

    const currencyObject = balances.find(i => i.currency === currency);
    const currencyCoverAmount = new BN(powAmount(value.toString(), currencyObject.decimals));

    const feeObject = fees[feeToken];
    const feeCoverAmount = new BN(feeObject.amount.toString());

    const sameCurrency = feeToken === currency;
    const utxoPickerDisabled = sameCurrency
      ? currencyCoverAmount.plus(feeCoverAmount).gt(selectedCurrencyAmount)
      : currencyCoverAmount.gt(selectedCurrencyAmount) || feeCoverAmount.gt(selectedFeeAmount);

    function renderCurrencyPick () {
      const enough = sameCurrency
        ? currencyCoverAmount.plus(feeCoverAmount).lte(selectedCurrencyAmount)
        : currencyCoverAmount.lte(selectedCurrencyAmount);

      return (
        <>
          <div className={styles.description}>
            Transfer amount to cover: {sameCurrency
              ? logAmount(currencyCoverAmount.plus(feeCoverAmount), currencyObject.decimals)
              : logAmount(currencyCoverAmount, currencyObject.decimals)}
          </div>

          <div className={[ styles.list, !sameCurrency ? styles.doubleList : '' ].join(' ')}>
            {!currencyUtxos.length && (
              <div className={styles.disclaimer}>You do not have any UTXOs for this token on the OMG Network.</div>
            )}
            {currencyUtxos.map((i, index) => {
              const selected = selectedUtxos.some(selected => selected.utxo_pos === i.utxo_pos);
              return (
                <div
                  key={index}
                  onClick={() => {
                    if (!enough || selected) {
                      handleUtxoClick(i);
                    }
                  }}
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
        </>
      );
    }

    function renderFeePick () {
      const logFeeAmount = new BN(feeObject.amount.toString()).div(new BN(feeObject.subunit_to_unit.toString()));
      const enough = selectedFeeAmount.gte(feeCoverAmount);
      return (
        <>
          <div className={styles.description}>
            Fee amount to cover: {logFeeAmount.toFixed()}
          </div>

          <div className={[ styles.list, !sameCurrency ? styles.doubleList : '' ].join(' ')}>
            {!feeUtxos.length && (
              <div className={styles.disclaimer}>You do not have any fee UTXOs on the OMG Network.</div>
            )}
            {feeUtxos.map((i, index) => {
              const selected = selectedFeeUtxos.some(selected => selected.utxo_pos === i.utxo_pos);
              return (
                <div
                  key={index}
                  onClick={() => {
                    if (!enough || selected) {
                      handleFeeUtxoClick(i);
                    }
                  }}
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
        </>
      );
    }

    return (
      <>
        <h2>Select UTXOs</h2>

        <div className={styles.description}>
          By default, Varna will automatically pick UTXOs to cover 
          your swap amount. However, if you are a more advanced 
          user, you can pick the UTXOs you would like to use in 
          this transaction manually.
        </div>

        {renderCurrencyPick()}

        {!sameCurrency && renderFeePick()}

        <div className={styles.disclaimer}>You can select a maximum of 4 UTXOs.</div>

        <div className={styles.buttons}>
          <Button
            onClick={handleUtxoPickerBack}
            type='outline'
            className={styles.button}
          >
            USE DEFAULT
          </Button>

          <Button
            className={styles.button}
            onClick={() => setUtxoPicker(false)}
            type='primary'
            disabled={utxoPickerDisabled}
          >
            SELECT UTXOS
          </Button>
        </div>
      </>
    );
  }

  function renderTransferScreen () {
    return (
      <>
        <h2>Swap</h2>

        <div className={styles.address}>
          {`From address : ${networkService.account}`}
        </div>

        <Input
          label='To Address'
          placeholder='Hash'
          paste
          value={recipient}
          onChange={i => setRecipient(i.target.value)}
        />

        <InputSelect
          label='Amount to send'
          placeholder={0}
          value={value}
          onChange={i => {
            setValue(i.target.value);
            setSelectedUtxos([]);
            setSelectedFeeUtxos([]);
          }}
          selectOptions={selectOptions}
          onSelect={i => {
            setCurrency(i.target.value);
            setSelectedUtxos([]);
            setSelectedFeeUtxos([]);
          }}
          selectValue={currency}
          maxValue={getMaxTransferValue()}
        />

        {value > 0 && (
          <div
            className={styles.utxoPickLink}
            onClick={() => setUtxoPicker(true)}
          >
            {selectedUtxos.length ? 'Change Selected UTXOs' : 'Advanced UTXO Select'}
          </div>
        )}

        <Select
          loading={feesLoading}
          label='Fee'
          value={feeToken}
          options={usableFees}
          onSelect={i => {
            setFeeToken(i.target.value);
            setSelectedUtxos([]);
            setSelectedFeeUtxos([]);
          }}
          error="No balance to pay fees"
        />

        <Input
          label='Message'
          placeholder='-'
          value={metadata}
          onChange={i => setMetadata(i.target.value || '')}
        />

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
            onClick={() => {
              ledgerConnect
                ? setLedgerModal(true)
                : submit({ useLedgerSign: false });
            }}
            type='primary'
            loading={loading}
            tooltip='Your transfer transaction is still pending. Please wait for confirmation.'
            disabled={disabledTransfer}
          >
            {ledgerConnect ? 'TRANSFER WITH LEDGER' : 'TRANSFER'}
          </Button>
        </div>
      </>
    );
  }

  return (
    <Modal open={open}>
      {!ledgerModal && !utxoPicker && renderTransferScreen()}
      {!ledgerModal && utxoPicker && renderUtxoPicker()}
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

export default React.memo(SwapModal);
