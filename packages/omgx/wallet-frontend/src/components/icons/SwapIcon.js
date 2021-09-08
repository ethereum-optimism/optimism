import { useTheme } from "@emotion/react";
import * as React from "react"

function SwapIcon() {
  const theme = useTheme();
  const isLight = theme.palette.mode === 'light';
  const color = theme.palette.common[isLight ? 'black' : 'white'];

  return (
    <svg
      width={22}
      height={22}
      viewBox="0 0 22 22"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M4.019 15.293l-.508-.552-.601.552.601.553.508-.553zm15.037-.75H4.019v1.5h15.037v-1.5zm-14.53 1.303l3.509-3.222-1.015-1.105-3.509 3.222 1.015 1.105zm-1.015 0l3.509 3.222 1.015-1.105-3.51-3.222-1.014 1.105zM17.981 6.704l.508-.553.601.553-.601.552-.508-.552zm-15.037-.75h15.037v1.5H2.944v-1.5zm14.53 1.302l-3.509-3.222 1.015-1.105 3.509 3.222-1.015 1.105zm1.015 0l-3.509 3.222-1.015-1.105 3.51-3.222 1.014 1.105z"
        fill={color}
      />
    </svg>
  )
}

export default SwapIcon
