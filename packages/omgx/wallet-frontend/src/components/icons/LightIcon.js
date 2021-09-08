import * as React from "react";
import { useTheme } from "@material-ui/core/styles";

function LightIcon() {
  const theme = useTheme();
  const isLight = theme.palette.mode === 'light';
  const color = theme.palette.common[isLight ? 'black' : 'white'];
  return (
    <svg
      width={32}
      height={32}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <mask id="light-icon" fill="#fff">
        <path d="M21.5 16a5.5 5.5 0 11-11 0 5.5 5.5 0 0111 0z" />
      </mask>
      <path
        d="M20 16a4 4 0 01-4 4v3a7 7 0 007-7h-3zm-4 4a4 4 0 01-4-4H9a7 7 0 007 7v-3zm-4-4a4 4 0 014-4V9a7 7 0 00-7 7h3zm4-4a4 4 0 014 4h3a7 7 0 00-7-7v3z"
        fill={color}
        mask="url(#light-icon)"
      />
      <path
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        d="M16.25 5.25v2.5M23.778 8.576l-1.767 1.768M26.75 16.25h-2.5M23.425 23.777l-1.768-1.767M7.75 16.25h-2.5M9.99 10.344L8.222 8.576M16.25 24.25v2.5M10.343 22.01l-1.768 1.768"
      />
    </svg>
  )
}

export default LightIcon;
