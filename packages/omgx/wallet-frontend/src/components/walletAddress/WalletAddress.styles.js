import styled from '@emotion/styled';

export const WalletPill = styled.div`
  padding: 8px 20px;
  background-color: ${(props) => props.theme.palette.mode === 'light' ? props.theme.palette.divider : props.theme.palette.common.black};
  border-radius: 50px;
  display: inline-block;
`;
