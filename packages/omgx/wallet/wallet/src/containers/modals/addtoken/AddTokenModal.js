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

        <h2>Add Token</h2>

        <Input
          label='Token Contract Address'
          placeholder='Hash (0x...)'
          paste
          value={tokenContractAddress}
          onChange={i=>{handleUpdateTokenContractAddress(i.target.value)}}
        />

        <div className={styles.disclaimer}>
          Token symbol: {tokenInfo.symbol}<br/>
          Token name: {tokenInfo.name}<br/>
          Token decimals: {tokenInfo.decimals}
        </div>

        {tokenInfo.error && 
          <div style={{background: 'white', color:'black', marginTop: '10px'}}>
            WARNING: {tokenContractAddress}<br/>could not be found on Ethereum.
            Please check for typos.
          </div>
        }
        
        {buttonDisplay === "Waiting" &&
          <div className={styles.buttons}>
            <Button
              onClick={handleClose}
              type='secondary'
              className={styles.button}
            >
              CANCEL
            </Button>
            <Button
              onClick={handleLookup}
              type='primary'
              className={styles.button}
              disabled={!tokenContractAddress}
            >
              LOOKUP
            </Button>
          </div>
        }

        {buttonDisplay === "Success" &&
          <div className={styles.buttons}>
            <Button
              onClick={handleClose}
              type='secondary'
              className={styles.button}
            >
              CANCEL
            </Button>
            <Button
              onClick={handleClose}
              type='primary'
              className={styles.button}
            >
              OK
            </Button>
          </div>
        }
      </div>

    );
  }

  return (
    <Modal open={open}>
      {renderAddTokenScreen()}
    </Modal>
  );
}

export default React.memo(AddTokenModal);
