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
import { isEqual } from 'lodash';
import { useSelector } from 'react-redux';

import BN from 'bn.js';
import moment from 'moment';
import truncate from 'truncate-middle';

import { setActiveHistoryTab1 } from 'actions/uiAction'
import { setActiveHistoryTab2 } from 'actions/uiAction'

import { selectActiveHistoryTab1 } from 'selectors/uiSelector'
import { selectActiveHistoryTab2 } from 'selectors/uiSelector'

import { selectLoading } from 'selectors/loadingSelector'

import networkService from 'services/networkService'
import { selectNetworkBURL } from 'selectors/setupSelector';

import Tabs from 'components/tabs/Tabs'
import Input from 'components/input/Input'
import Transaction from 'components/transaction/Transaction'
import Pager from 'components/pager/Pager'

import Exits from './Exits';
import Deposits from './Deposits';

import * as styles from './Transactions.module.scss';

const PER_PAGE = 5;

function Transactions () {

  const dispatch = useDispatch();

  const [ page1, setPage1 ] = useState(1);
  const [ page2, setPage2 ] = useState(1);
  
  const [ searchHistory, setSearchHistory ] = useState('');

  const loading = useSelector(selectLoading([ 'TRANSACTION/GETALL' ]));

  const activeTab1 = useSelector(selectActiveHistoryTab1, isEqual);
  const activeTab2 = useSelector(selectActiveHistoryTab2, isEqual);

  const transactions = [];

  const blockexplorerURL = useSelector(selectNetworkBURL());

  function renderStatus (utxo) {
    if (utxo.status === 'Pending') {
      return 'Pending';
    }
    const total = utxo.outputs.reduce((prev, curr) => {
      if (curr.owner !== networkService.account) {
        return prev.add(new BN(curr.amount));
      }
      return prev;
    }, new BN(0));
    return `${total.toString()}`;
  }

  const _transactions = transactions.filter(i => {
    return i.txhash.includes(searchHistory) || i.metadata.includes(searchHistory);
  });

  const startingIndex = page1 === 1 ? 0 : ((page1 - 1) * PER_PAGE);
  const endingIndex = page1 * PER_PAGE;
  const paginatedTransactions = _transactions.slice(startingIndex, endingIndex);

  let totalNumberOfPages = Math.ceil(_transactions.length / PER_PAGE);

  //if totalNumberOfPages === 0, set to one so we don't get the strange "page 1 of 0" display
  if (totalNumberOfPages === 0) totalNumberOfPages = 1;

  return (
    <div className={styles.container}>

      <div className={styles.header}>
        <h2>Search</h2>
        <Input
          icon
          placeholder='Search history'
          value={searchHistory}
          onChange={i => {
            setPage1(1);
            setSearchHistory(i.target.value);
          }}
          className={styles.searchBar}
        />
      </div>

      <div className={styles.data}>

        <div className={styles.section}>
        
          <Tabs
            onClick={tab => {
              setPage1(1);
              dispatch(setActiveHistoryTab1(tab));
            }}
            activeTab={activeTab1}
            tabs={[ 'Transactions', 'Deposits' ]}
          />

          {activeTab1 === 'Transactions' && (
            <div className={styles.transactions}>
              <Pager
                currentPage={page1}
                isLastPage={paginatedTransactions.length < PER_PAGE}
                totalPages={totalNumberOfPages}
                onClickNext={()=>setPage1(page1 + 1)}
                onClickBack={()=>setPage1(page1 - 1)}
              />
              {!paginatedTransactions.length && !loading && (
                <div className={styles.disclaimer}>No transaction history.</div>
              )}
              {!paginatedTransactions.length && loading && (
                <div className={styles.disclaimer}>Loading...</div>
              )}
              {paginatedTransactions.map((i, index) => {
                return (
                  <Transaction
                    key={index}
                    link={
                      i.status === 'Pending'
                        ? undefined
                        : `${blockexplorerURL}/transaction/${i.txhash}`
                    }
                    title={`${truncate(i.txhash, 6, 4, '...')}`}
                    midTitle={i.metadata ? i.metadata : '-'}
                    subTitle={moment.unix(i.block.timestamp).format('lll')}
                    status={renderStatus(i)}
                    subStatus={`Block ${i.block.blknum}`}
                  />
                );
              })}
            </div>
          )}

          {activeTab1=== 'Deposits' && <
            Deposits searchHistory={searchHistory} />
          }

        </div>

        <div className={styles.section}>
          <Tabs
            onClick={tab => {
              setPage2(1);
              dispatch(setActiveHistoryTab2(tab));
            }}
            activeTab={activeTab2}
            tabs={[ 'Exits', 'TBD' ]}
          />

          {activeTab2 === 'Exits' && 
            <Exits searchHistory={searchHistory} />
          }

        </div>

      </div>
    </div>
  );
}

export default React.memo(Transactions);
