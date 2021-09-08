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

import { Box, Grid } from '@material-ui/core';
import { fetchTransactions } from 'actions/networkAction';
import PageHeader from 'components/pageHeader/PageHeader';
import StyledTabs from 'components/tabs';
import Deposits from 'containers/history/deposits';
import Exits from 'containers/history/exits';
import Transactions from 'containers/history/transactions';
import { isEqual, orderBy } from 'lodash';
import moment from 'moment';
import React, { useState } from 'react';
import DatePicker from 'react-datepicker';
import "react-datepicker/dist/react-datepicker.css";
import { batch, useDispatch, useSelector } from 'react-redux';
import { selectNetwork } from 'selectors/setupSelector';
import { selectTransactions } from 'selectors/transactionSelector';
import { POLL_INTERVAL } from 'util/constant';
import { getAllNetworks } from 'util/masterConfig';
import useInterval from 'util/useInterval';
import {
  PageContent
} from '../page.style';

function HistoryPage() {
  const dispatch = useDispatch();
  const [selectedTab, setSelectedTab] = useState(0);
  const [startDate, setStartDate] = useState(null);
  const [endDate, setEndDate] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');

  const tabList = ['All', 'Deposits', 'Exits']

  const unorderedTransactions = useSelector(selectTransactions, isEqual);
  const orderedTransactions = orderBy(unorderedTransactions, i => i.timeStamp, 'desc');

  const transactions = orderedTransactions.filter((i) => {
    if (startDate && endDate) {
      return (moment.unix(i.timeStamp).isSameOrAfter(startDate) && moment.unix(i.timeStamp).isSameOrBefore(endDate));
    }
    return true;
  })

  const currentNetwork = useSelector(selectNetwork());

  const nw = getAllNetworks();

  const chainLink = (item) => {
    let network = nw[currentNetwork];
    if (!!network && !!network[item.chain]) {
      // network object should have L1 & L2
      return `${network[item.chain].transaction}${item.hash}`;
    }
    return '';
  }

  useInterval(() => {
    batch(() => {
      dispatch(fetchTransactions());
    });
  }, POLL_INTERVAL * 2);

  console.log(['transactions', transactions]);

  const onTabChagne = (event, newValue) => {
    console.log([event, newValue]);
    setSelectedTab(newValue);
  }

  return (
    <PageContent>
      <PageHeader title="Transaction History" />
      <Grid
        justifyContent="space-between"
        display="flex"
      >
        <StyledTabs
          selectedTab={selectedTab}
          onChange={onTabChagne}
          optionList={tabList}
          isSearch={true}
        />
        <Box display="flex" 
          alignItems="center"
          justifyContent="space-around"
          >
          <Box style={{marginRight:'10px'}}>
            <span style={{marginRight: '10px'}}> Show period from</span>
            <DatePicker
              selected={startDate}
              onChange={(date) => setStartDate(date)}
              selectsStart
              startDate={startDate}
              endDate={endDate}
            />
          </Box>
          <Box>
          <span style={{marginRight: '10px'}}> To </span>
            <DatePicker
              selected={endDate}
              onChange={(date) => setEndDate(date)}
              selectsEnd
              startDate={startDate}
              endDate={endDate}
              minDate={startDate}
            />
          </Box>
        </Box>
      </Grid>
      {tabList[selectedTab] === 'All' && transactions && transactions.length > 0 ?
        <Transactions
          transactions={transactions}
          chainLink={chainLink}
        /> : null
      }
      {tabList[selectedTab] === 'Deposits' ?
        <Deposits
          transactions={transactions}
          chainLink={chainLink}
        /> : null
      }
      {tabList[selectedTab] === 'Exits' ?
        <Exits
          transactions={transactions}
          chainLink={chainLink}
        /> : null
      }
    </PageContent>
  );

}

export default React.memo(HistoryPage);
