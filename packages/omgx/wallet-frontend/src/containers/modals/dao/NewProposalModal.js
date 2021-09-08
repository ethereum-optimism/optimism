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
import React, { useState } from 'react'
import { useDispatch } from 'react-redux'

import { closeModal, openAlert, openError } from 'actions/uiAction'

import * as styles from './daoModal.module.scss'

import Modal from 'components/modal/Modal'
import Button from 'components/button/Button'
import Input from 'components/input/Input'
import ProposalAction from './ProposalAction'
import { createDaoProposal } from 'actions/daoAction'

function NewProposalModal({ open }) {
    const dispatch = useDispatch()

    const [action, setAction] = useState(null);
    const [votingThreshold, setVotingThreshold] = useState(0);
    const [proposeText, setProposeText] = useState();

    const onActionChange = (e) =>{
        setVotingThreshold('0');
        setProposeText('');
        setAction(e.target.value);
    }

    function handleClose() {
        setVotingThreshold(null);
        setAction(null);
        dispatch(closeModal('newProposalModal'))
    }

    const submit = async () => {
        let res = await dispatch(createDaoProposal({ votingThreshold, text: proposeText }));
    
        if (res) {
            dispatch(openAlert(`Proposal has been submitted!`))
            handleClose()
        } else {
            dispatch(openError(`Failed to create proposal`));
            handleClose()
        }
    }

    const disabledProposal = () => {
        if (action === 'change-threshold') {
            return !votingThreshold
        } else {
            return !proposeText
        }
    };

    return (
        <Modal open={open}
            onClose={handleClose}
            width="400px"
        >
            <h2>New Proposal</h2>
            <div className={styles.modalContent}>
                <div className={styles.proposalAction}>

                    <select
                        className={styles.actionPicker}
                        onChange={onActionChange}
                    >
                        <option>Select Proposal Type...</option>
                        <option value="change-threshold">Change Voting Threshold</option>
                        <option value="text-proposal">Propose text</option>
                    </select>
                    {action === 'change-threshold' && <Input
                        label="Enter voting threshold"
                        placeholder="0000"
                        value={votingThreshold}
                        type="number"
                        onChange={(i) => setVotingThreshold(i.target.value)}
                        variant="standard"
                        newStyle
                    />
                    }
                    {action === 'text-proposal' && <Input
                        label="Enter proposal text"
                        value={proposeText}
                        onChange={(i) => setProposeText(i.target.value)}
                        variant="standard"
                        newStyle
                    />
                    }
                </div>
            </div>

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
                    onClick={() => { submit({ useLedgerSign: false }) }}
                    type='primary'
                    // loading={loading} // TODO: Implement loading base on the action trigger
                    disabled={disabledProposal()}
                >
                    PROPOSE
                </Button>
            </div>
        </Modal >
    )
}

export default React.memo(NewProposalModal)