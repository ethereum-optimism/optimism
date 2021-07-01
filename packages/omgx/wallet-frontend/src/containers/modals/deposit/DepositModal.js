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

import React, { useState, useCallback, useEffect } from 'react';
import { ethers } from 'ethers';
import { useDispatch } from 'react-redux';

import { closeModal } from 'actions/uiAction';
import { getToken } from 'actions/tokenAction';

import Modal from 'components/modal/Modal';

import InputStep from './steps/InputStep';
import ApproveStep from './steps/ApproveStep';

const ETH0x = "0x0000000000000000000000000000000000000000";

function DepositModal ({ open, omgOnly = false, fast = false }) {

  const dispatch = useDispatch();

  const [ step, setStep ] = useState('INPUT_STEP');
  const [ currency, setCurrency ] = useState(ETH0x);
  const [ tokenInfo, setTokenInfo ] = useState({});
  const [ value, setValue ] = useState('');

  useEffect(() => {
    async function getTokenInfo () {
      const _currency = currency.toLowerCase();
      if (_currency && ethers.utils.isAddress(_currency)) {
        const tokenInfo = await getToken(_currency);
        setTokenInfo(tokenInfo);
      } else {
        setTokenInfo({});
      }
    }
    getTokenInfo();
  }, [ currency ]);

  const handleClose = useCallback(() => {
    setCurrency(ETH0x);
    setValue('');
    setStep('INPUT_STEP');
    dispatch(closeModal('depositModal'));
  }, [ dispatch ]);

  return (
    <Modal open={open}>
      {step === 'INPUT_STEP' && (
        <InputStep
          onClose={handleClose}
          onNext={()=>setStep('APPROVE_STEP')}
          currency={currency}
          tokenInfo={tokenInfo}
          value={value}
          setCurrency={setCurrency}
          setTokenInfo={setTokenInfo}
          setValue={setValue}
          omgOnly={omgOnly}
          fast={fast}
        />
      )}
      {step === 'APPROVE_STEP' && (
        <ApproveStep
          onClose={handleClose}
          currency={currency}
          value={value}
          tokenInfo={tokenInfo}
          fast={fast}
        />
      )}
    </Modal>
  );
}

export default React.memo(DepositModal);
