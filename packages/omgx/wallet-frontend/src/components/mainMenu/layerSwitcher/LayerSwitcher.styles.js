import styled from '@emotion/styled';
import { Typography } from '@material-ui/core';

export const WalletPickerContainer = styled.div`
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: flex-start;
  margin-left: -5px;
  @include mobile {
    font-size: 0.9em;
    padding: 10px;
  }
`;

export const Label = styled(Typography)(({ theme }) => ({
  marginLeft: theme.spacing(1),
  color: theme.palette.text.disabled,
}));

export const WalletPickerWrapper = styled.div`
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  align-items: flex-start;
  width: 100%;
  @include mobile {
    flex-direction: column;
  }
  img {
    height: 20px;
  }
`;

export const Menu = styled.div`
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  z-index: 1;
  position: relative;
  @include mobile {
    width: 100%;
    justify-content: space-between;
  }
  a {
    cursor: pointer;
  }
`;

export const NetWorkStyle = styled.div`
  display: flex;
  flex-direction: row;
  align-items: center;
  cursor: ${(props) => props.walletEnabled !== false ? 'inherit' : 'pointer'};
`;

export const ButtonStyle = styled.div`
  display: flex;
  flex-direction: row;
  align-items: center;
  border-radius: 3px;
  background-color: blue;
  padding-top: 2px;
  padding-left: 5px;
  padding-right: 5px;
  font-size: 0.9em;
  font-weight: 600;
  cursor: ${(props) => props.walletEnabled !== false ? 'inherit' : 'pointer'};
`;

export const LayerSwitch = styled.div`
  margin-left: 15px;
  padding: 3px;
  border-radius: 100px;
  background: #3C5DFC;
  cursor: pointer;
  display: flex;
  height: 28px;
  span {
    padding: 2px 15px;
    font-weight: bold;
    border-radius: 100px;
    display: flex;
    justify-content: center;
    align-items: center;
    &.active {
      color: #3c5dfc;
      background: white;
    }
  }
`;
