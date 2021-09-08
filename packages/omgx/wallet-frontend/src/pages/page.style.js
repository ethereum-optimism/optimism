import styled from '@emotion/styled';

export const PageContent = styled.div`
    width: 90%;
    margin: 0px auto;
    height: 100vh;
    overflow: auto;
`;

export const PageTitle = styled.h1`
    font-weight: bold;
    font-size: 62px;
    line-height: 62px;
    margin: 0px
`;


// TODO:  Move me to the table specific folders.

export const Th = styled.div`
    font-weight: normal;
    font-size: 18px;
    line-height: 18px;
    color: rgba(255, 255, 255, 0.7);
    flex: none;
    order: 0;
    flex-grow: 0;
    margin: 0px 6px;
    svg {
        margin-left: 5px;
    }
`

// TODO: move to the common style file.
export const Chevron = styled.img`
  transform: ${props => props.open ? 'rotate(-90deg)' : 'rotate(90deg)'};
  transition: all 200ms ease-in-out;
  height: 20px;
  margin-bottom: 0;
`;
