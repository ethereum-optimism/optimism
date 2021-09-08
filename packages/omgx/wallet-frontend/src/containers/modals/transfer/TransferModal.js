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

import { transfer } from 'actions/networkAction';

import { closeModal, openAlert } from 'actions/uiAction';
import { selectLoading } from 'selectors/loadingSelector';

import Button from 'components/button/Button';
import Modal from 'components/modal/Modal';

import { amountToUsd, logAmount } from 'util/amountConvert'
import networkService from 'services/networkService';

import Input from 'components/input/Input';
import { selectLookupPrice } from 'selectors/lookupSelector';
import { Box, Typography, useMediaQuery } from '@material-ui/core';
import * as S from './TransferModal.style';
import { useTheme } from '@emotion/react';
import truncate from 'truncate-middle';

function TransferModal ({ open, token }) {
  const dispatch = useDispatch()

  const [ value, setValue ] = useState('')
  const [ recipient, setRecipient ] = useState('')

  const loading = useSelector(selectLoading([ 'TRANSFER/CREATE' ]));
  const wAddress = networkService.account ? truncate(networkService.account, 6, 14, '.') : '';

  const lookupPrice = useSelector(selectLookupPrice);

  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));

  async function submit () {
    if (
      value > 0 &&
      token.address &&
      recipient
    ) {
      try {
        const transferResponse = await dispatch(transfer(recipient, value, token.address));
        if (transferResponse) {
          dispatch(openAlert('Transaction submitted'));
          handleClose();
        }
      } catch (err) {
        //guess not really?
      }
    }
  }

  function handleClose () {
    setValue('')
    setRecipient('')
    dispatch(closeModal('transferModal'))
  }

  const disabledTransfer = value <= 0 ||
    !token.address ||
    !recipient

  if(typeof(token) === 'undefined') return

  return (
    <Modal open={open} onClose={handleClose} maxWidth="md">
      <Typography variant="h2" sx={{fontWeight: 700, mb: 2}}>
        Transfer
      </Typography>

      <Typography variant="body1" sx={{mb: 1}}>
        From Adress: {wAddress}
      </Typography>

      <Typography variant="body1" sx={{mb: 1}}>
        To Adress
      </Typography>

      <Box sx={{display: 'flex', flexDirection: 'column', gap: '20px'}}>
        <Input
          placeholder='Enter adress to send to...'
          value={recipient}
          onChange={i => setRecipient(i.target.value)}
          fullWidth
          paste
          sx={{fontSize: '50px'}}
        />

        <Input
          label="Enter Amount to Deposit"
          placeholder="0.00"
          value={value}
          type="number"
          onChange={(i) => {setValue(i.target.value)}}
          unit={token.symbol}
          maxValue={logAmount(token.balance, token.decimals)}
          variant="standard"
          newStyle
        />
      </Box>

      {Object.keys(lookupPrice) && !!value && !!amountToUsd(value, lookupPrice, token) && (
        <Typography variant="body2" component="p" sx={{opacity: 0.5, mt: 3}}>
          {`($${amountToUsd(value, lookupPrice, token).toFixed(2)})`}
        </Typography>
      )}

      <S.WrapperActions>
        {!isMobile ? (
          <Button
            onClick={handleClose}
            color="neutral"
            size="large"
          >
            Cancel
          </Button>
        ) : null}
          <Button
            onClick={() => {submit({useLedgerSign: false})}}
            color='primary'
            variant="contained"
            loading={loading}
            tooltip='Your exit is still pending. Please wait for confirmation.'
            disabled={disabledTransfer}
            triggerTime={new Date()}
            fullWidth={isMobile}
            size="large"
          >
            Transfer
          </Button>
      </S.WrapperActions>
    </Modal>
  );
}

export default React.memo(TransferModal);
