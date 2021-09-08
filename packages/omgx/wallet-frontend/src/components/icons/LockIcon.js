import * as React from "react";
import { useTheme } from "@material-ui/core/styles";

function LockIcon() {
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
            <rect x="11"
                y="14"
                width="9"
                height="7"
                rx="1"
                stroke={color}
                strokeWidth="1.5"
                strokeLinecap="round" />
            <path
                d="M13 12C13 10.8954 13.8954 10 15 10H16C17.1046 10 18 10.8954 18 12V14H13V12Z"
                stroke={color}
                strokeWidth="1.5"
                strokeLinecap="round" />
        </svg>
    );
}

export default LockIcon;
