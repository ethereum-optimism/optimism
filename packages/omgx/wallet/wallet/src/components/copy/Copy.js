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

import React, { useState } from 'react';
import { CopyToClipboard } from 'react-copy-to-clipboard';
import { FileCopyOutlined } from '@material-ui/icons';

import Alert from 'components/alert/Alert';

import * as styles from './Copy.module.scss';

function Copy ({ value, light }) {
  const [ open, setOpen ] = useState(false);

  return (
    <div className={styles.Copy}>
      <CopyToClipboard
        text={value}
        onCopy={() => setOpen(true)}
      >
        <div
          className={[
            styles.icon,
            light ? styles.light : ''
          ].join(' ')}>
          <FileCopyOutlined />
        </div>
      </CopyToClipboard>
      <Alert open={open} onClose={() => setOpen(false)}>
        Copied to clipboard!
      </Alert>
    </div>
  );
}

export default React.memo(Copy);
