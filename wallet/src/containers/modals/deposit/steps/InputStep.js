import React, { useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';

import Button from 'components/button/Button';
import Input from 'components/input/Input';
import Tabs from 'components/tabs/Tabs';

import { openAlert, openError, setActiveHistoryTab1 } from 'actions/uiAction';
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

  const [ activeTab1, setActiveTab1 ] = useState(uSC);
  const [ LPBalance, setLPBalance ] = useState(0);
  const [ feeRate, setFeeRate ] = useState(0);
  const depositLoading = useSelector(selectLoading([ 'DEPOSIT/CREATE' ]));

  function handleClose () {
    setActiveTab1('ETH');
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
        dispatch(setActiveHistoryTab1('Deposits'));
        if (fast) {
          dispatch(openAlert(`ETH was deposited the the L1LP. You will receive ${(Number(value) * 0.97).toFixed(2)} oETH on L2`));
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
    networkService.getL1LPFeeRatio().then((feeRate)=>{
      setFeeRate(feeRate)
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
          setActiveTab1(i);
        }}
        activeTab={activeTab1}
        tabs={[ 'ETH', 'ERC20' ]}
      />

      {activeTab1 === 'ERC20' && (
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

      {fast && activeTab1 === 'ETH' && Object.keys(tokenInfo).length && currency ? (
        <>
          <h3>
            The L2 liquidity pool has {LPBalance} oETH. The liquidity fee is {feeRate}%.{" "} 
            {value && `You will receive ${(Number(value) * 0.97).toFixed(2)} oETH on L2.`}
          </h3>
        </>
      ):<></>}

      {fast && activeTab1 === 'ERC20' && Object.keys(tokenInfo).length && currency ? (
        <>
          <h3>
            The L2 liquidity pool contains {LPBalance} {tokenInfo.symbol}. The liquidity fee is {feeRate}%.{" "} 
            {value && `You will receive ${(Number(value) * 0.97).toFixed(2)} ${tokenInfo.symbol} on L2.`}
          </h3>
        </>
      ):<></>}

      <div className={styles.buttons}>
        <Button
          onClick={handleClose}
          type='outline'
          style={{ flex: 0 }}
        >
          CANCEL
        </Button>
        {activeTab1 === 'ETH' && (
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
        {activeTab1 === 'ERC20' && (
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
