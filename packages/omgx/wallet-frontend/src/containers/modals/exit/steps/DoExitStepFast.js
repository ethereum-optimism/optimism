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

import React, { useState, useEffect } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { depositL2LP } from 'actions/networkAction'
import { openAlert, openError } from 'actions/uiAction'
import { selectLoading } from 'selectors/loadingSelector'

import Button from 'components/button/Button'

import { amountToUsd, logAmount, powAmount } from 'util/amountConvert'
import networkService from 'services/networkService'

import * as styles from '../ExitModal.module.scss'
import Input from 'components/input/Input'
import { selectLookupPrice } from 'selectors/lookupSelector'

function DoExitStepFast({ handleClose, token }) {

  const dispatch = useDispatch()
    
  const [value, setValue] = useState('')
  const [LPBalance, setLPBalance] = useState(0)
  const [feeRate, setFeeRate] = useState(0)
  const [disabledSubmit, setDisabledSubmit] = useState(true)

  const exitLoading = useSelector(selectLoading(['EXIT/CREATE']))
  const lookupPrice = useSelector(selectLookupPrice);

  function setAmount(value) {
    if (
      Number(value) > 0 &&
      Number(value) < Number(LPBalance) &&
      Number(value) < Number(token.balance)
    ) {
      setDisabledSubmit(false)
    } else {
      setDisabledSubmit(true)
    }
    setValue(value)
  }

  const receivableAmount = (value) => {
    return (Number(value) * ((100 - Number(feeRate)) / 100)).toFixed(2)
  }

  async function doExit() {

    let res = await dispatch(
      depositL2LP(
        token.address,
        powAmount(value, token.decimals) //take a value, convert to 18 decimals, generate string
      )
    )

    let currencyL1 = token.symbol

    //person will receive ETH on the L1, not oETH
    if (currencyL1 === 'oETH') {
      currencyL1 = 'ETH'
    }

    if (res) {
      dispatch(openAlert(`${token.symbol} was deposited into the L2 liquidity pool. 
        You will receive ${receivableAmount(value)} ${currencyL1} on L1.`));
      handleClose();
    } else {
      dispatch(openError(`Failed to fast exit funds from L2`));
    }

  }

  useEffect(() => {
    if (typeof(token) !== 'undefined') {
      networkService.L1LPBalance(token.addressL1).then((res) => {
        setLPBalance(Number(res).toFixed(2))
      })
      networkService.getTotalFeeRate().then((feeRate) => {
        setFeeRate(feeRate)
      })
    }
  }, [token])

  const label = 'There is a ' + feeRate + '% fee.'

  return (
    <>
      <h2>Fast Exit</h2>
      
      <Input
        label={label}
        placeholder={`Amount to exit`}
        value={value}
        type="number"
        onChange={(i)=>{setAmount(i.target.value)}}
        unit={token.symbol}
        maxValue={logAmount(token.balance, token.decimals)}
      />

      {token && token.symbol === 'oETH' && (
        <h3>
          {value &&
            `You will receive 
            ${receivableAmount(value)} 
            ETH 
            ${!!amountToUsd(value, lookupPrice, token) ?  `($${amountToUsd(value, lookupPrice, token).toFixed(2)})`: ''}
            on L1.`
          }
        </h3>
      )}

      {token && token.symbol !== 'oETH' && (
        <h3>
          {value &&
            `You will receive 
            ${receivableAmount(value)} 
            ${token.symbol} 
            ${!!amountToUsd(value, lookupPrice, token) ?  `($${amountToUsd(value, lookupPrice, token).toFixed(2)})`: ''}
            on L1.`
          }
        </h3>
      )}

      {Number(LPBalance) < Number(value) && (
        <h3 style={{color: 'red'}}>
          The liquidity pool balance (of {LPBalance}) is too low to cover your swap - please
          use the traditional exit or reduce the amount to swap.
        </h3>
      )}

      <div className={styles.buttons}>
        <Button
          onClick={handleClose}
          className={styles.button}
          type="outline"
          style={{flex: 0}}
        >
          CANCEL
        </Button>        
        <Button
          onClick={doExit}
          type='primary'
          style={{flex: 0, minWidth: 200}}
          loading={exitLoading}
          className={styles.button}
          tooltip='Your exit is still pending. Please wait for confirmation.'
          disabled={disabledSubmit}
          triggerTime={new Date()}
        >
          FAST EXIT
        </Button>
      </div>
    </>
  )
}

export default React.memo(DoExitStepFast)
