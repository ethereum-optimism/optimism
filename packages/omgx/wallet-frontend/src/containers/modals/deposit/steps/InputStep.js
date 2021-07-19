
import React, { useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'

import { depositETHL2, depositErc20 } from 'actions/networkAction'
import { openAlert, openError, setActiveHistoryTab1 } from 'actions/uiAction'

import Button from 'components/button/Button'
import Input from 'components/input/Input'
import GasPicker from 'components/gaspicker/GasPicker'

import { selectLoading } from 'selectors/loadingSelector'
import { logAmount, powAmount } from 'util/amountConvert'

import * as styles from '../DepositModal.module.scss'

function InputStep({ handleClose, token }) {

  const dispatch = useDispatch()
  const [value, setValue] = useState('')
  const [disabledSubmit, setDisabledSubmit] = useState(true)
  const [gasPrice, setGasPrice] = useState()
  const [selectedSpeed, setSelectedSpeed] = useState('normal')
  const depositLoading = useSelector(selectLoading(['DEPOSIT/CREATE']))

  async function doDeposit() {

    let res

    if(token.symbol === 'ETH') {
      console.log("Depositing ETH")
      if (value > 0) {
        res = await dispatch(depositETHL2(value))
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

  console.log("Loading:", depositLoading)

  return (
    <>
      <h2>
        {`Deposit ${token && token.symbol ? token.symbol : ''}`}
      </h2>

      <Input
        placeholder={'Amount to deposit'}
        value={value}
        type="number"
        onChange={(i)=>setAmount(i.target.value)}
        unit={token.symbol}
        maxValue={logAmount(token.balance, token.decimals)}
      />

      {renderGasPicker}

      <div className={styles.buttons}>
        <Button 
          onClick={handleClose} 
          type="outline" 
          style={{ flex: 0 }}
        >
          CANCEL
        </Button>
        <Button
          onClick={doDeposit}
          type="primary"
          style={{flex: 0, minWidth: 200}}
          loading={depositLoading}
          tooltip="Your swap is still pending. Please wait for confirmation."
          disabled={disabledSubmit}
        >
          DEPOSIT
        </Button>
      </div>
    </>
  )
}

export default React.memo(InputStep)
