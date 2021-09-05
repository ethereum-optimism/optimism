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

import { closeModal } from 'actions/uiAction'

import * as styles from './daoModal.module.scss'

import Modal from 'components/modal/Modal'
import Button from 'components/button/Button'
import Input from 'components/input/Input'
import ProposalAction from './ProposalAction'

function NewProposalModal({ open }) {

    const [title, setTitle] = useState('')
    const [description, setDescription] = useState('')

    const [actionList, setActionList] = useState([])
    const [contracts, setContracts] = useState(['select'])

    const [calldata, setCalldata] = useState(undefined)

    const dispatch = useDispatch()

    const renderActions = () => {
        return actionList.map((action, index) => {
            return <div key={index}>
                <ProposalAction index={index}
                    setActionList={setActionList}
                    actionList={actionList}
                    contracts={contracts}
                    setContracts={setContracts}
                    setCalldata={setCalldata}
                />
            </div>
        })
    }

    const addAction = () => {
        setActionList([...actionList, ''])
        setContracts([...contracts, 'select'])
    }


    function handleClose() {
        setTitle('');
        dispatch(closeModal('newProposalModal'))
    }

    const submit = async () => {
        console.log({
            description,
            title, 
            actionList,
            contracts,
            calldata
        })
    }

    const disabledProposal = !title;

    return (
        <Modal open={open}
            width="700px"
        >
            <h2>New Proposal</h2>
            <div className={styles.modalContent}>
                <div className={styles.proposalAction}>
                    <div className={styles.actionTitle}>
                        <h3>Actions</h3>
                        <Button
                            style={{
                                borderRadius: '8px',
                                width: '110px'
                            }}
                            size="small"
                            type="outline"
                            onClick={addAction}
                        >
                            + Add an action
                        </Button>
                    </div>
                    {renderActions()}

                </div>
                <div className={styles.proposalDetail}>
                    <Input
                        label='Proposal Title'
                        value={title}
                        onChange={i => setTitle(i.target.value)}
                    />
                    <Input
                        label='Proposal Description'
                        value={description}
                        type="textArea"
                        onChange={i => setDescription(i.target.value)}
                    />
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
                    disabled={disabledProposal}
                >
                    SUBMIT PROPOSAL
                </Button>
            </div>
        </Modal >
    )
}

export default React.memo(NewProposalModal)