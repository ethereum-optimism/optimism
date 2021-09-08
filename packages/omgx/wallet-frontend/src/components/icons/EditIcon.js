import * as React from "react";
import { useTheme } from "@material-ui/core/styles";

function EditIcon() {
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
            <path d="M13.5205 15.8203L19.6339 9.70696C20.0244 9.31644 20.6575 9.31644 21.0481 9.70696L22.2154 10.8743C22.6059 11.2648 22.6059 11.898 22.2154 12.2885L16.1021 18.4019L13.5205 15.8203Z"
                stroke={color}
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round" />
            <path d="M11.9768 17.9177L14.1535 20.0944L10.0884 21.9829L11.9768 17.9177Z"
                stroke={color}
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round" />
        </svg>
    );
}

export default EditIcon;
