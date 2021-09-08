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
import { batch, useDispatch } from 'react-redux';
import { isEqual, orderBy } from 'lodash';
import { useSelector } from 'react-redux';
import DatePicker from 'react-datepicker';
import "react-datepicker/dist/react-datepicker.css";

import moment from 'moment';

import { setActiveHistoryTab1 } from 'actions/uiAction'
import { fetchTransactions } from 'actions/networkAction';

import { selectActiveHistoryTab1 } from 'selectors/uiSelector'
import { selectTransactions } from 'selectors/transactionSelector';
import { selectNetwork } from 'selectors/setupSelector'

import Tabs from 'components/tabs/Tabs'

import Exits from './Exits';
import Deposits from './Deposits';

import * as styles from './Transactions.module.scss';

import { getAllNetworks } from 'util/masterConfig';
import useInterval from 'util/useInterval';
import PageHeader from 'components/pageHeader/PageHeader';
import Transactions from './Transactions';

const POLL_INTERVAL = 5000; //milliseconds

function History () {

  const dispatch = useDispatch()
  const [startDate, setStartDate] = useState(null)
  const [endDate, setEndDate] = useState(null)

  const [ searchHistory, setSearchHistory ] = useState('')

  const activeTab1 = useSelector(selectActiveHistoryTab1, isEqual)

  const unorderedTransactions = useSelector(selectTransactions, isEqual)

  //sort transactions by timeStamp
  const orderedTransactions = orderBy(unorderedTransactions, i => i.timeStamp, 'desc')
  //'desc' or 'asc'

  const transactions = orderedTransactions.filter((i)=>{
    if(startDate && endDate) {
      return (moment.unix(i.timeStamp).isSameOrAfter(startDate) && moment.unix(i.timeStamp).isSameOrBefore(endDate));
    }
    return true;
  })

  useInterval(() => {
    batch(() => {
      dispatch(fetchTransactions());
    });
  }, POLL_INTERVAL * 2);

  return (
    <>
      <PageHeader title="Transaction History" />

    {
    /*TODO: fix the search history
    <Input
            icon
            placeholder='Search by hash'
            value={searchHistory}
            onChange={i => {
              setSearchHistory(i.target.value);
            }}
            className={styles.searchBar}
          />

    */}

      <div className={styles.header}>
        <div className={styles.actions}>
          <div style={{margin: '0px 10px', opacity: 0.7}}>Show period from </div>
          <DatePicker
            wrapperClassName={styles.datePickerInput}
            selected={startDate}
            onChange={(date) => setStartDate(date)}
            selectsStart
            startDate={startDate}
            endDate={endDate}
          />

          <div style={{margin: '0px 10px', opacity: 0.7}}>to </div>
          <DatePicker
            wrapperClassName={styles.datePickerInput}
            selected={endDate}
            onChange={(date) => setEndDate(date)}
            selectsEnd
            startDate={startDate}
            endDate={endDate}
            minDate={startDate}
          />
        </div>
      </div>
      <div className={styles.data}>
        <div className={styles.section}>
          <Tabs
            onClick={tab => {
              dispatch(setActiveHistoryTab1(tab));
            }}
            activeTab={activeTab1}
            tabs={['All', 'Deposits', 'Exits']}
          />

          {activeTab1 === 'All' && (
            <Transactions
              searchHistory={searchHistory}
              transactions={transactions}
            />
          )}

          {activeTab1 === 'Deposits' &&
            <Deposits
              searchHistory={searchHistory}
              transactions={transactions}
            />
          }

          {activeTab1 === 'Exits' &&
            <Exits
              searchHistory={searchHistory}
              transactions={transactions}
            />
          }
        </div>
      </div>
    </>
  );
}

export default React.memo(History);
