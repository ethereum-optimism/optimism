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

import React, { useState, useEffect } from 'react';

import { useDispatch } from 'react-redux';

import { openAlert, openError } from 'actions/uiAction';

import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import Button from 'components/button/Button';

import * as styles from './Proposal.module.scss';

import { castProposalVote } from 'actions/daoAction';

function Proposal({
    proposal,
}) {
    const dispatch = useDispatch()

    const [dropDownBox, setDropDownBox] = useState(false)
    const [dropDownBoxInit, setDropDownBoxInit] = useState(true)

    const [votePercent, setVotePercent] = useState(undefined)
    
    useEffect(() => {
        const init = async () => {
            if (proposal.totalVotes > 0) {
                setVotePercent(Math.round((100 * proposal.forVotes) / proposal.totalVotes));
            } else {
                setVotePercent(50);
            }
        };
        init();
    }, [proposal])


    const updateVote = async (id, userVote, label) => {
        let res = await dispatch(castProposalVote({id, userVote}));
        if(res) {
            dispatch(openAlert(`${label}`));
        } else {
            dispatch(openError(`Failed to cast vote!`));
        }
    }

    return (<div
        className={styles.proposalCard}

        style={{
            background: 'linear-gradient(132.17deg, rgba(255, 255, 255, 0.1) 0.24%, rgba(255, 255, 255, 0.03) 94.26%)',
            borderRadius: '12px'
        }}>

        {proposal.state === 'Active' &&
            <div
                onClick={() => {
                    setDropDownBox(!dropDownBox);
                    setDropDownBoxInit(false);
                }}
            >
                <div className={styles.proposalHeader}>
                    <div className={styles.title}>
                        <p>Proposal #{proposal.id}</p>
                        <p className={styles.muted}>Title: {proposal.description}</p>
                        <p className={styles.muted}>Status: {proposal.state}</p>
                        <p className={styles.muted}>Start L1 Block: {proposal.startBlock} &nbsp; &nbsp; End L1 Block: {proposal.endBlock}</p>
                    </div>
                    <ExpandMoreIcon /> VOTE
                </div>
                <div className={styles.proposalContent}>
                    <div className={styles.vote}>For: {proposal.forVotes}</div>
                    <div className={styles.vote}>Against: {proposal.againstVotes}</div>
                    <div className={styles.vote}>Abstain: {proposal.abstainVotes}</div>
                    <div className={styles.vote} style={{minWidth: '150px'}}>Percentage For: {votePercent}% </div>
                    <div className={styles.vote}>Total Votes: {proposal.totalVotes}</div>
                </div>
            </div>
        }

        {proposal.state !== 'Active' &&
            <div
            >
                <div className={styles.proposalHeader}>
                    <div className={styles.title}>
                        <p>Proposal #{proposal.id}</p>
                        <p className={styles.muted}>Title: {proposal.description}</p>
                        <p className={styles.muted}>Status: {proposal.state}</p>
                        <p className={styles.muted}>Start L1 Block: {proposal.startBlock} End L1 Block: {proposal.endBlock}</p>
                    </div>
                </div>
                <div className={styles.proposalContent}>
                    <div className={styles.vote}>For: {proposal.forVotes}</div>
                    <div className={styles.vote}>Against: {proposal.againstVotes}</div>
                    <div className={styles.vote}>Abstain: {proposal.abstainVotes}</div>
                    <div className={styles.vote} style={{minWidth: '150px'}}>Percentage For: {votePercent}% </div>
                    <div className={styles.vote}>Total Votes: {proposal.totalVotes}</div>
                </div>
            </div>
        }

        <div className={dropDownBox ? styles.dropDownContainer : dropDownBoxInit ? styles.dropDownInit : styles.closeDropDown}>
            <div className={styles.proposalDetail}>
                <Button
                    type="primary"
                    variant="outlined"
                    style={{
                        maxWidth: '180px',
                        padding: '15px 10px',
                        borderRadius: '8px',
                        alignSelf: 'center'
                    }}
                    onClick={(e) => {
                        updateVote(proposal.id, 1, 'Cast Vote For')
                    }}

                > Cast Vote For</Button>
                <Button
                    type="primary"
                    variant="contained"
                    style={{
                        maxWidth: '180px',
                        padding: '15px 10px',
                        borderRadius: '8px',
                        alignSelf: 'center'
                    }}
                    onClick={(e) => {
                        updateVote(proposal.id, 0, 'Cast Vote Against')
                    }}

                > Cast Vote Against</Button>
                <Button
                    type="outline"
                    variant="outlined"
                    style={{
                        maxWidth: '180px',
                        padding: '15px 10px',
                        borderRadius: '8px',
                        alignSelf: 'center'
                    }}
                    onClick={(e) => {
                        updateVote(proposal.id, 2, 'Cast Vote Abstain')
                    }}
                > Cast Vote Abstain</Button>
            </div>
        </div>
    </div>)
}


export default React.memo(Proposal);