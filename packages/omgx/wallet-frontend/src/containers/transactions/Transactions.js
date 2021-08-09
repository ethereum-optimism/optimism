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
import { useSelector } from 'react-redux';
import "react-datepicker/dist/react-datepicker.css";

import moment from 'moment';
import truncate from 'truncate-middle';

import { selectLoading } from 'selectors/loadingSelector'

import Transaction from 'components/transaction/Transaction'
import Pager from 'components/pager/Pager'

import * as styles from './Transactions.module.scss';

const PER_PAGE = 8;

function Transactions({ searchHistory, transactions, chainLink }) {
  const [page, setPage] = useState(1);
  const loading = useSelector(selectLoading(['EXIT/GETALL']));
  useEffect(() => {
    setPage(1);
  }, [searchHistory]);

  const _transactions = transactions.filter(i => {
    return i.hash.includes(searchHistory);
  });

  const startingIndex = page === 1 ? 0 : ((page - 1) * PER_PAGE);
  const endingIndex = page * PER_PAGE;
  const paginatedTransactions = _transactions.slice(startingIndex, endingIndex);

  let totalNumberOfPages = Math.ceil(_transactions.length / PER_PAGE);

  //if totalNumberOfPages === 0, set to one so we don't get the strange "page 1 of 0" display
  if (totalNumberOfPages === 0) totalNumberOfPages = 1

  return (<div className={styles.transactions}>
    <Pager
      currentPage={page}
      isLastPage={paginatedTransactions.length < PER_PAGE}
      totalPages={totalNumberOfPages}
      onClickNext={() => setPage(page + 1)}
      onClickBack={() => setPage(page - 1)}
    />
    {!paginatedTransactions.length && !loading && (
      <div className={styles.disclaimer}>Transaction history coming soon...</div>
    )}
    {!paginatedTransactions.length && loading && (
      <div className={styles.disclaimer}>Loading...</div>
    )}
    {paginatedTransactions.map((i, index) => {
      const metaData = typeof (i.typeTX) === 'undefined' ? '' : i.typeTX
      const {
        l1BlockHash,
        l1BlockNumber,
        l1From,
        l1Hash,
        l1To
      } = i;
      return (
        <Transaction
          key={index}
          link={chainLink(i)}
          title={`${truncate(i.hash, 8, 6, '...')}`}
          midTitle={moment.unix(i.timeStamp).format('lll')}
          blockNumber={`Block ${i.blockNumber}`}
          chain={`${i.chain} Chain`}
          typeTX={`${metaData}`}
          detail={l1Hash ? {
            l1BlockHash: truncate(l1BlockHash, 8, 6, '...'),
            l1BlockNumber,
            l1From,
            l1Hash: truncate(l1Hash, 8, 6, '...'),
            l1To,
            l1TxLink: chainLink({
              chain: i.chain,
              hash: l1Hash
            })
          } : null}
        />
      )
    })}
  </div>)
}

export default React.memo(Transactions);
