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
import { Search } from '@material-ui/icons'
import BN from 'bignumber.js'

import * as styles from './Input.module.scss'

function Input({
  placeholder,
  label,
  type = 'text',
  disabled,
  icon,
  unit,
  value,
  onChange,
  paste,
  className,
  maxValue,
  small,
}) {
  async function handlePaste() {
    try {
      const text = await navigator.clipboard.readText()
      if (text) {
        onChange({ target: { value: text } })
      }
    } catch (err) {
      // navigator clipboard api not supported in client browser
    }
  }

  function handleMaxClick() {
    onChange({ target: { value: maxValue } })
  }

  const overMax = new BN(value).gt(new BN(maxValue))

  return (
    <div className={[styles.Input, className].join(' ')}>
      {label && <div className={styles.label}>{label}</div>}
      <div className={[styles.field, overMax ? styles.error : ''].join(' ')}>
        {icon && <Search className={styles.icon} />}
        {
          type === 'textArea' ?
            <textarea
              className={[styles.input, small ? styles.small : ''].join(' ')}
              placeholder={placeholder}
              value={value}
              onChange={onChange}
              disabled={disabled}
              rows="4"
              style={{
                height: 'auto'
              }}
            />
            : <input
              className={[styles.input, small ? styles.small : ''].join(' ')}
              placeholder={placeholder}
              type={type}
              value={value}
              onChange={onChange}
              disabled={disabled}
              style={{
                paddingLeft: `${icon ? '40px' : '5px'}`
              }}
            />
        }
        {unit && (
          <div className={`${styles.unit} ${!maxValue ? styles.isPaste : ''}`}>
            {maxValue && value !== maxValue && (
              <div onClick={handleMaxClick} className={styles.maxValue}>
                MAX
              </div>
            )}
            <div style={{display: 'flex', flexDirection: 'column'}}>
              {unit}
              <span style={{fontSize: '0.7em'}}>Balance: {Number(maxValue).toFixed(3)}</span>
            </div>
          </div>
        )}
        {paste && (
          <div onClick={handlePaste} className={styles.paste}>
            Paste
          </div>
        )}
      </div>
    </div>
  )
}

export default React.memo(Input)
