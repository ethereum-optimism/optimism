import React from 'react';

import * as styles from './Hamburger.module.scss';

function Hamburger ({ hamburgerClick, isOpen, className }) {
  return (
    <div onClick={() => hamburgerClick()} className={className}>
      <div
        className={[
          styles.xline,
          styles.xline1,
          isOpen ? styles.open : styles.closed
        ].join(' ')}
      />
      <div
        className={[
          styles.xline,
          styles.xline2,
          isOpen ? styles.open : styles.closed
        ].join(' ')}
      />
      <div
        className={[
          styles.xline,
          styles.xline3,
          isOpen ? styles.open : styles.closed
        ].join(' ')}
      />
    </div>
  );
}

export default Hamburger;
