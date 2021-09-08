import styled from '@emotion/styled';


export const Tabs = styled.div`
display: flex;
flex-direction: row;
justify-content: flex-start;
flex: 1;
margin-bottom: 20px;
`;

export const TabItem = styled.div`
opacity: 0.7;
transition: color 200ms ease-in-out;
cursor: pointer;
margin-right: 20px;
 &.active {
    color: ${props => props.theme.palette.mode === 'light' ? props.theme.palette.primary.main : props.theme.palette.neutral.main};
    opacity: 1;
    border-bottom: 2px solid ${props => props.theme.palette.neutral.contrastText};
    margin-bottom: -2px;
    z-index: 1;
 }
`


