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
import moment from 'moment'
import { useSelector } from 'react-redux'
import truncate from 'truncate-middle'

import { selectLoading } from 'selectors/loadingSelector'

import ProcessExitsModal from 'containers/modals/processexit/ProcessExitsModal'
import Transaction from 'components/transaction/Transaction'
import Pager from 'components/pager/Pager'

import networkService from 'services/networkService'

import * as styles from './Transactions.module.scss'

const PER_PAGE = 8;

function Exits ({ searchHistory, transactions,chainLink }) {
  
  const [ page, setPage ] = useState(1);
  const [ processExitModal, setProcessExitModal ] = useState(false);
  
  const loading = useSelector(selectLoading([ 'EXIT/GETALL' ]));  

  const _exits = transactions.filter(i => {
    return i.hash.includes(searchHistory) && (
      i.to !== null && (
        i.to.toLowerCase() === networkService.L2LPAddress.toLowerCase() ||
        //i.to.toLowerCase() === networkService.L2_ETH_Address.toLowerCase() ||
        //i.to.toLowerCase() === networkService.L2_TEST_Address.toLowerCase() ||
        i.to.toLowerCase() === networkService.L2StandardBridgeAddress.toLowerCase()
      )
    )
  })

  const renderExits = _exits.map((i, index) => {

    const metaData = typeof(i.typeTX) === 'undefined' ? '' : i.typeTX
    
    let tradExit = false
    let isExitable = false
    let midTitle = 'Swapped: ' + moment.unix(i.timeStamp).format('lll')
    
    const to = i.to.toLowerCase()
        
    //are we dealing with a traditional exit?
    if(to === networkService.L2StandardBridgeAddress.toLowerCase()) {
      
      tradExit = true
      isExitable = moment().isAfter(moment.unix(i.crossDomainMessageEstimateFinalizedTime))
      
      if(isExitable) {
        midTitle = 'Ready to exit - initiated:' + moment.unix(i.timeStamp).format('lll')
      } else {
        const secondsToGo = i.crossDomainMessageEstimateFinalizedTime - Math.round(Date.now() / 1000)
        const daysToGo = Math.floor(secondsToGo / (3600 * 24))
        const hoursToGo = Math.round((secondsToGo % (3600 * 24)) / 3600)
        const time = moment.unix(i.timeStamp).format("mm/DD hh:MM");
        midTitle = `7 day window started ${time}. ${daysToGo} days and ${hoursToGo} hours remaining`
      }

    }

    return (
      <Transaction
        key={`${index}`}
        chain='L2->L1 Exit'
        link={chainLink(i)}
        title={truncate(i.hash, 8, 6, '...')}
        blockNumber={`Block ${i.blockNumber}`}
        midTitle={midTitle}
        button={isExitable && tradExit ? {onClick: ()=>setProcessExitModal(i), text: 'Process Exit'}: undefined}
        typeTX={`${metaData}`}
      />
    )
  })

  const startingIndex = page === 1 ? 0 : ((page - 1) * PER_PAGE);
  const endingIndex = page * PER_PAGE;
  const paginatedExits = renderExits.slice(startingIndex, endingIndex);

  let totalNumberOfPages = Math.ceil(renderExits.length / PER_PAGE);

  //if totalNumberOfPages === 0, set to one so we don't get the strange "Page 1 of 0" display
  if (totalNumberOfPages === 0) totalNumberOfPages = 1;

  return (
    <>
      <ProcessExitsModal
        exitData={processExitModal}
        open={!!processExitModal}
        toggle={()=>setProcessExitModal(false)}
      />
      <div className={styles.section}>
        <div className={styles.transactionSection}>
          <div className={styles.transactions}>
            <Pager
              currentPage={page}
              isLastPage={paginatedExits.length < PER_PAGE}
              totalPages={totalNumberOfPages}
              onClickNext={() => setPage(page + 1)}
              onClickBack={() => setPage(page - 1)}
            />
            {!renderExits.length && !loading && (
              <div className={styles.disclaimer}>Exit history coming soon...</div>
            )}
            {!renderExits.length && loading && (
              <div className={styles.disclaimer}>Loading...</div>
            )}
            {React.Children.toArray(paginatedExits)}
          </div>
        </div>
      </div>
    </>
  );
}

export default React.memo(Exits);
