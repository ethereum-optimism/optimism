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
import Select from 'react-select'
import etherIcon from '../../images/ethereum.svg'
import testIcon from '../../images/test.svg'
import omgxIcon from '../../images/omg-icon-circle.png'
import sushiIcon from '../../images/sushi-icon.png'
import usdtIcon from '../../images/usdt-icon.png'
import * as styles from './iconSelect.module.scss'

const tokenIcons = {
  omgxIcon: omgxIcon,
  sushiIcon: sushiIcon,
  etherIcon: etherIcon,
  usdtIcon: usdtIcon,
  testIcon: testIcon,
}

/**
 * 📓  Option should have property called balance
 *    - in case of deposit it will hold the value as balanceL1
 *    - in case of exit it will hold the value as balanceL2
 */

function IconSelect({
  onTokenSelect,
  priorityOptions = [],
  dropdownOptions = [],
  disableDD = false,
}) {
  /* These are the Token Icons */
  const tokenPicker = (
    <div className={styles.tokenPicker}>
      {priorityOptions.map((t) => {
        return (
          <div
            key={t.symbol}
            className={styles.tokenIcon}
            onClick={()=>{onTokenSelect(t)}}
          >
            <img 
              src={tokenIcons[t.icon]} 
              width='40px' 
              alt={t.title} 
            />
            <p className={styles.tokenSymbol}>{t.symbol}</p>
            {Number(t.balance) ? (
              <p className={styles.tokenIconBalance}>
                {Number(t.balance).toFixed(2)}
              </p>
            ) : null}
          </div>
        )
      })}
    </div>
  )

  const formatOptionLabel = (t) => (
    <div className={styles.ddOptionLabel}>
      <span>{t.label}</span>
      <div className={styles.optionBalance}>
        {Number(t.balance) ? (
          <span className={styles.tokenIconBalance}>
            {Number(t.balance).toFixed(2)}
          </span>
        ) : null}
      </div>
    </div>
  )

  const dropdownTokenPicker = (
    <>
      {!disableDD && dropdownOptions && (
        <div className={styles.selectContainer}>
          <Select
            formatOptionLabel={formatOptionLabel}
            options={[
              ...dropdownOptions,
              {
                name: 'Manual',
                label: 'Manual',
                value: 'Manual',
              },
            ].map((i) => {
              return {
                label: i.name,
                value: i.name,
                ...i,
              }
            })}
            onChange={(token)=>{onTokenSelect(token)}}
          />
        </div>
      )}
    </>
  )

  return (
    <div className={styles.iconSelectContainer}>
      {tokenPicker}
      {dropdownTokenPicker}
    </div>
  )
}

export default React.memo(IconSelect)
