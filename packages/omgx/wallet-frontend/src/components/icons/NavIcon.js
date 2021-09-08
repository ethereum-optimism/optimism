import * as React from "react"
import { useTheme } from "@material-ui/core/styles";

function NavIcon({onClick}) {
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
      <path
        stroke={color}
        strokeWidth={2}
        strokeLinecap="round"
        d="M10 11h12M10 16h12M10 21h7"
      />
    </svg>
  )
}

export default NavIcon
