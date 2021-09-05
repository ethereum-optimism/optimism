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
import { useDispatch,useSelector } from 'react-redux';

import { openModal } from 'actions/uiAction';

import * as styles from './Dao.module.scss';


import Button from 'components/button/Button';
import ProposalList from './proposal/ProposalList';
import { selectDaoBalance, selectDaoVotes } from 'selectors/daoSelector';


function DAO() {
    const dispatch = useDispatch();

    const balance = useSelector(selectDaoBalance);
    const votes = useSelector(selectDaoVotes);

    console.log(['balance', balance])
    console.log(['votes', votes])

    return (
        <div className={styles.container}>
            <div className={styles.header}>
                <h2 className={styles.title}>
                    BOBA DAO
                </h2>
            </div>
            <div className={styles.content}>
                <div className={styles.action}>
                    <div className={styles.tranferContainer}>
                        <div className={styles.info}>
                            <h3 className={styles.title}>{balance} Comp</h3>
                            <h4 className={styles.subTitle}>Wallet Balance</h4>
                            <div className={styles.helpText}>Expanation Here - Help Text Help Text Help Text?</div>
                        </div>
                        <Button
                            type="primary"
                            onClick={() => {
                                dispatch(openModal('transferDaoModal'))
                            }}
                            style={{
                                width: '60%',
                                padding: '15px 10px',
                                borderRadius: '8px',
                                alignSelf: 'center'
                            }}
                        > Transfer Governance Token</Button>
                    </div>
                    <div className={styles.delegateCotainer}>
                        <div className={styles.info}>
                            <h3 className={styles.title}>{votes} Votes</h3>
                            <h4 className={styles.subTitle}>Voting Power</h4>
                            <div className={styles.helpText}>Expanation Here - What does it mean to delegate Votes?</div>
                        </div>
                        <Button
                            type="primary"
                            onClick={() => {
                                dispatch(openModal('delegateDaoModal'))
                            }}
                            style={{
                                width: '60%',
                                padding: '15px 10px',
                                borderRadius: '8px',
                                alignSelf: 'center'
                            }}

                        > Delegate Votes</Button>
                    </div>
                </div>
                <div className={styles.proposal}>
                    <ProposalList />
                </div>
            </div>
        </div>
    )
}


export default React.memo(DAO);


