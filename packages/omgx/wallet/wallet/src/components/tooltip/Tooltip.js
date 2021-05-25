import React from 'react';
import { Tooltip as MuiTooltip } from '@material-ui/core';

import * as styles from './Tooltip.module.scss';

function Tooltip ({
  title,
  arrow = true,
  children
}) {
  return (
    <MuiTooltip
      title={
        <div className={styles.title}>
          {title}
        </div>
      }
      arrow={arrow}
    >
      {children}
    </MuiTooltip>
  );
}

export default React.memo(Tooltip);
