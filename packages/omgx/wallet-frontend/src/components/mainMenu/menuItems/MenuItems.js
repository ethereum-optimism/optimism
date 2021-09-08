import { useTheme } from '@emotion/react';
import React, { useState } from 'react';
import { menuItems } from '../menuItems';
import * as S from './MenuItems.styles';

import EarnIcon from 'components/icons/EarnIcon'
import WalletIcon from 'components/icons/WalletIcon'
import HistoryIcon from 'components/icons/HistoryIcon'
import NFTIcon from 'components/icons/NFTIcon'
import DAOIcon from 'components/icons/DAOIcon'

function MenuItems ({handleSetPage, pageDisplay, setOpen }) {

  const [ activeItem, setActiveItem ] = useState(false)
  const theme = useTheme()
  const isLight = theme.palette.mode === 'light'
  const colorIcon = theme.palette.common[isLight ? 'black' : 'white']

  const iconObj = {
    WalletIcon,
    EarnIcon,
    HistoryIcon,
    NFTIcon,
    DAOIcon
  }

  return (
    <S.Nav>
      <S.NavList>
        {menuItems.map((item) => {
          const Icon = iconObj[item.icon];
          const isActive = pageDisplay === item.key;
          const title = item.title;
          return (
            <li key={title}>
              <S.MenuItem
                onClick={() => {
                  handleSetPage(item.key)
                  setOpen(false)
                }}
                onMouseEnter={() => setActiveItem(title)}
                onMouseLeave={() => setActiveItem(false)}
                // to={item.url}
                selected={isActive}
              >
                <Icon 
                  color={isActive || activeItem === title ? theme.palette.secondary.main : colorIcon}
                  width={'20px'} 
                />
                  {item.title}
              </S.MenuItem>
            </li>
          )
        })}
      </S.NavList>
    </S.Nav>
  );
}

export default MenuItems
