import styled from '@emotion/styled';
import { Box } from '@material-ui/core';

export const Menu = styled.div`
  flex: 0 0 320px;
  width: 260px;
  padding-top: 40px;
  padding-left: 40px;
  ul {
    list-style-type: none;
    margin: 0;
    padding: 0;
    margin-left: -40px;
  }

  > a {
    margin-bottom: 50px;
    display: flex;
  }
`
export const MobileNavTag = styled(Box)`
  width: 100%;
  padding: 20px 0 40px 0;
  display: flex;
  gap: 20px;
  align-items: center;
  justify-content: space-between;
`;

export const StyleDrawer = styled(Box)`
  background-color: ${(props) => props.theme.palette.mode === 'light' ? 'white' : '#061122' };
  height: 100%;
`;

export const DrawerHeader = styled(Box)`
  display: flex;
  flex-direction: column;
  gap: 30px;
  padding: 40px 40px 20px 40px;
`;

export const WrapperCloseIcon = styled(Box)`
  display: flex;
  justify-content: space-between;
  align-items: center;
`;
