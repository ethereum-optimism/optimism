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
import { NavigateNext, NavigateBefore } from '@material-ui/icons';

import * as styles from './Pager.module.scss';
import * as S from './Pager.styles';
import { useTheme } from '@emotion/react';

function Pager ({ currentPage, totalPages, isLastPage, onClickNext, onClickBack, label }) {
  const theme = useTheme();
  return (
    <S.PagerContainer>
      <div className={styles.numberLeft}>{label}</div>
      <S.PagerContent>
        <S.PagerLabel>
        {`Page ${currentPage} of ${totalPages}`}
        </S.PagerLabel>

        <S.PagerNavigation
          variant={theme.palette.mode === "light" ? "contained" : "outlined"}
          size="small"
          color='primary'
          disabled={currentPage === 1}
          onClick={onClickBack}
        >
          <NavigateBefore />
        </S.PagerNavigation>

        <S.PagerNavigation
          variant={theme.palette.mode === "light" ? "contained" : "outlined"}
          size="small"
          color='primary'
          disabled={isLastPage}
          onClick={onClickNext}
        >
          <NavigateNext />
        </S.PagerNavigation>

      </S.PagerContent>
    </S.PagerContainer>
  )

  // return (
  //   <div className={styles.Pager}>
  //     <div className={styles.numberLeft}>{label}</div>
  //     <div className={styles.numberRight}>
  //       <div className={styles.number}>{`Page ${currentPage} of ${totalPages}`}</div>
  //       <div
  //         className={[
  //           styles.box,
  //           currentPage === 1 ? styles.disabled : ''
  //         ].join(' ')}
  //         onClick={onClickBack}
  //       >
  //         <NavigateBefore className={styles.icon} />
  //       </div>
  //       <div
  //         className={[
  //           styles.box,
  //           isLastPage ? styles.disabled : ''
  //         ].join(' ')}
  //         onClick={onClickNext}
  //       >
  //         <NavigateNext className={styles.icon} />
  //       </div>
  //       </div>
  //   </div>
  // );
}

export default Pager;
