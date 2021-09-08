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
import { useDispatch } from 'react-redux';
import { getToken } from 'actions/tokenAction';
import { closeModal } from 'actions/uiAction';

import Button from 'components/button/Button';
import Modal from 'components/modal/Modal';
import Input from 'components/input/Input';

import * as styles from './AddTokenModal.module.scss';
import * as S from './AddTokenModal.styles';
import { Typography } from '@material-ui/core';

function AddTokenModal ({ open }) {

  const dispatch = useDispatch();

  const [ tokenContractAddress, setTokenContractAddress ] = useState('');
  const [ tokenInfo, setTokenInfo ] = useState('');
  const [ buttonDisplay, setButtonDisplay ] = useState('Waiting');

  function handleUpdateTokenContractAddress (i) {
    setTokenContractAddress(i);
    setButtonDisplay('Waiting');
  }

  function handleClose (v) {
    dispatch(closeModal('addNewTokenModal'));
  }

  async function handleLookup () {
    const tokenInfo = await getToken(tokenContractAddress);
    if (tokenInfo.error) {
      setTokenInfo(tokenInfo);
      setButtonDisplay('Waiting');
    } else {
      setTokenInfo(tokenInfo);
      setButtonDisplay('Success');
    }
  }

  function renderAddTokenScreen () {

    return (
      <div className={`${tokenInfo.error ? styles.alert : styles.normal}`}>

        <Typography variant="h2" sx={{fontWeight: 700, mb: 2}}>
          Add Token
        </Typography>

        <Typography variant="body1" sx={{mb: 1}}>
          Token Contract Address
        </Typography>

        <Input
          placeholder='Hash (0x...)'
          paste
          value={tokenContractAddress}
          onChange={i=>{handleUpdateTokenContractAddress(i.target.value)}}
          sx={{fontSize: '50px', boxShadow: '-13px 15px 19px rgba(0, 0, 0, 0.15), inset 53px 36px 120px rgba(255, 255, 255, 0.06)', backgroundColor: 'rgba(9, 22, 43, 0.5)'}}
        />

        <Typography variant="body1" component="p" sx={{mt: 3, opacity: 0.7}}>
          Token symbol: {tokenInfo.symbol}<br/>
          Token name: {tokenInfo.name}<br/>
          Token decimals: {tokenInfo.decimals}
        </Typography>

        {tokenInfo.error &&
          <Typography variant="body2" component="p" sx={{background: 'white', color: 'black', marginTop: '10px'}}>
            WARNING: {tokenContractAddress}<br/>could not be found on Ethereum.
            Please check for typos.
          </Typography>
        }

        {buttonDisplay === "Waiting" &&
          <S.WrapperActions>
            <Button
              onClick={handleClose}
              color="neutral"
              size="large"
            >
              CANCEL
            </Button>
            <Button
              onClick={handleLookup}
              color='primary'
              variant="contained"
              disabled={!tokenContractAddress}
              size="large"
            >
              LOOKUP
            </Button>
          </S.WrapperActions>
        }

        {buttonDisplay === "Success" &&
          <S.WrapperActions>
            <Button
              onClick={handleClose}
              color="neutral"
              size="large"
            >
              CANCEL
            </Button>
            <Button
              onClick={handleClose}
              color='primary'
              variant="contained"
              size="large"
            >
              OK
            </Button>
          </S.WrapperActions>
        }
      </div>

    );
  }

  return (
    <Modal open={open} onClose={handleClose} maxWidth="md">
      {renderAddTokenScreen()}
    </Modal>
  );
}

export default React.memo(AddTokenModal);
