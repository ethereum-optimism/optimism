
import React from 'react';
import * as S from './ThemeSwitcher.styles.js'
import DarkIcon from 'components/icons/DarkIcon.js';
import LightIcon from 'components/icons/LightIcon.js';
import { ReactComponent as ShadowMenu } from './../../../images/backgrounds/shadow-menu.svg';

function ThemeSwitcher ({ light, setLight }) {
  return (
    <S.ThemeSwitcherTag>
      <S.Button onClick={() => setLight(false)} selected={!light}>
        <DarkIcon />
      </S.Button>
      <S.Button onClick={() => setLight(true)} selected={light}>
        <LightIcon />
      </S.Button>
      <S.Shadow>
        <ShadowMenu height={250}/>
      </S.Shadow>
    </S.ThemeSwitcherTag>
  );
}

export default ThemeSwitcher;
