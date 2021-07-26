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
import { KeyboardArrowDown } from '@material-ui/icons'
import Input from 'components/input/Input'

import * as styles from './InputSelect.module.scss'

function InputSelect({
  placeholder,
  label,
  value,
  onChange,
  selectOptions,
  onSelect,
  selectValue,
  maxValue,
  disabledSelect = false,
  type = 'number',
  paste = false,
}) {
  const selected = selectOptions.find((i) => i.value === selectValue)

  const renderUnit = (
    <div className={styles.selectContainer}>
      <select
        className={styles.select}
        value={selectValue}
        onChange={onSelect}
        disabled={disabledSelect}
      >
        {selectOptions.map((i, index) => (
          <option key={index} value={i.value}>
            {i.title} - {i.subTitle}
          </option>
        ))}
      </select>
      <div className={styles.selected}>
        <div className={styles.details}>
          <div className={styles.title}>{selected ? selected.title : ''}</div>
          <div className={styles.subTitle}>
            {selected ? selected.subTitle : ''}
          </div>
        </div>
        {disabledSelect ? <></> : <KeyboardArrowDown />}
      </div>
    </div>
  )

  return (
    <Input
      placeholder={placeholder}
      label={label}
      type={type}
      unit={renderUnit}
      maxValue={maxValue}
      value={value}
      onChange={onChange}
      paste={paste}
    />
  )
}

export default React.memo(InputSelect)
