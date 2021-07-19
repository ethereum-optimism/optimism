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

import React from 'react'
import { useSelector } from 'react-redux'

import { isEqual } from 'lodash'
import truncate from 'truncate-middle'

import { selectLoading } from 'selectors/loadingSelector'
import { selectIsSynced } from 'selectors/statusSelector'

import { selectlayer2Balance, selectlayer1Balance } from 'selectors/balanceSelector'

import AccountList from 'components/accountList/AccountList';

import Copy from 'components/copy/Copy'

import { logAmount } from 'util/amountConvert'
import networkService from 'services/networkService'

import bunny_happy from 'images/bunny_happy.svg'
import bunny_sad from 'images/bunny_sad.svg'

import * as styles from './Account.module.scss'

function Account () {
  
  const childBalance = useSelector(selectlayer2Balance, isEqual);
  const rootBalance = useSelector(selectlayer1Balance, isEqual);

  const isSynced = useSelector(selectIsSynced);
  const criticalTransactionLoading = useSelector(selectLoading([ 'EXIT/CREATE' ]));

  const disabled = criticalTransactionLoading || !isSynced

  let balances = {
    oETH : {have: false, amount: 0, amountShort: '0'}
  }

  childBalance.reduce((acc, cur) => {
    if (cur.symbol === 'oETH' && cur.balance > 0 ) {
      acc['oETH']['have'] = true;
      acc['oETH']['amount'] = cur.balance;
      acc['oETH']['amountShort'] = logAmount(cur.balance, cur.decimals, 2);
    }
    return acc;
  }, balances)

  const wAddress = networkService.account ? truncate(networkService.account, 6, 4, '...') : '';
  const networkLayer = networkService.L1orL2 === 'L1' ? 'L1' : 'L2';

  return (
    <div className={styles.Account}>

      <div className={styles.wallet}>
        <span className={styles.address}>{`Wallet Address : ${wAddress}`}</span>
        <Copy value={networkService.account} />
      </div>

      {balances['oETH']['have'] &&
        <div className={styles.RabbitBox}>
          <img className={styles.bunny} src={bunny_happy} alt='Happy Bunny' />
          <div className={styles.RabbitRight}>
            <div className={styles.RabbitRightTop}>
              OMGX Balance
            </div>
            <div className={styles.RabbitRightMiddle}>
              <div className={styles.happy}>{balances['oETH']['amountShort']}</div>
            </div>
            <div className={styles.RabbitRightBottom}>
              oETH
            </div>
            <div className={styles.RabbitRightBottomNote}>
            {networkLayer === 'L1' && 
              <span>You are on L1. To use the L2, please switch to L2 in MetaMask.</span>
            }
            {networkLayer === 'L2' && 
              <span>You are on L2. To use the L1, please switch to L1 in MetaMask.</span>
            }
            </div>
          </div>
        </div>
      }

      {!balances['oETH']['have'] &&
        <div className={styles.RabbitBox}>
          <img className={styles.bunny} src={bunny_sad} alt='Sad Bunny' />
          <div className={styles.RabbitRight}>
            <div className={styles.RabbitRightTop}>
              OMGX Balance
            </div>
            <div className={styles.RabbitRightMiddle}>
                <div className={styles.sad}>0</div>
            </div>
            <div className={styles.RabbitRightBottom}>
              oETH
            </div>
            <div className={styles.RabbitRightBottomNote}>
            {networkLayer === 'L1' && 
              <span>You are on L1. To use the L2, please switch to L2 in MetaMask.</span>
            }
            {networkLayer === 'L2' && 
              <span>You are on L2. To use the L1, please switch to L1 in MetaMask.</span>
            }
            </div>
          </div>
        </div>
      }

  <div className={styles.BalanceWrapper}>
    <div>
      <div className={styles.title}>
        <span style={{fontSize: '0.8em'}}>Balance on L1</span><br/>
        <span>Ethereum Network</span><br/>
      </div>
      <div className={styles.TableContainer}>
        {rootBalance.map((i, index) => {
          return (
            <AccountList 
              key={i.currency}
              token={i}
              chain={'L1'}
              networkLayer={networkLayer}
              disabled={disabled}
            />
          )
        })}
      </div>
    </div>
    <div>
      <div className={styles.title}>
        <span style={{fontSize: '0.8em'}}>Balance on L2</span><br/>
        <span>OMGX</span><br/>
      </div>
      <div className={styles.TableContainer}>
        {childBalance.map((i, index) => {
          return (
            <AccountList 
              key={i.currency}
              token={i}
              chain={'L2'}
              networkLayer={networkLayer}
              disabled={disabled}
            />
          )
        })}
      </div>
    </div>
  </div>

  </div>
  );

}

export default React.memo(Account);
