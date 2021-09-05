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

import React from 'react';
import { useDispatch, useSelector } from 'react-redux';

import { openModal } from 'actions/uiAction';

import Button from 'components/button/Button';
import Proposal from 'components/Proposal/Proposal';

import * as styles from './proposalList.module.scss'
import { selectProposals } from 'selectors/daoSelector';
import { selectLoading } from 'selectors/loadingSelector';

function ProposalList() {

    const dispatch = useDispatch()
    const loading = useSelector(selectLoading(['PROPOSAL/GET']))
    const proposals = useSelector(selectProposals)

    return <>
        <div className={styles.containerAction}>
            <p className={styles.listTitle}>Proposal List</p>
            <Button
                type="outline"
                onClick={() => {
                    dispatch(openModal('newProposalModal'))
                }}
                style={{
                    maxWidth: '180px',
                    padding: '10px',
                    borderRadius: '8px',
                    alignSelf: 'center'
                }}

            > Create Proposal </Button>
        </div>
        <div className={styles.listContainer}>
            {!!loading ? <div className={styles.loadingContainer}> Loading... </div> : null}
            {proposals.map((p, index) => {
                return <React.Fragment key={index}><Proposal /></React.Fragment>
            })}
        </div>
    </>
}

export default React.memo(ProposalList);

