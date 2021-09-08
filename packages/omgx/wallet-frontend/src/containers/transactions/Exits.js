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
import {Grid, Box} from '@material-ui/core'
import moment from 'moment'
import { useSelector } from 'react-redux'
import truncate from 'truncate-middle'

import { selectLoading } from 'selectors/loadingSelector'

import ProcessExitsModal from 'containers/modals/processexit/ProcessExitsModal'
import Transaction from 'components/transaction/Transaction'
import Pager from 'components/pager/Pager'

import networkService from 'services/networkService'

import * as styles from './Transactions.module.scss'

import * as S from './history.styles';

const PER_PAGE = 8

function Exits({ searchHistory, transactions, chainLink }) {

  const [page, setPage] = useState(1);
  const [processExitModal, setProcessExitModal] = useState(false);

  const loading = useSelector(selectLoading(['EXIT/GETALL']));

  const _exits = transactions.filter(i => {
    return i.hash.includes(searchHistory) && (
      i.to !== null && (
        i.to.toLowerCase() === networkService.L2LPAddress.toLowerCase() ||
        //i.to.toLowerCase() === networkService.L2_ETH_Address.toLowerCase() ||
        i.to.toLowerCase() === networkService.L2StandardBridgeAddress.toLowerCase()
      )
    )
  })

  const renderExits = _exits.map((i, index) => {

    const metaData = typeof (i.typeTX) === 'undefined' ? '' : i.typeTX
    const chain = (i.chain === 'L1pending') ? 'L1' : i.chain

    let tradExit = false
    let isExitable = false
    let details = null

    let timeLabel = moment.unix(i.timeStamp).format('lll')

    const to = i.to.toLowerCase()

    //are we dealing with a traditional exit?
    if (to === networkService.L2StandardBridgeAddress.toLowerCase()) {
      tradExit = true

      isExitable = moment().isAfter(moment.unix(i.crossDomainMessage.crossDomainMessageEstimateFinalizedTime))

      if (isExitable) {
        timeLabel = 'Ready to exit - initiated:' + moment.unix(i.timeStamp).format('lll')
      } else {
        const secondsToGo = i.crossDomainMessage.crossDomainMessageEstimateFinalizedTime - Math.round(Date.now() / 1000)
        const daysToGo = Math.floor(secondsToGo / (3600 * 24))
        const hoursToGo = Math.round((secondsToGo % (3600 * 24)) / 3600)
        const time = moment.unix(i.timeStamp).format('MM/DD/YYYY hh:mm a')//.format("mm/dd hh:mm")
        timeLabel = `7 day window started ${time}. ${daysToGo} days and ${hoursToGo} hours remaining`
      }

    }

    if( i.crossDomainMessage && i.crossDomainMessage.l1BlockHash ) {
      details = {
        blockHash: i.crossDomainMessage.l1BlockHash,
        blockNumber: i.crossDomainMessage.l1BlockNumber,
        from: i.crossDomainMessage.l1From,
        hash: i.crossDomainMessage.l1Hash,
        to: i.crossDomainMessage.l1To,
      }
    }

    return (
      <Transaction
        key={`${index}`}
        chain='L2->L1 Exit'
        title={`${chain} Hash: ${i.hash}`}
        blockNumber={`Block ${i.blockNumber}`}
        time={timeLabel}
        button={isExitable && tradExit ? { onClick: () => setProcessExitModal(i), text: 'Process Exit' } : undefined}
        typeTX={`TX Type: ${metaData}`}
        detail={details}
        oriChain={chain}
        oriHash={i.hash}
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
        toggle={() => setProcessExitModal(false)}
      />
      <div className={styles.section}>
        <div className={styles.transactionSection}>
          <S.HistoryContainer>
            <Pager
              currentPage={page}
              isLastPage={paginatedExits.length < PER_PAGE}
              totalPages={totalNumberOfPages}
              onClickNext={() => setPage(page + 1)}
              onClickBack={() => setPage(page - 1)}
            />

            <Grid item xs={12}>
              <Box>
                <S.Content>
                  {!renderExits.length && !loading && (
                    <div className={styles.disclaimer}>Scanning for exits...</div>
                  )}
                  {!renderExits.length && loading && (
                    <div className={styles.disclaimer}>Loading...</div>
                  )}
                  {React.Children.toArray(paginatedExits)}
                </S.Content>
              </Box>
            </Grid>
          </S.HistoryContainer>
        </div>
      </div>
    </>
  );
}

export default React.memo(Exits);
