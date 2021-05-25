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

import React from 'react';

import Tooltip from 'components/tooltip/Tooltip';

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
  tooltip = ''
}) {
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
      );
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
    );
  }

  const Resolved = link ? 'a' : 'div';
  return (
    <div className={styles.Transaction}>
      <Resolved
        href={link}
        target={'_blank'}
        rel='noopener noreferrer'
        className={styles.left}
      >
        <div>{title}</div>
        {midTitle && (
          <div className={styles.midTitle}>{midTitle}</div>
        )}
        <div>{subTitle}</div>
      </Resolved>
      <div className={styles.right}>
        {renderValue()}
      </div>
    </div>
  );
}

export default React.memo(Transaction);
