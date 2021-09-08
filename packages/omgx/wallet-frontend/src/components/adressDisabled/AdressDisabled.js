import React from 'react'
import { styled } from '@material-ui/system';
import { Box } from '@material-ui/core'
import LockIcon from 'components/icons/LockIcon';

const AdressDisabledStyles= styled(Box)`
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16.5px 14px;
  border-radius: 0px 12px 12px 0px;
  background-image: linear-gradient(135deg, #121e30 12.50%, #0c192c 12.50%, #0c192c 50%, #121e30 50%, #121e30 62.50%, #0c192c 62.50%, #0c192c 100%);
  background-size: 28.28px 28.28px;
  opacity: 0.5;
`;

function AdressDisabled({ children }) {

  return (
    <AdressDisabledStyles>
      {children}
      <LockIcon />
    </AdressDisabledStyles>
  )
}

export default React.memo(AdressDisabled)
