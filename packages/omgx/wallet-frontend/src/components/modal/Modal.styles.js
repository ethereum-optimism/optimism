import { Box, IconButton } from '@material-ui/core';
import { dark } from '@material-ui/core/styles/createPalette';
import { styled } from '@material-ui/system';
import ModalUnstyled from '@material-ui/unstyled/ModalUnstyled';

export const StyledModal = styled(ModalUnstyled)`
  position: fixed;
  z-index: 1300;
  right: 0;
  bottom: 0;
  top: 0;
  left: 0;
  z-index: 1300;
  display: flex;
  align-items: center;
  justify-content: center;
  backdrop-filter: ${(props) => props.ismobile ? 'blur(20px)' : 'none'};
`;

export const Backdrop = styled('div')`
  z-index: -1;
  position: fixed;
  right: 0;
  bottom: 0;
  top: 0;
  left: 0;
  background-color: rgba(8, 22, 44, 0.7);
  backdrop-filter: blur(10px);
  -webkit-tap-highlight-color: transparent;
`;

export const Style = styled(Box)`
  background: ${(props) => props.transparent ? props.theme.palette.background.modalTransparent : props.theme.palette.background.modal};
  backdrop-filter: ${(props) => props.transparent ? 'none' :'blur(66px)'};
  padding: ${(props) => props.transparent ? '20px' : '40px'};
  border: 0;
  outline: 0;
  box-sizing: border-box;
  max-width: 100%;
  border-radius: 12px;
`;

export const IconButtonTag = styled(IconButton)(({ theme }) => ({
  [theme.breakpoints.down('md')]: {
    position: 'absolute',
    top: 0,
    right: 20,
  },
  [theme.breakpoints.up('md')]: {
  },
}));

export const WrapperActionsModal = styled(Box)`
   display: flex;
   align-items: center;
   gap: 10px;
   justify-content: flex-end;
   margin-top: 50px;
 `;
