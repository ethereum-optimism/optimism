
import React, { useState, useEffect } from 'react'
import { useDispatch, useSelector } from 'react-redux'

import { depositETHL2, depositErc20 } from 'actions/networkAction'
import { openAlert, openError, setActiveHistoryTab1 } from 'actions/uiAction'

import Button from 'components/button/Button'
import Input from 'components/input/Input'
import GasPicker from 'components/gaspicker/GasPicker'

import { selectLoading } from 'selectors/loadingSelector'
import { selectSignatureStatus_depositTRAD } from 'selectors/signatureSelector'
import { amountToUsd, logAmount, powAmount } from 'util/amountConvert'

import * as S from './InputSteps.styles'
import { selectLookupPrice } from 'selectors/lookupSelector'
import { Typography, useMediaQuery } from '@material-ui/core'
import { useTheme } from '@emotion/react'

function InputStep({ handleClose, token }) {

  const dispatch = useDispatch()
  const [value, setValue] = useState('')
  const [disabledSubmit, setDisabledSubmit] = useState(true)
  const [gasPrice, setGasPrice] = useState()
  const [selectedSpeed, setSelectedSpeed] = useState('normal')
  const depositLoading = useSelector(selectLoading(['DEPOSIT/CREATE']))
  const signatureStatus = useSelector(selectSignatureStatus_depositTRAD)
  const lookupPrice = useSelector(selectLookupPrice)

  async function doDeposit() {

    let res

    if(token.symbol === 'ETH') {
      console.log("Depositing ETH")
      if (value > 0) {
        res = await dispatch(depositETHL2(value, gasPrice))
        if (res) {
          dispatch(setActiveHistoryTab1('Deposits'))
          dispatch(openAlert('ETH deposit submitted'))
          handleClose()
        }
      }
    } else {
      console.log("Depositing ERC20")
      res = await dispatch(
        depositErc20(powAmount(value, token.decimals), token.address, gasPrice, token.addressL2)
      )
      if (res) {
        dispatch(setActiveHistoryTab1('Deposits'))
        dispatch(openAlert(`${token.symbol} deposit submitted.`))
        handleClose()
      } else {
        dispatch(openError(`Failed to deposit ${token.symbol}`))
      }
    }
  }

  function setAmount(value) {
    if (Number(value) > 0 && Number(value) < Number(token.balance)) {
      setDisabledSubmit(false)
    } else {
      setDisabledSubmit(true)
    }
    setValue(value)
  }

  const renderGasPicker = (
    <GasPicker
      selectedSpeed={selectedSpeed}
      setSelectedSpeed={setSelectedSpeed}
      setGasPrice={setGasPrice}
    />
  )
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));

  useEffect(() => {
    if (signatureStatus && depositLoading) {
      //we are all set - can close the window
      //transaction has been sent and signed
      handleClose()
    }
  }, [ signatureStatus, depositLoading, handleClose ])

  console.log("Loading:", depositLoading)

  let buttonLabel_1 = 'CANCEL'
  if( depositLoading ) buttonLabel_1 = 'CLOSE WINDOW'

  return (
    <>
      <Typography variant="h2" sx={{fontWeight: 700, mb: 3}}>
        {`Deposit ${token && token.symbol ? token.symbol : ''}`}
      </Typography>

      <Input
        label="Enter amount to deposit"
        placeholder="0.0000"
        value={value}
        type="number"
        onChange={(i)=>setAmount(i.target.value)}
        unit={token.symbol}
        maxValue={logAmount(token.balance, token.decimals)}
        variant="standard"
        newStyle
      />

      {Object.keys(lookupPrice) && !!value && !!amountToUsd(value, lookupPrice, token) && (
        <h3>
          {`Amount in USD ${amountToUsd(value, lookupPrice, token).toFixed(2)}`}
        </h3>
      )}

      {renderGasPicker}

      <S.WrapperActions>
        <Button
          onClick={handleClose}
          color="neutral"
          size="large"
        >
          {buttonLabel_1}
        </Button>
        <Button
          onClick={doDeposit}
          color='primary'
          size="large"
          variant="contained"
          loading={depositLoading}
          tooltip='Your swap is still pending. Please wait for confirmation.'
          disabled={disabledSubmit}
          triggerTime={new Date()}
          fullWidth={isMobile}
        >
          Deposit
        </Button>
      </S.WrapperActions>
    </>
  )
}

export default React.memo(InputStep)
