import React from 'react';
import { Box } from '@material-ui/core';
import networkService from 'services/networkService';
import truncate from 'truncate-middle';
import * as S from './WalletAddress.styles';
import { ReactComponent as FoxIcon } from './../../images/icons/fox-icon.svg';
import Copy from 'components/copy/Copy';

const WalletAddress = () => {
  const wAddress = networkService.account ? truncate(networkService.account, 6, 4, '...') : '';
  return (
    <S.WalletPill>
      <Box sx={{ display: 'flex', alignItems: 'center' }}>
        <FoxIcon width={21} height={19} />
        <Box sx={{ mx: 1.5 }}>{wAddress}</Box>
        <Copy value={networkService.account} light={false} />
      </Box>
    </S.WalletPill>
  )
};

export default WalletAddress;
