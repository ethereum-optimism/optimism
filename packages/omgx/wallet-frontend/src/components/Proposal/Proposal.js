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
import { utils } from 'ethers';
import { useDispatch } from 'react-redux';

import { openAlert } from 'actions/uiAction';

import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import Button from 'components/button/Button';

import * as styles from './Proposal.module.scss';
import networkService from 'services/networkService';

function Proposal({
    id,
    proposal,
}) {
    const dispatch = useDispatch()
    
    const { delegate } = networkService

    const [dropDownBox, setDropDownBox] = useState(false)
    const [dropDownBoxInit, setDropDownBoxInit] = useState(true)

    const [votePercent, setVotePercent] = useState(undefined)
    const [totalVotes, setTotalVotes] = useState(undefined)

    console.log("Delegate address:", delegate.address)

    useEffect(() => {
        const init = async () => {
            if (proposal && proposal[0][0] === delegate.address) {
                const proposalData = await delegate.proposals(id);
                let forVotes = utils.formatEther(proposalData.forVotes);
                forVotes = parseInt(forVotes);
                let againstVotes = utils.formatEther(proposalData.againstVotes);
                againstVotes = parseInt(againstVotes);
                const totalVotes = forVotes + againstVotes;
                setTotalVotes(totalVotes);

                if (totalVotes > 0) {
                    setVotePercent(Math.round((100 * forVotes) / totalVotes));
                } else {
                    setVotePercent(50);
                }
            }
        };
        init();
    }, [delegate, id, proposal])


    const updateVote = async (e, userVote, label) => {
        console.log('update vote');
        await delegate.castVote(id, userVote);
        // show alert
        dispatch(openAlert(`${label}`));
    }

    return (<div
        className={styles.proposalCard}
        style={{
            background: `${!!dropDownBox ? 'linear-gradient(132.17deg, rgba(255, 255, 255, 0.019985) 0.24%, rgba(255, 255, 255, 0.03) 94.26%)' : 'none'}`,
            borderRadius: `${!!dropDownBox ? '12px' : ''}`
        }}>
        <div
            onClick={() => {
                setDropDownBox(!dropDownBox);
                setDropDownBoxInit(false);
            }}>
            <div className={styles.proposalHeader}>
                <div className={styles.title}>
                    <p>Proposal #{Math.random() * 100}</p>
                    <p className={styles.muted}>
                        signaturesignaturesignature
                    </p>
                </div>
                <ExpandMoreIcon />
            </div>
            <div className={styles.proposalContent}>
                <div>For Votes : <span>10</span> </div>
                <div>Against Votes : <span>1231</span> </div>
                <div>Abstain Votes : <span>8912</span> </div>
                <div>Vote Percentage : {votePercent}% </div>
                <div>Total Votes : {totalVotes} </div>
            </div>
        </div>

        <div
            className={dropDownBox ?
                styles.dropDownContainer : dropDownBoxInit ? styles.dropDownInit : styles.closeDropDown}
        >
            <div className={styles.proposalDetail}>
                <Button
                    type="outline"
                    style={{
                        maxWidth: '180px',
                        padding: '15px 10px',
                        borderRadius: '8px',
                        alignSelf: 'center'
                    }}
                    onClick={(e) => {
                        updateVote(e, 1, 'Cast Vote For')
                    }}

                > Cast Vote For</Button>
                <Button
                    type="primary"
                    style={{
                        maxWidth: '180px',
                        padding: '15px 10px',
                        borderRadius: '8px',
                        alignSelf: 'center'
                    }}
                    onClick={(e) => {
                        updateVote(e, 0, 'Cast Vote Against')
                    }}

                > Cast Vote Against</Button>
                <Button
                    type="outline"
                    style={{
                        maxWidth: '180px',
                        padding: '15px 10px',
                        borderRadius: '8px',
                        alignSelf: 'center'
                    }}
                    onClick={(e) => {
                        updateVote(e, 2, 'Cast Vote Abstain')
                    }}
                > Cast Vote Abstain</Button>
            </div>

        </div>
        <div className={styles.divider}></div>
    </div>)
}


export default React.memo(Proposal);