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
import { CircularProgress } from '@material-ui/core';
import * as styles from './Button.module.scss';

function Button ({
  children,
  style,
  onClick,
  type,
  disabled,
  loading,
  pulsate,
  tooltip = '',
  size,
  className,
  triggerTime,
}) {

  if(disabled || loading)
    pulsate = false;

  let timeDefined = false;

  if(typeof triggerTime !== 'undefined') {
    timeDefined = true;
  }

  // Save the current date to be able to trigger an update
  const [now, setTime] = React.useState(new Date()); 

  React.useEffect(() => {
    const timer = setInterval(()=>{setTime(new Date())}, 1000);
    return () => {clearInterval(timer)}
  }, []);

  let waitTime = (now-triggerTime) / 1000;
  if(waitTime < 0) waitTime = 0;
  waitTime = Math.round(waitTime);

  return (
    <div
      style={style}
      className={[
        styles.Button,
        type === 'primary' ? styles.primary : '',
        type === 'secondary' ? styles.secondary : '',
        type === 'outline' ? styles.outline : '',
        size === 'small' ? styles.small : '',
        size === 'tiny' ? styles.tiny : '',
        loading ? styles.disabled : '',
        disabled ? styles.disabled : '',
        pulsate ? styles.pulsate : '',
        className
      ].join(' ')}
      onClick={loading || disabled ? null : onClick}
    >
      {children}
      {(disabled || loading) && timeDefined && (waitTime > 3) &&
        <div style={{marginLeft: '10px'}}
        >
          {waitTime}s ago
        </div>
      }
      {loading && 
        <div style={{paddingTop: '4px', marginLeft: '10px'}}>
          <CircularProgress size={14} color='inherit' />
        </div>
      }
    </div>
  );
}

export default React.memo(Button);
