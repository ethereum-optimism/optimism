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

import { useTheme } from '@emotion/react'

import { Typography, useMediaQuery } from '@material-ui/core'

import { exitBOBA } from 'actions/networkAction'
import { openAlert, openError } from 'actions/uiAction'

import Button from 'components/button/Button'
import Input from 'components/input/Input'

import { selectLoading } from 'selectors/loadingSelector'
import { selectSignatureStatus_exitTRAD } from 'selectors/signatureSelector'
import { selectLookupPrice } from 'selectors/lookupSelector'

import { amountToUsd, logAmount } from 'util/amountConvert'

import * as S from './DoExitSteps.styles'

function DoExitStep({ handleClose, token }) {

  const dispatch = useDispatch()

  const [value, setValue] = useState('')
  const [disabledSubmit, setDisabledSubmit] = useState(true)
  const exitLoading = useSelector(selectLoading(['EXIT/CREATE']))
  const signatureStatus = useSelector(selectSignatureStatus_exitTRAD)
  const lookupPrice = useSelector(selectLookupPrice)

  async function doExit() {

    let res = await dispatch(exitBOBA(token.address, value))

    //person will receive ETH on the L1, not oETH
    let currencyL1 = token.symbol

    if (currencyL1 === 'oETH')
      currencyL1 = 'ETH'

    if (res) {
      dispatch(
        openAlert(
          `${token.symbol} was exited to L1. You will receive
          ${Number(value).toFixed(2)} ${currencyL1}
          on L1 in 7 days.`
        )
      )
      handleClose()
    } else {
      dispatch(openError(`Failed to exit L2`))
    }
  }

  function setExitAmount(value) {
    if (Number(value) > 0 && Number(value) < Number(token.balance)) {
      setDisabledSubmit(false)
    } else {
      setDisabledSubmit(true)
    }
    setValue(value)
  }

  const theme = useTheme()
  const isMobile = useMediaQuery(theme.breakpoints.down('md'))

  let buttonLabel = 'CANCEL'
  if( exitLoading ) buttonLabel = 'CLOSE WINDOW'

  useEffect(() => {
    if (signatureStatus && exitLoading) {
      //we are all set - can close the window
      //transaction has been sent and signed
      handleClose()
    }
  }, [ signatureStatus, exitLoading, handleClose ])

  return (
    <>
      <Typography variant="h2" sx={{fontWeight: 700, mb: 3}}>
        Standard Exit ({`${token ? token.symbol : ''}`})
      </Typography>

      <Input
        label={'Amount to exit'}
        placeholder="0.0"
        value={value}
        type="number"
        onChange={(i)=>{setExitAmount(i.target.value)}}
        unit={token.symbol}
        maxValue={logAmount(token.balance, token.decimals)}
        variant="standard"
        newStyle
      />

      {token && token.symbol === 'oETH' && (
        <Typography variant="body2" sx={{mt: 2}}>
          {value &&
            `You will receive ${Number(value).toFixed(2)} ETH
            ${!!amountToUsd(value, lookupPrice, token) ?  `($${amountToUsd(value, lookupPrice, token).toFixed(2)})`: ''}
            on L1.
            Your funds will be available on L1 in 7 days.`}
        </Typography>
      )}

      {token && token.symbol !== 'oETH' && (
        <Typography variant="body2" sx={{mt: 2}}>
          {value &&
            `You will receive ${Number(value).toFixed(2)} ${token.symbol}
            ${!!amountToUsd(value, lookupPrice, token) ?  `($${amountToUsd(value, lookupPrice, token).toFixed(2)})`: ''}
            on L1.
            Your funds will be available on L1 in 7 days.`}
        </Typography>
      )}

      {exitLoading && (
        <Typography variant="body2" sx={{mt: 2, color: 'green'}}>
          This window will automatically close when your transaction has been signed and submitted.
        </Typography>
      )}

      <S.WrapperActions>
          <Button
            onClick={handleClose}
            color="neutral"
            size="large"
          >
            {buttonLabel}
          </Button>
          {token && (
            <Button
              onClick={doExit}
              color="primary"
              variant="contained"
              loading={exitLoading}
              tooltip="Your exit is still pending. Please wait for confirmation."
              disabled={disabledSubmit}
              triggerTime={new Date()}
              fullWidth={isMobile}
              size="large"
            >
              Exit
            </Button>
          )}
      </S.WrapperActions>
    </>
  )
}

export default React.memo(DoExitStep)
