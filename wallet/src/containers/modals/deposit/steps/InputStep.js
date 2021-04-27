import React, { useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';

import Button from 'components/button/Button';
import Input from 'components/input/Input';
import Tabs from 'components/tabs/Tabs';

import { openAlert, openError, setActiveHistoryTab } from 'actions/uiAction';
import networkService from 'services/networkService';
import { selectLoading } from 'selectors/loadingSelector';

import { depositETHL2 } from 'actions/networkAction';

import * as styles from '../DepositModal.module.scss';

const ETH0x = networkService.OmgUtil.transaction.ETH_CURRENCY;
//OMG smart contract address
const OMG0x = '0xd26114cd6EE289AccF82350c8d8487fedB8A0C07';

function InputStep ({
  onClose,
  onNext,
  currency,
  tokenInfo,
  value,
  setCurrency,
  setTokenInfo,
  setValue,
  omgOnly
}) {

  const dispatch = useDispatch();

  let uSC = 'ETH';
  if(omgOnly) uSC = 'ERC20'

  let [ activeTab, setActiveTab ] = useState(uSC);

  const depositLoading = useSelector(selectLoading([ 'DEPOSIT/CREATE' ]));

  function handleClose () {
    setActiveTab('ETH');
    onClose();
  }

  async function depositETH () {
    if (value > 0 && tokenInfo) {
      const res = await dispatch(depositETHL2(value));
      if (res) {
        dispatch(setActiveHistoryTab('Deposits'));
        dispatch(openAlert('ETH deposit submitted.'));
        handleClose();
      } else {
        dispatch(openError('Failed to deposit ETH'));
      }
    }
  }

  const disabledSubmit = value <= 0 || !currency || !networkService.l1Web3Provider.utils.isAddress(currency);

  if(omgOnly) {
    setCurrency(OMG0x)
  }

  return (
    <>
      <h2>Deposit</h2>
      
      {!omgOnly &&
        <Tabs
          className={styles.tabs}
          onClick={i => {
            i === 'ETH' ? setCurrency(ETH0x) : setCurrency('');
            setActiveTab(i);
          }}
          activeTab={activeTab}
          tabs={[ 'ETH', 'ERC20' ]}
        />
      }

      {!omgOnly && activeTab === 'ERC20' && (
        <Input
          label='ERC20 Token Smart Contract Address. For example, if you are depositing OMG, this would be the address starting in 0xd26114...'
          placeholder='0x'
          paste
          value={currency}
          onChange={i=>setCurrency(i.target.value.toLowerCase())} //because this is a user input!!
        />
      )}

      {omgOnly && (
        <Input
          label='For extra safety, please compare this address with the OMG smart contract address you obtained from another trusted source.'
          placeholder={OMG0x}
          value={OMG0x}
          onChange={i=>setCurrency(OMG0x)}
        />
      )}

      <Input
        label='Amount to deposit into the Optimism'
        type='number'
        unit={tokenInfo ? tokenInfo.symbol : ''}
        placeholder={0}
        value={value}
        onChange={i=>setValue(i.target.value)} 
      />

      <div className={styles.buttons}>
        <Button
          onClick={handleClose}
          type='outline'
          style={{ flex: 0 }}
        >
          CANCEL
        </Button>
        {activeTab === 'ETH' && (
          <Button
            onClick={depositETH}
            type='primary'
            style={{ flex: 0 }}
            loading={depositLoading}
            tooltip='Your deposit transaction is still pending. Please wait for confirmation.'
            disabled={disabledSubmit}
          >
            DEPOSIT
          </Button>
        )}
        {activeTab === 'ERC20' && (
          <Button
            onClick={onNext}
            type='primary'
            style={{ flex: 0 }}
            disabled={disabledSubmit}
          >
            NEXT
          </Button>
        )}
      </div>
    </>
  );
}

export default React.memo(InputStep);
