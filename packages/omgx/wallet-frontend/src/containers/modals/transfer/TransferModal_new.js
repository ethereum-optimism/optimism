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

import * as styles from './TransferModal.module.scss';
import Input from 'components/input/Input';
import { selectLookupPrice } from 'selectors/lookupSelector';
import { Box, Grid, TextField, Typography } from '@material-ui/core';
import NetworkSwitcherIcon from 'components/icons/NetworkSwitcherIcon';
import * as S from './TransferModal.style';
import BoxConfirmation from './boxConfirmation/BoxConfirmation';
import { styled } from '@material-ui/core/styles';
import { useTheme } from '@emotion/react';
import AdressDisabled from 'components/adressDisabled/AdressDisabled';
import truncate from 'truncate-middle';
import SwapIcon from 'components/icons/SwapIcon';

function TransferModal ({ open, token }) {
  const dispatch = useDispatch()

  const [ value, setValue ] = useState('')
  const [ recipient, setRecipient ] = useState('')
  const [ showFeedback, setShowFeedback] = useState(false);
  const [ activeButton, setActiveButton ] = useState("slow");

  const loading = useSelector(selectLoading([ 'TRANSFER/CREATE' ]));
  const wAddress = networkService.account ? truncate(networkService.account, 6, 14, '.') : '';

  const lookupPrice = useSelector(selectLookupPrice);
  const theme = useTheme();

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

  function renderTransferScreen () {
    if(typeof(token) === 'undefined') return

    return (
      <>
        <S.StyleCreateTransactions>
          <Grid container>
            <Grid item xs={5}>
              <Box sx={{display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "30px"}}>
                <Typography variant="h2" component="h2">From ETH Mainnet</Typography>
                <NetworkSwitcherIcon active />
              </Box>
              <Typography variant="body2" component="div" sx={{opacity: 0.5}}>Select Token</Typography>

              <Grid container>
                <Grid item xs={4}>
                  <Box sx={{ background: "#121e30", borderRight: "1px solid #2F2E40", height: "100%", display: "flex", justifyContent: "center", alignItems: "center"}}>
                    {token.symbol}
                  </Box>
                </Grid>

                <Grid item xs={8}>
                  <AdressDisabled>
                    <Typography variant="body1" component="p">
                      {wAddress}
                    </Typography>
                  </AdressDisabled>
                </Grid>
              </Grid>
            </Grid>

            <Grid item xs={2}>
              <S.SwapCircle>
                <SwapIcon />
              </S.SwapCircle>
              <S.Line />
            </Grid>

            <Grid item xs={5}>
              <Box sx={{display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "30px"}}>
                <Typography variant="h2" component="h2">To OMGX Mainnet</Typography>
                <NetworkSwitcherIcon />
              </Box>
              <Input
                label='Enter Adress'
                placeholder='Enter adress to send to...'
                value={recipient}
                onChange={i => setRecipient(i.target.value)}
                fullWidth
                // size="small"
              />
            </Grid>
          </Grid>

          <S.Balance>
            <Box display="flex">
              <Box sx={{ flexGrow: 1, flexBasis: 1 }}>
                <S.ContentBalance>
                  <Box>
                    <Input
                      label="Enter Amount"
                      placeholder="0.00"
                      value={value}
                      type="number"
                      onChange={(i) => {setValue(i.target.value)}}
                      unit={token.symbol}
                      maxValue={logAmount(token.balance, token.decimals)}
                      size="small"
                      variant="standard"
                      newStyle
                    />
                    {Object.keys(lookupPrice) && !!value && !!amountToUsd(value, lookupPrice, token) && (
                      <Typography variant="body2" component="p" sx={{opacity: 0.5, position: 'absolute'}}>
                        {`($${amountToUsd(value, lookupPrice, token).toFixed(2)})`}
                      </Typography>
                    )}
                  </Box>
                </S.ContentBalance>
              </Box>

              <Box sx={{ flexGrow: 0, flexShrink: 0 }}>
                <S.TransactionsButton>
                  <S.FastButton active={activeButton} onClick={() => setActiveButton("fast")}>Fast</S.FastButton>
                  <S.BridgeButton onClick={() => setShowFeedback(true)} >
                    <Typography variant="body2" component="span">Bridge</Typography>
                  </S.BridgeButton>
                  <S.SlowButton active={activeButton} onClick={() => setActiveButton("slow")}>Slow</S.SlowButton>
                </S.TransactionsButton>
              </Box>

              <Box sx={{ flexGrow: 1, flexBasis: 1 }}>
                <S.ContentBalance>
                  <Box>
                    <Typography variant="body2" component="p"sx={{opacity: 0.5}}>Current Balance</Typography>
                    <Typography variant="body2" component="p" sx={{opacity: 0.5}}>0,0224</Typography>
                  </Box>
                  <Box>
                    <Typography variant="h3" component="span" sx={{color: theme.palette.secondary.main}}>+ 0,3142</Typography>
                    <Typography variant="body2" component="p" sx={{opacity: 0.5}}>New Balance: 0,3364</Typography>
                  </Box>
                </S.ContentBalance>
              </Box>
            </Box>
          </S.Balance>

        {/* <Button
          className={styles.button}
          onClick={()=>{submit({useLedgerSign: false})}}
          type='primary'
          loading={loading}
          tooltip='Your transfer is still pending. Please wait for confirmation.'
          disabled={disabledTransfer}
          triggerTime={new Date()}
        >
          TRANSFER
        </Button> */}
        </S.StyleCreateTransactions>
      </>
    );
  }

  return (
    <Modal title="Create bridging transaction" open={open} transparent onClose={handleClose}>
      {renderTransferScreen()}
      <BoxConfirmation
        recipient={recipient}
        value={value}
        showFeedback={showFeedback}
        setShowFeedback={setShowFeedback}
        handleClose={handleClose}
        onSubmit={() => {submit({useLedgerSign: false})}} />
    </Modal>
  );
}

export default React.memo(TransferModal);
