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

import React from 'react'
import { useDispatch } from 'react-redux'

import Modal from 'components/modal/Modal'
import { closeModal } from 'actions/uiAction'

import InputStep from './steps/InputStep'
import InputStepFast from './steps/InputStepFast'

function DepositModal({ open, token, fast }) {

  const dispatch = useDispatch()

  function handleClose() {
    dispatch(closeModal('depositModal'))
  }

  return (
    <Modal open={open} maxWidth="md" onClose={handleClose}>
      {!!fast ? (
          <InputStepFast handleClose={handleClose} token={token}/>
        ) : (
          <InputStep handleClose={handleClose} token={token}/>
      )}
    </Modal>
  )
}

export default React.memo(DepositModal)
