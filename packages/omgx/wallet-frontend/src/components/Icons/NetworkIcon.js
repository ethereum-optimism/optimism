import { useTheme } from "@emotion/react";
import * as React from "react"

function NetworkIcon() {
  const theme = useTheme();
  const isLight = theme.palette.mode === 'light';
  const color = isLight ? 'black' : 'white';

  return (
    <svg
      width={32}
      height={32}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M23.65 16c0 .701-.094 1.38-.27 2.025h-3.833a38.19 38.19 0 000-4.05h3.832c.177.645.271 1.324.271 2.025zm-5.456-2.025a36.389 36.389 0 010 4.05h-4.389a36.732 36.732 0 010-4.05h4.39zm1.254-1.35h3.42a7.678 7.678 0 00-4.005-3.721c.259 1.102.459 2.36.585 3.721zm-2.09-4.155c.325 1.158.581 2.572.734 4.155h-4.184c.153-1.583.409-2.997.735-4.155a7.696 7.696 0 012.714 0zm-4.22.434a7.678 7.678 0 00-4.005 3.721h3.419a27.51 27.51 0 01.585-3.721zm-.684 5.071H8.62A7.658 7.658 0 008.35 16c0 .701.094 1.38.27 2.025h3.833a38.19 38.19 0 010-4.05zm.096 5.4H9.133a7.678 7.678 0 003.994 3.717 27.755 27.755 0 01-.577-3.717zm2.08 4.153a7.688 7.688 0 002.74 0c.32-1.154.573-2.572.723-4.153h-4.186c.15 1.58.402 2.999.723 4.153zm4.82-4.153a27.755 27.755 0 01-.576 3.717 7.678 7.678 0 003.993-3.717H19.45zM25 16a9 9 0 11-18 0 9 9 0 0118 0z"
        fill={color}
        fillOpacity="0.5"
      />
    </svg>
  )
}

export default NetworkIcon;
