import * as React from "react"
import { useTheme } from "@material-ui/core/styles";

function SortIcon() {
    const theme = useTheme();
    const isLight = theme.palette.mode === 'light';
    const color = theme.palette.common[isLight ? 'black' : 'white'];
    return (
        <svg
            width="7"
            height="12"
            viewBox="0 0 7 12"
            fill="none"
            xmlns="http://www.w3.org/2000/svg">
            <path
                d="M3.5 0.5L7 4.5H0L3.5 0.5Z"
                fill={color}
                fillOpacity="0.7"
            />
            <path
                d="M3.5 11.5L7 7.5H0L3.5 11.5Z"
                fill={color}
                fillOpacity="0.7"
            />
        </svg>
    )

}

export default SortIcon
