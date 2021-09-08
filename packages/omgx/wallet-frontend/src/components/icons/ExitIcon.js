import * as React from "react";
import { useTheme } from "@material-ui/core/styles";

function ExitIcon() {
    const theme = useTheme();
    const isLight = theme.palette.mode === 'light';
    const color = theme.palette.common[isLight ? 'black' : 'white'];
    return (
        <svg width="32"
            height="32"
            viewBox="0 0 32 32"
            fill="none"
            xmlns="http://www.w3.org/2000/svg">
            <path d="M16.0003 16L12.4647 12.4645M16.0003 16L19.5358 12.4645M16.0003 16L19.5358 19.5355M16.0003 16L12.4647 19.5355"
                stroke={color}
                strokeWidth="1.5"
                strokeLinecap="round"
            />
        </svg>

    );
}

export default ExitIcon;
