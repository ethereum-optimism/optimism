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
import { Search } from '@material-ui/icons';
import BN from 'bignumber.js';

import * as styles from './Input.module.scss';

function Input ({
  placeholder,
  label,
  type = 'text',
  icon,
  unit,
  value,
  onChange,
  paste,
  className,
  maxValue
}) {
  
  async function handlePaste () {
    try {
      const text = await navigator.clipboard.readText();
      if (text) {
        onChange({ target: { value: text } });
      }
    } catch (err) {
      // navigator clipboard api not supported in client browser
    }
  }

  function handleMaxClick () {
    onChange({ target: { value: maxValue } });
  }

  const overMax = new BN(value).gt(new BN(maxValue));

  return (
    <div className={[ styles.Input, className ].join(' ')}>
      {label && <div className={styles.label}>{label}</div>}
      <div
        className={[
          styles.field,
          overMax ? styles.error : ''
        ].join(' ')}
      >
        {icon && <Search className={styles.icon} />}
        <input
          className={styles.input}
          placeholder={placeholder}
          type={type}
          value={value}
          onChange={onChange}
        />
        {unit && (
          <div className={styles.unit}>
            {maxValue && (value !== maxValue) && (
              <div
                onClick={handleMaxClick}
                className={styles.maxValue}
              >
                MAX
              </div>
            )}
            {unit}
          </div>
        )}
        {paste && (
          <div onClick={handlePaste} className={styles.paste}>
            Paste
          </div>
        )}
      </div>
    </div>
  );
}

export default React.memo(Input);
