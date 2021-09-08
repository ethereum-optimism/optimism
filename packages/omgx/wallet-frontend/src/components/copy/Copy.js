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
import { IconButton, Tooltip } from '@material-ui/core';
import { useEffect } from 'react';

function Copy ({ value }) {
  const [ open, setOpen ] = useState(false);

  useEffect(() => {
    if (open) {
      setTimeout(() => {
        setOpen(false);
      }, 1500);
    }
  }, [open, setOpen]);

  return (
    <CopyToClipboard
      text={value}
      onCopy={() => setOpen(true)}
    >
      <Tooltip open={open} title="Copied to clipboard!">
        <IconButton>
          <FileCopyOutlined sx={{ fontSize: 16 }} />
        </IconButton>
      </Tooltip>
    </CopyToClipboard>
  );
}

export default React.memo(Copy);
