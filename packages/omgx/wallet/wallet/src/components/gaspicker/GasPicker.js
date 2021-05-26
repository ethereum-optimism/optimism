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

import React, { useEffect } from 'react';
import { useSelector } from 'react-redux';

import { selectGas } from 'selectors/gasSelector';

import * as styles from './GasPicker.module.scss';

function GasPicker ({ selectedSpeed, setSelectedSpeed, setGasPrice }) {
  const gas = useSelector(selectGas);

  useEffect(() => {
    setGasPrice(gas[selectedSpeed]);
  }, [ selectedSpeed, gas, setGasPrice ]);

  return (
    <div className={styles.GasPicker}>
      <div className={styles.label}>
        Gas Fee
      </div>

      <div className={styles.items}>
        <div
          onClick={() => setSelectedSpeed('slow')}
          className={[
            styles.category,
            selectedSpeed === 'slow' ? styles.selected : ''
          ].join(' ')}
        >
          <div className={styles.title}>Slow</div>
          <div>{gas.slow / 1000000000} gwei</div>
        </div>

        <div
          onClick={() => setSelectedSpeed('normal')}
          className={[
            styles.category,
            selectedSpeed === 'normal' ? styles.selected : ''
          ].join(' ')}
        >
          <div className={styles.title}>Normal</div>
          <div>{gas.normal / 1000000000} gwei</div>
        </div>

        <div
          onClick={() => setSelectedSpeed('fast')}
          className={[
            styles.category,
            selectedSpeed === 'fast' ? styles.selected : ''
          ].join(' ')}
        >
          <div className={styles.title}>Fast</div>
          <div>{gas.fast / 1000000000} gwei</div>
        </div>
      </div>
    </div>
  );
}

export default React.memo(GasPicker);
