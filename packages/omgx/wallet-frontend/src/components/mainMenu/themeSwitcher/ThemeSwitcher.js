import React from 'react';
import * as S from './ThemeSwitcher.styles.js'
import DarkIcon from 'components/icons/DarkIcon.js';
import LightIcon from 'components/icons/LightIcon.js';
import { ReactComponent as ShadowMenu } from './../../../images/backgrounds/shadow-menu.svg';
import { setTheme } from 'actions/uiAction.js';
import { useSelector } from 'react-redux';
import { selectModalState } from 'selectors/uiSelector.js';
import { useDispatch } from 'react-redux';

function ThemeSwitcher () {
  const theme = useSelector(selectModalState('theme'));
  const dispatch = useDispatch();
  return (
    <S.ThemeSwitcherTag>
      <S.Button onClick={() => {
        localStorage.setItem('theme', 'dark');
        dispatch(setTheme('dark'));
      }} selected={theme === 'dark'}>
        <DarkIcon />
      </S.Button>
      <S.Button onClick={() => {
        localStorage.setItem('theme', 'light');
        dispatch(setTheme('light'));
      }} selected={theme === 'light'}>
        <LightIcon />
      </S.Button>
      <S.Shadow>
        <ShadowMenu height={250}/>
      </S.Shadow>
    </S.ThemeSwitcherTag>
  );
}

export default ThemeSwitcher;
