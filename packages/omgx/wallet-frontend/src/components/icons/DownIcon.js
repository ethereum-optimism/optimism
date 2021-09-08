import * as React from "react";
import { useTheme } from "@material-ui/core/styles";

function DownIcon() {
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
            <path
                d="M20.9999 14.9999L16 19.9998L11.0001 14.9999" 
                stroke={color}
                strokeWidth="1.5" 
                strokeLinecap="round"/>
        </svg>
        
    )
}

export default DownIcon;
