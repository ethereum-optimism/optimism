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
import moment from 'moment';
import truncate from 'truncate-middle';

import { selectLoading } from 'selectors/loadingSelector';

import Pager from 'components/pager/Pager';
import Transaction from 'components/transaction/Transaction';

import networkService from 'services/networkService';

import * as styles from './Transactions.module.scss';

const PER_PAGE = 10;

function Deposits ({ searchHistory, transactions }) {

  const [ page, setPage ] = useState(1);

  // const ethDeposits = useSelector(selectEthDeposits, isEqual);
  // const erc20Deposits = useSelector(selectErc20Deposits, isEqual);
  const loading = useSelector(selectLoading([ 'TRANSACTION/GETALL' ]));

  useEffect(() => {
    setPage(1);
  }, [ searchHistory ]);

  // const deposits = orderBy(
  //   [ ...ethDeposits, ...erc20Deposits ],
  //   i => i.blockNumber, 'desc'
  // );

  const _deposits = transactions.filter(i => {
    return i.hash.includes(searchHistory) && (
      i.to !== null && (
      i.to.toLowerCase() === networkService.L1LPAddress.toLowerCase() ||
      i.to.toLowerCase() === networkService.L1_ETH_Address.toLowerCase() ||
      i.to.toLowerCase() === networkService.L1StandardBridgeAddress.toLowerCase()
      )
    )
  })

  const startingIndex = page === 1 ? 0 : ((page - 1) * PER_PAGE);
  const endingIndex = page * PER_PAGE;
  const paginatedDeposits = _deposits.slice(startingIndex, endingIndex);

  let totalNumberOfPages = Math.ceil(_deposits.length / PER_PAGE);

  //if totalNumberOfPages === 0, set to one so we don't get the strange "page 1 of 0" display
  if (totalNumberOfPages === 0) totalNumberOfPages = 1;

  return (
    <div className={styles.transactionSection}>
      <div className={styles.transactions}>
        <Pager
          currentPage={page}
          isLastPage={paginatedDeposits.length < PER_PAGE}
          totalPages={totalNumberOfPages}
          onClickNext={()=>setPage(page + 1)}
          onClickBack={()=>setPage(page - 1)}
        />
        {!paginatedDeposits.length && !loading && (
          <div className={styles.disclaimer}>Deposit history coming soon...</div>
        )}
        {!paginatedDeposits.length && loading && (
          <div className={styles.disclaimer}>Loading...</div>
        )}
        {paginatedDeposits.map((i, index) => {
          const metaData = typeof(i.typeTX) === 'undefined' ? '' : i.typeTX
          return (
            <Transaction
              key={index}
              link={
                i.chain === 'L1' ? 
                `https://rinkeby.etherscan.io/tx/${i.hash}` :
                `https://blockexplorer.rinkeby.omgx.network/tx/${i.hash}`
              }
              title={truncate(i.hash, 6, 4, '...')}
              midTitle='Deposit'
              subTitle={moment.unix(i.timeStamp).format('lll')}
              subStatus={`Block ${i.blockNumber}`}
              chain={`${i.chain} Chain`}
              typeTX={`${metaData}`}
            />
          );
        })}
      </div>
    </div>
  );
}

export default React.memo(Deposits);
