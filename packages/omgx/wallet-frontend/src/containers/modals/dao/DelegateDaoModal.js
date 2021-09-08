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
import { Typography } from '@material-ui/core'

import { closeModal, openAlert, openError } from 'actions/uiAction';

import * as styles from './daoModal.module.scss';

import Modal from 'components/modal/Modal';
import Input from 'components/input/Input';
import Button from 'components/button/Button';
import { delegateVotes } from 'actions/daoAction';

function DelegateDaoModal({ open }) {
    const [recipient, setRecipient] = useState('');
    const dispatch = useDispatch()

    const disabledTransfer = !recipient;

    function handleClose() {
        setRecipient('');
        dispatch(closeModal('delegateDaoModal'))
    }

    const submit = async () => {
        let res = await dispatch(delegateVotes({ recipient }));
        if (res) {
            dispatch(openAlert(`Votes delegated successfully!`));
            handleClose();
        } else {
            dispatch(openError(`Failed to delegate`));
            handleClose();
        }
    }

    return (
        <Modal open={open}>
            <Typography variant="h2">Delegate Boba</Typography>
            <Input
                label='Delegate Address'
                placeholder='Hash'
                paste
                value={recipient}
                onChange={i => setRecipient(i.target.value)}
            />

            <div className={styles.buttons}>
                <Button
                    onClick={handleClose}
                    color='neutral'
                    className={styles.button}
                >
                    CANCEL
                </Button>

                <Button
                    className={styles.button}
                    onClick={() => { submit() }}
                    color='primary'
                    variant="outlined"
                    // loading={loading} // TODO: Implement loading base on the action trigger
                    disabled={disabledTransfer}
                >
                    Delegate
                </Button>
            </div>
        </Modal>
    )
}

export default React.memo(DelegateDaoModal)