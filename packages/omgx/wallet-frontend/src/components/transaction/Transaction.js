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

import React, {useState} from 'react';

import Tooltip from 'components/tooltip/Tooltip';
import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import LinkIcon from 'components/Icons/LinkIcon';
import * as styles from './Transaction.module.scss';

function Transaction ({
  link,
  status,
  statusPercentage,
  subStatus,
  button,
  title,
  midTitle,
  subTitle,
  chain,
  typeTX,
  blockNumber,
  tooltip = '',
  detail
}) {

  const [dropDownBox, setDropDownBox] = useState(false);
  const [dropDownBoxInit, setDropDownBoxInit] = useState(true);

  function renderValue () {
    
    if (button) {
      return (
        <div className={styles.statusContainer}>
          <div
            onClick={button.onClick}
            className={styles.button}
          >
            {button.text}
          </div>
          <div>{subStatus}</div>
        </div>
      )
    }
    
    return (
      <div className={styles.statusContainer}>
        <div className={styles.status}>
          <div
            className={[
              styles.indicator,
              status === 'Pending' ? styles.pending : '',
              status === 'Exited' ? styles.exited : '',
              status === 'Failed' ? styles.failed : ''
            ].join(' ')}
          />
          <span>{status}</span>
          {status === 'Pending' && !!statusPercentage && (
            <Tooltip title={tooltip}>
              <span className={styles.percentage}>
                {`(${Math.max(statusPercentage, 0)}%)`}
              </span>
            </Tooltip>
          )}
        </div>
        <div>{subStatus}</div>
      </div>
    )
  }

  function renderDetail() {
    if (!detail) {
      return null;
    }
    return <> 
      <div className={`${styles.subTitle} ${styles.viewMore}`} style={{ cursor: 'pointer' }}
        onClick={() => {
          setDropDownBox(!dropDownBox)
          setDropDownBoxInit(false)
        }}
      >
        <div>View More</div>
        <ExpandMoreIcon />
      </div>
      <div
        className={dropDownBox ?
          styles.dropDownContainer : dropDownBoxInit ? styles.dropDownInit : styles.closeDropDown}
      >
        <div className={styles.title}>
        <a className={styles.href} href={detail.l1TxLink} target="_blank" rel="noopener noreferrer">
          {detail.l1Hash}
        </a>
        </div>
        <div className={styles.content}>
          <div>L1 Block : {detail.l1BlockNumber}</div>
        </div>
        <div className={styles.content}>
          <div>Block Hash : {detail.l1BlockHash}</div>
        </div>
        <div className={styles.content}>
          <div>L1 From : {detail.l1From}</div>
        </div>
        <div className={styles.content}>
          <div>L1 To : {detail.l1To}</div>
        </div>
      </div>
    </>
  }

  return (
    <div className={styles.Transaction}
      style={{
        background: `${!!dropDownBox ? 'rgba(255, 255, 255, 0.03)' : ''}`
      }}
    >
      <div className={styles.transactionItem}>
        <div className={styles.title}>
          <div>{chain}</div>
          <div>{title}</div>
        </div>
        {(midTitle || status) && 
          <div className={styles.subTitle}>
            <div>{midTitle}</div>
            <div>{blockNumber}</div>
          </div>
        }
        {subTitle && 
          <div className={styles.subTitle}>
            {subTitle}
          </div>
        }
        <div className={styles.content}>
          <div>{typeTX}</div>
          {link && 
            <a 
              href={link}
              target={'_blank'}
              rel='noopener noreferrer'
              className={styles.button}
            > 
            <LinkIcon />
             Advanced Details</a>  
          }
        </div>
        {renderDetail()}
        {(button || status) &&
          <div className={styles.content}>
            <div className={styles.right}>
              {renderValue()}
            </div>
          </div>
        }
      </div>
      <div className={styles.divider}></div>

    </div>
  )
}

export default React.memo(Transaction);
