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

import { closeModal, openAlert, openError } from 'actions/uiAction';
import { transferDao } from 'actions/daoAction';

import * as styles  from './daoModal.module.scss'

import Modal from 'components/modal/Modal'
import Button from 'components/button/Button'
import Input from 'components/input/Input'

function TransferDaoModal({ open }) {

    const [recipient, setRecipient] = useState('')
    const [amount, setAmount] = useState('')
    const dispatch = useDispatch()

    function handleClose() {
        setRecipient('')
        setAmount('')
        dispatch(closeModal('transferDaoModal'))
    }

    const submit = async () => {
        let res = await dispatch(transferDao({recipient, amount}));

        if(res) {
            dispatch(openAlert(`Governance token transferred`))
            handleClose()
        } else {
            dispatch(openError(`Failed to transfer governance token`))
            handleClose()
        }
    }

    const disabledTransfer = amount <= 0 || !recipient;

    return (
        <Modal open={open}>
            <h2>Transfer Boba</h2>

            <Input
                label='To Address'
                placeholder='Hash or ENS name'
                paste
                value={recipient}
                onChange={i => setRecipient(i.target.value)}
            />

            <Input
                label='Amount'
                placeholder={`Amount to transfer`}
                value={amount}
                type="number"
                onChange={(i) => { setAmount(i.target.value) }}
            />

            <div className={styles.buttons}>
                <Button
                    onClick={handleClose}
                    type='secondary'
                    className={styles.button}
                >
                    CANCEL
                </Button>

                <Button
                    className={styles.button}
                    onClick={() => { submit() }}
                    type='primary'
                    // loading={loading} // TODO: Implement loading base on the action trigger
                    disabled={disabledTransfer}
                >
                    TRANSFER
                </Button>
            </div>
        </Modal>
    )
}

export default React.memo(TransferDaoModal)