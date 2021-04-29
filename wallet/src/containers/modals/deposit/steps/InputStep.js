import React, { useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';

import Button from 'components/button/Button';
import Input from 'components/input/Input';
import Tabs from 'components/tabs/Tabs';

import { openAlert, openError, setActiveHistoryTab } from 'actions/uiAction';
import networkService from 'services/networkService';
import { selectLoading } from 'selectors/loadingSelector';
import { depositETHL2, depositL1LP } from 'actions/networkAction';

import * as styles from '../DepositModal.module.scss';

const ETH0x = networkService.OmgUtil.transaction.ETH_CURRENCY;

function InputStep ({
  onClose,
  onNext,
  currency,
  tokenInfo,
  value,
  setCurrency,
  setTokenInfo,
  setValue,
  fast
}) {

  const dispatch = useDispatch(); 

  let uSC = 'ETH';

  const [ activeTab, setActiveTab ] = useState(uSC);
  const [ LPBalance, setLPBalance ] = useState(0);
  const depositLoading = useSelector(selectLoading([ 'DEPOSIT/CREATE' ]));

  function handleClose () {
    setActiveTab('ETH');
    onClose();
  }

  async function depositETH () {
    if (value > 0 && tokenInfo) {
      let res
      if (fast) {
        res = await dispatch(depositETHL2(value));
      } else {
        res = await dispatch(depositL1LP(currency, value))
      }
      if (res) {
        dispatch(setActiveHistoryTab('Deposits'));
        if (fast) {
          dispatch(openAlert(`ETH was deposited the the L1LP. You will receive ${(Number(value) * 0.97).toFixed(2)} WETH on L2`));
        } else {
          dispatch(openAlert('ETH deposit submitted.'));
        }
        handleClose();
      } else {
        dispatch(openError('Failed to deposit ETH'));
      }
    }
  }

  const disabledSubmit = value <= 0 || !currency || !networkService.l1Web3Provider.utils.isAddress(currency) || (fast && Number(value) > Number(LPBalance));

  if (fast && Object.keys(tokenInfo).length && currency) {
    networkService.L2LPBalance(currency).then((LPBalance)=>{
      setLPBalance(LPBalance)
    })
  }

  return (
    <>

      {fast &&
        <h2>Fast swap onto OMGX</h2>
      }

      {!fast &&
        <h2>Traditional Deposit</h2>
      }
      
      <Tabs
        className={styles.tabs}
        onClick={i => {
          i === 'ETH' ? setCurrency(ETH0x) : setCurrency('');
          setActiveTab(i);
        }}
        activeTab={activeTab}
        tabs={[ 'ETH', 'ERC20' ]}
      />

      {activeTab === 'ERC20' && (
        <Input
          label='ERC20 Token Smart Contract Address.'
          placeholder='0x'
          paste
          value={currency}
          onChange={i=>setCurrency(i.target.value.toLowerCase())} //because this is a user input!!
        />
      )}

      <Input
        label='Amount to deposit into OMGX'
        type='number'
        unit={tokenInfo ? tokenInfo.symbol : ''}
        placeholder={0}
        value={value}
        onChange={i=>setValue(i.target.value)} 
      />

      {fast && Object.keys(tokenInfo).length && currency ? (
        <h3>
          The L1 liquidity pool currently has {LPBalance} {tokenInfo.symbol}.
        </h3>
      ):<></>}

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
            tooltip='Your deposit is still pending. Please wait for confirmation.'
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
