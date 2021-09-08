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

export const WallerPickerWrapper = styled.div`
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
  flex-direction: row;
  align-items: center;
  z-index: 1000;
  position: relative;
  @include mobile {
    width: 100%;
    justify-content: space-between;
  }
  a {
    cursor: pointer;
  }
`;

export const Chevron = styled.img`
  transform: ${props => props.open ? 'rotate(-90deg)' : 'rotate(90deg)'};
  transition: all 200ms ease-in-out;
  height: 20px;
  margin-bottom: 0;
`;

export const Dropdown = styled.div`
  display: flex;
  flex-direction: column;
  position: absolute;
  left: 0;
  top: 27px;
  background: #09162B;
  border-radius: 12px;
  padding: 15px;
  box-shadow: -13px 15px 39px rgba(0, 0, 0, 0.16), inset 53px 36px 120px rgba(255, 255, 255, 0.06);
  @include mobile {
    right: unset;
    left: 10px;
    width: 150px;
  }
  a {
    background-color: gray;
    transition: all 200ms ease-in-out;
    padding: 10px 15px;
    &:hover {
      background-color: gray;
    }
  }
  > p {
    cursor: pointer;
  }
`;

export const NetWorkStyle = styled.div`
  display: flex;
  flex-direction: row;
  align-items: center;
  cursor: ${(props) => props.walletEnabled !== false ? 'inherit' : 'pointer'};
`;
