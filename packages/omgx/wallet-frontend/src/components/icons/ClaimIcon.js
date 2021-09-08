import * as React from "react";
import { useTheme } from "@material-ui/core/styles";

function ClaimIcon() {
    const theme = useTheme();
    const isLight = theme.palette.mode === 'light';
    const color = theme.palette.common[isLight ? 'black' : 'white'];
    return (
        <svg
            width="32"
            height="32"
            viewBox="0 0 32 32"
            fill="none"
            xmlns="http://www.w3.org/2000/svg">
            <line x1="11.1206"
                y1="11"
                x2="21.1206"
                y2="11"
                stroke={color}
                strokeWidth="2"
                strokeLinecap="round" />
            <line x1="13.1206"
                y1="16"
                x2="19.1206"
                y2="16"
                stroke={color}
                strokeWidth="2"
                strokeLinecap="round" />
            <line x1="15.1206"
                y1="21"
                x2="17.1206"
                y2="21"
                stroke={color}
                strokeWidth="2"
                strokeLinecap="round" />
        </svg>
    );
}

export default ClaimIcon;
