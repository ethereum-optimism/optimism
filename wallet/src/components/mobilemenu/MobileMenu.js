import React from 'react';

import Status from 'containers/status/Status';

import * as styles from './MobileMenu.module.scss';

function MobileMenu ({ mobileMenuOpen }) {
  return (
    <Status
      className={[
        styles.menu,
        mobileMenuOpen ? styles.open : ''
      ].join(' ')}
    />
  );
}

export default MobileMenu;
