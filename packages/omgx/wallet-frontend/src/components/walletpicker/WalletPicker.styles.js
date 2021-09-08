import styled from '@emotion/styled';
import { Box, Typography } from '@material-ui/core';
import BgWallet from "../../images/backgrounds/bg-wallet.png";

export const Loading = styled.div`
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  font-size: 20px;
  color: $gray4;
`;

export const Subtitle = styled(Typography)(({ theme }) => ({
  opacity: 0.7,
  [theme.breakpoints.down('md')]: {
    margin: "16px 40px 16px 0",
  },
  [theme.breakpoints.up('md')]: {
    margin: "32px 80px 32px 0",
  },
}));

export const WalletCard = styled.div`

  border-top-left-radius: 4px;
  border-top-right-radius: 16px;
  border-bottom-left-radius: 16px;
  border-bottom-right-radius: 4px;
  padding: 40px;
  background: url(${BgWallet});
  background-repeat: no-repeat;
  background-size: cover;
  box-shadow: inset 0 0 0 2px rgba(255, 255, 255, 0.2);
  cursor: pointer;
  margin-top: ${(props) => props.isMobile ? "-30px" : "0"};
  margin-bottom: ${(props) => props.isMobile ? "20px" : "30px"};
  display: flex;
`;

export const WalletCardHeading = styled(Box)(({ theme }) => ({
  display: "flex",
  flexDirection: "column",
  [theme.breakpoints.down('md')]: {
    gap: "10px"
  },
  [theme.breakpoints.up('md')]: {
    gap: "50px"
  },
}));

export const WalletCardTitle = styled(Box)(({ theme }) => ({
  display: "flex",
  gap: "10px",
  [theme.breakpoints.down('md')]: {
    flexDirection: "column",
    gap: "50px"
  },
  [theme.breakpoints.up('md')]: {
    alignItems: "center",
  },
}));

export const WalletCardDescription = styled(Box)(({ theme }) => ({
  display: "flex",
  marginLeft: "auto",
  [theme.breakpoints.down('md')]: {
    alignItems: "flex-start",
  },
  [theme.breakpoints.up('md')]: {
    alignItems: "flex-end",
  },
}));

export const PlusIcon = styled.div`
    background-color: #091426;
    border-radius: 50%;
    font-size: 20px;
    font-weight: 700;
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
`;

export const WrapperLink = styled(Box)(({ theme }) => ({
  alignItems: "center",
  [theme.breakpoints.down('md')]: {
    marginTop: "20px",
  },
  [theme.breakpoints.up('md')]: {
    marginTop: "20px",
  },
}));
