import * as React from "react"
import { useTheme } from "@material-ui/core/styles";

function CloseIcon() {
  const theme = useTheme();
  const isLight = theme.palette.mode === 'light';
  const color = theme.palette.common[isLight ? '#ccc' : '#201D31'];
  return (
    <svg
      width={40}
      height={40}
      viewBox="0 0 40 40"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <rect width={40} height={40} rx={20} fill={color} fillOpacity={0.8} />
      <path
        d="M20 20l-4.419-4.42m4.42 4.42l4.419-4.42M20 20l4.42 4.42M20 20l-4.419 4.42"
        stroke="#fff"
        strokeOpacity={0.7}
        strokeWidth={1.5}
        strokeLinecap="round"
      />
    </svg>
  )
}

export default CloseIcon
